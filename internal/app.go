package internal

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/newrelic/go-sdk/api"
	"github.com/newrelic/go-sdk/log"
	"github.com/newrelic/go-sdk/version"
)

// AppRun contains information regarding a single connection session with the
// collector.  It is created upon application connect and is afterwards
// immutable.
type AppRun struct {
	*ConnectReply
	collector string
}

type appData struct {
	id   AgentRunID
	data Harvestable
}

// App is the implementation of api.Application.
type App struct {
	config       api.Config
	client       *http.Client
	TestConsumer DataConsumer

	harvestTicker      *time.Ticker
	harvestChan        <-chan time.Time
	dataChan           chan appData
	collectorErrorChan chan error
	connectChan        chan *AppRun

	// run is non-nil when the app is successfully connected.  It is
	// immutable.  It is assigned by the processor goroutine and accessed by
	// goroutines calling app API methods.  It should be accessed using
	// getRun and SetRun.
	run *AppRun
	sync.RWMutex
}

func (app *App) String() string {
	return app.config.AppName
}

var (
	placeholderRun = &AppRun{
		ConnectReply: ConnectReplyDefaults(),
	}
)

func isFatalHarvestError(e error) bool {
	return isDisconnect(e) ||
		isLicenseException(e) ||
		isRestartException(e)
}

func shouldSaveFailedHarvest(e error) bool {
	if e == ErrPayloadTooLarge || e == ErrUnsupportedMedia {
		return false
	}
	return true
}

func (app *App) doHarvest(h *Harvest, harvestStart time.Time, run *AppRun) {
	h.createFinalMetrics()
	h.applyMetricRules(run.MetricRules)

	payloads := h.payloads()
	for cmd, p := range payloads {

		data, err := p.Data(run.RunID.String(), harvestStart)

		if nil == data && nil == err {
			continue
		}

		if nil == err {
			call := Cmd{
				UseTLS:    app.config.UseTLS,
				Collector: run.collector,
				License:   app.config.License,
				RunID:     run.RunID.String(),
				Name:      cmd,
				Data:      data,
			}

			// The reply from harvest calls is always unused.
			_, err = collectorRequest(call, app.client)
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
		collector, reply, err := connectAttempt(&app.config, app.client)
		if nil == err {
			app.connectChan <- &AppRun{reply, collector}
			return
		}

		if isDisconnect(err) || isLicenseException(err) {
			app.collectorErrorChan <- err
			return
		}

		log.Warn("application connect failure", log.Context{
			"error": err.Error(),
		})

		time.Sleep(connectBackoff)
	}
}

func debug(data Harvestable) {
	now := time.Now()
	h := NewHarvest(now)
	data.MergeIntoHarvest(h)
	ps := h.payloads()
	for cmd, p := range ps {
		d, err := p.Data("agent run id", now)
		if nil == d && nil == err {
			continue
		}
		if nil != err {
			log.Debug("integration", log.Context{
				"cmd":   cmd,
				"error": err.Error(),
			})
			continue
		}
		log.Debug("integration", log.Context{
			"cmd":  cmd,
			"data": JSONString(d),
		})
	}
}

func (app *App) process() {
	var harvest *Harvest

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
				harvest = NewHarvest(now)
			}
		case d := <-app.dataChan:
			if "" != debugLogging {
				debug(d.data)
			}

			run := app.getRun()
			if "" != d.id && nil != harvest && run.RunID == d.id {
				d.data.MergeIntoHarvest(harvest)
			}

		case err := <-app.collectorErrorChan:
			harvest = nil
			app.SetRun(nil)

			switch {
			case isDisconnect(err):
				log.Info("application disconnected", cn)
			case isLicenseException(err):
				log.Error("invalid license", cn)
			case isRestartException(err):
				log.Info("application restarted", cn)
				go app.connectRoutine()
			}
		case r := <-app.connectChan:
			harvest = NewHarvest(time.Now())
			app.SetRun(r)
			log.Info("application connected", cn,
				log.Context{"run": r.RunID})
		}
	}
}

func NewApp(c api.Config) (*App, error) {
	c = copyConfigReferenceFields(c)
	if err := c.Validate(); nil != err {
		return nil, err
	}

	app := &App{
		config:             c,
		connectChan:        make(chan *AppRun),
		collectorErrorChan: make(chan error),
		dataChan:           make(chan appData, appDataChanSize),
		client: &http.Client{
			Transport: c.Transport,
			Timeout:   collectorTimeout,
		},
	}

	log.Info("application created", log.Context{
		"app":     app.String(),
		"version": version.Version,
	})

	if app.config.Development {
		return app, nil
	}

	app.harvestTicker = time.NewTicker(harvestPeriod)
	app.harvestChan = app.harvestTicker.C

	go app.process()
	go app.connectRoutine()

	return app, nil
}

func (app *App) getRun() *AppRun {
	app.RLock()
	defer app.RUnlock()

	if nil == app.run {
		return placeholderRun
	}
	return app.run
}

func (app *App) SetRun(run *AppRun) {
	app.Lock()
	defer app.Unlock()

	app.run = run
}

func (app *App) StartTransaction(name string, w http.ResponseWriter, r *http.Request) api.Transaction {
	run := app.getRun()
	return NewTxn(txnInput{
		Config:   app.config,
		Reply:    run.ConnectReply,
		Request:  r,
		Writer:   w,
		Consumer: app,
	}, name)
}

var (
	ErrHighSecurityEnabled        = errors.New("high security enabled")
	ErrCustomEventsDisabled       = errors.New("custom events disabled")
	ErrCustomEventsRemoteDisabled = errors.New("custom events disabled by server")
)

func (app *App) RecordCustomEvent(eventType string, params map[string]interface{}) error {
	if app.config.HighSecurity {
		return ErrHighSecurityEnabled
	}

	if !app.config.CustomInsightsEvents.Enabled {
		return ErrCustomEventsDisabled
	}

	event, e := createCustomEvent(eventType, params, time.Now())
	if nil != e {
		return e
	}

	run := app.getRun()
	if !run.CollectCustomEvents {
		return ErrCustomEventsRemoteDisabled
	}

	app.Consume(run.RunID, event)

	return nil
}

func (app *App) Consume(id AgentRunID, data Harvestable) {
	if nil != app.TestConsumer {
		app.TestConsumer.Consume(id, data)
		return
	}

	if "" == id {
		return
	}

	app.dataChan <- appData{id, data}
}
