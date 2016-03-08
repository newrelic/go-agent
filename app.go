package newrelic

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"go.datanerd.us/p/will/newrelic/internal"
	"go.datanerd.us/p/will/newrelic/log"
	"go.datanerd.us/p/will/newrelic/version"
)

func NewApplication(c Config) (Application, error) {
	return newApp(c)
}

type Application interface {
	// RecordCustomEvent adds a custom event to the application.  Each
	// application holds and reports up to 10*1000 custom events per minute.
	// Once this limit is reached, sampling will occur.  This feature is
	// incompatible with high security mode.
	//
	// eventType must consist of alphanumeric characters, underscores, and
	// colons, and must contain fewer than 255 bytes.
	//
	// Each value in the params map must be a number, string, or boolean.
	// Keys must be less than 255 bytes.  The params map may not contain
	// more than 64 attributes.  For more information, and a set of
	// restricted keywords, see:
	//
	// https://docs.newrelic.com/docs/insights/new-relic-insights/adding-querying-data/inserting-custom-events-new-relic-apm-agents
	RecordCustomEvent(eventType string, params map[string]interface{}) error

	// StartTransaction begins a Transaction.  The Transaction can always be
	// used safely, as nil will never be returned.
	StartTransaction(name string, w http.ResponseWriter, r *http.Request) Transaction
}

// appRun contains information regarding a single connection session with the
// collector.  It is created upon application connect and is afterwards
// immutable.
type appRun struct {
	*internal.ConnectReply
	collector string
}

type appData struct {
	id   internal.AgentRunID
	data internal.Harvestable
}

type App struct {
	config       Config
	connectJSON  []byte
	client       *http.Client
	testConsumer internal.DataConsumer

	harvestTicker      *time.Ticker
	harvestChan        <-chan time.Time
	dataChan           chan appData
	collectorErrorChan chan error
	connectChan        chan *appRun

	// run is non-nil when the app is successfully connected.  It is
	// immutable.  It is assigned by the processor goroutine and accessed by
	// goroutines calling app API methods.  It should be accessed using
	// getRun and setRun.
	run *appRun
	sync.RWMutex
}

func (app *App) String() string {
	return app.config.AppName
}

var (
	placeholderRun = &appRun{
		ConnectReply: internal.ConnectReplyDefaults(),
	}
)

func isFatalHarvestError(e error) bool {
	return internal.IsDisconnect(e) ||
		internal.IsLicenseException(e) ||
		internal.IsRestartException(e)
}

func shouldSaveFailedHarvest(e error) bool {
	if e == internal.ErrPayloadTooLarge || e == internal.ErrUnsupportedMedia {
		return false
	}
	return true
}

func (app *App) doHarvest(h *internal.Harvest, harvestStart time.Time, run *appRun) {
	h.CreateFinalMetrics()
	h.ApplyMetricRules(run.MetricRules)

	payloads := h.Payloads()
	for cmd, p := range payloads {

		data, err := p.Data(run.RunID.String(), harvestStart)

		if nil == data && nil == err {
			continue
		}

		if nil == err {
			call := internal.Cmd{
				UseSSL:    app.config.UseSSL,
				Collector: run.collector,
				License:   app.config.License,
				RunID:     run.RunID.String(),
				Name:      cmd,
				Data:      data,
			}

			// The reply from harvest calls is always unused.
			_, err = internal.CollectorRequest(call, app.client)
		}

		if nil == err {
			continue
		}

		if isFatalHarvestError(err) {
			app.collectorErrorChan <- err
			return
		}

		log.Warn("harvest failure", log.Context{
			"cmd":   cmd,
			"error": err.Error(),
		})

		if shouldSaveFailedHarvest(err) {
			app.Consume(run.RunID, p)
		}
	}
}

func (app *App) connectRoutine() {
	for {
		collector, reply, err := internal.ConnectAttempt(
			internal.ConnectAttemptArgs{
				UseSSL:            app.config.UseSSL,
				RedirectCollector: app.config.Collector,
				License:           app.config.License,
				ConnectJSON:       app.connectJSON,
				Client:            app.client,
			})

		if nil == err {
			app.connectChan <- &appRun{reply, collector}
			return
		}

		if internal.IsDisconnect(err) || internal.IsLicenseException(err) {
			app.collectorErrorChan <- err
			return
		}

		log.Warn("application connect failure", log.Context{"error": err.Error()})

		time.Sleep(internal.ConnectBackoff)
	}
}

var goAgentDebug = os.Getenv("GOAGENTDEBUG")

func debug(data internal.Harvestable) {
	now := time.Now()
	h := internal.NewHarvest(now)
	data.MergeIntoHarvest(h)
	ps := h.Payloads()
	for cmd, p := range ps {
		d, err := p.Data("agent run id", now)
		if nil == d && nil == err {
			continue
		}
		if nil != err {
			d = []byte(err.Error())
		}
		log.Debug("integration", log.Context{
			"cmd":  cmd,
			"data": string(d),
		})
	}
}

func (app *App) process() {
	var harvest *internal.Harvest

	cn := log.Context{
		"app":     app.String(),
		"license": app.config.License,
	}

	for {
		select {
		case <-app.harvestChan:
			run := app.getRun()
			if "" != run.RunID && nil != harvest {
				now := time.Now()
				go app.doHarvest(harvest, now, run)
				harvest = internal.NewHarvest(now)
			}
		case d := <-app.dataChan:
			if "" != goAgentDebug {
				debug(d.data)
			}

			run := app.getRun()
			if "" != d.id && nil != harvest && run.RunID == d.id {
				d.data.MergeIntoHarvest(harvest)
			}

		case err := <-app.collectorErrorChan:
			harvest = nil
			app.setRun(nil)

			switch {
			case internal.IsDisconnect(err):
				log.Info("application disconnected", cn)
			case internal.IsLicenseException(err):
				log.Error("invalid license", cn)
			case internal.IsRestartException(err):
				log.Info("application restarted", cn)
				go app.connectRoutine()
			}
		case r := <-app.connectChan:
			harvest = internal.NewHarvest(time.Now())
			app.setRun(r)
			log.Info("application connected", cn,
				log.Context{"run": r.RunID})
		}
	}
}

func newApp(c Config) (*App, error) {
	if err := c.Validate(); nil != err {
		return nil, err
	}

	// NOTE: If this is changed and the connect JSON is created afresh
	// before each connect (to get recent utilization info), the contents of
	// the config labels map should be copied to prevent data races.
	connectJSON, err := configConnectJSON(&c)
	if nil != err {
		return nil, fmt.Errorf("unable to create config connect JSON: %s", err)
	}

	app := &App{
		config:             c,
		connectJSON:        connectJSON,
		connectChan:        make(chan *appRun),
		collectorErrorChan: make(chan error),
		dataChan:           make(chan appData, internal.AppDataChanSize),
		client: &http.Client{
			Transport: c.Transport,
			Timeout:   internal.CollectorTimeout,
		},
	}

	log.Info("application created", log.Context{
		"app":     app.String(),
		"version": version.Version,
	})

	if app.config.Development {
		return app, nil
	}

	app.harvestTicker = time.NewTicker(internal.HarvestPeriod)
	app.harvestChan = app.harvestTicker.C

	go app.process()
	go app.connectRoutine()

	return app, nil
}

func (app *App) getRun() *appRun {
	app.RLock()
	defer app.RUnlock()

	if nil == app.run {
		return placeholderRun
	}
	return app.run
}

func (app *App) setRun(run *appRun) {
	app.Lock()
	defer app.Unlock()

	app.run = run
}

func (app *App) StartTransaction(name string, w http.ResponseWriter, r *http.Request) Transaction {
	run := app.getRun()
	return internal.NewTxn(internal.TxnInput{
		Config: internal.TxnConfig{
			TransactionEventsEnabled:    app.config.TransactionEvents.Enabled,
			ErrorCollectorEnabled:       app.config.ErrorCollector.Enabled,
			ErrorCollectorCaptureEvents: app.config.ErrorCollector.CaptureEvents,
			HighSecurity:                app.config.HighSecurity,
		},
		Reply:    run.ConnectReply,
		Request:  r,
		Writer:   w,
		Consumer: app,
	}, name)
}

var (
	highSecurityEnabledError        = fmt.Errorf("high security enabled")
	customEventsDisabledError       = fmt.Errorf("custom events disabled")
	customEventsRemoteDisabledError = fmt.Errorf("custom events disabled by server")
)

func (app *App) RecordCustomEvent(eventType string, params map[string]interface{}) error {
	if app.config.HighSecurity {
		return highSecurityEnabledError
	}

	if !app.config.CustomEvents.Enabled {
		return customEventsDisabledError
	}

	event, e := internal.CreateCustomEvent(eventType, params, time.Now())
	if nil != e {
		return e
	}

	run := app.getRun()
	if !run.CollectCustomEvents {
		return customEventsRemoteDisabledError
	}

	app.Consume(run.RunID, event)

	return nil
}

func (app *App) Consume(id internal.AgentRunID, data internal.Harvestable) {
	if nil != app.testConsumer {
		app.testConsumer.Consume(id, data)
		return
	}

	if "" == id {
		return
	}
	// TODO: Perhaps do not block if the channel is full.
	app.dataChan <- appData{id, data}
}
