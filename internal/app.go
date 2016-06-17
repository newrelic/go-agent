package internal

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/newrelic/go-agent/api"
	"github.com/newrelic/go-agent/log"
	"github.com/newrelic/go-agent/version"
)

// appRun contains information regarding a single connection session with the
// collector.  It is created upon application connect and is afterwards
// immutable.
type appRun struct {
	*ConnectReply
	collector string
}

type appData struct {
	id   AgentRunID
	data harvestable
}

// App is the implementation of api.Application.
type App struct {
	config      api.Config
	attrConfig  *attributeConfig
	client      *http.Client
	testHarvest *harvest

	harvestTicker      *time.Ticker
	harvestChan        <-chan time.Time
	dataChan           chan appData
	collectorErrorChan chan error
	connectChan        chan *appRun

	// run is non-nil when the app is successfully connected.  It is
	// immutable.  It is assigned by the processor goroutine and accessed by
	// goroutines calling app API methods.  It should be accessed using
	// getRun and SetRun.
	run *appRun
	sync.RWMutex
}

func (app *App) String() string {
	return app.config.AppName
}

var (
	placeholderRun = &appRun{
		ConnectReply: connectReplyDefaults(),
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

func (app *App) doHarvest(h *harvest, harvestStart time.Time, run *appRun) {
	h.createFinalMetrics()
	h.applyMetricRules(run.MetricRules)

	payloads := h.payloads()
	for cmd, p := range payloads {

		data, err := p.Data(run.RunID.String(), harvestStart)

		if nil == data && nil == err {
			continue
		}

		if nil == err {
			call := rpmCmd{
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
			app.consume(run.RunID, p)
		}
	}
}

func (app *App) connectRoutine() {
	for {
		collector, reply, err := connectAttempt(&app.config, app.client)
		if nil == err {
			app.connectChan <- &appRun{reply, collector}
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

func debug(data harvestable) {
	now := time.Now()
	h := newHarvest(now)
	data.mergeIntoHarvest(h)
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
	var h *harvest

	cn := log.Context{
		"app":     app.String(),
		"license": app.config.License,
	}

	for {
		select {
		case <-app.harvestChan:
			run := app.getRun()
			if "" != run.RunID && nil != h {
				now := time.Now()
				go app.doHarvest(h, now, run)
				h = newHarvest(now)
			}
		case d := <-app.dataChan:
			if "" != debugLogging {
				debug(d.data)
			}

			run := app.getRun()
			if "" != d.id && nil != h && run.RunID == d.id {
				d.data.mergeIntoHarvest(h)
			}

		case err := <-app.collectorErrorChan:
			h = nil
			app.setRun(nil)

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
			h = newHarvest(time.Now())
			app.setRun(r)
			log.Info("application connected", cn,
				log.Context{"run": r.RunID})
		}
	}
}

// NewApp creates and returns an App or an error.
func NewApp(c api.Config) (api.Application, error) {
	c = copyConfigReferenceFields(c)
	if err := c.Validate(); nil != err {
		return nil, err
	}

	app := &App{
		config: c,
		attrConfig: createAttributeConfig(attributeConfigInput{
			attributes:        c.Attributes,
			errorCollector:    c.ErrorCollector.Attributes,
			transactionEvents: c.TransactionEvents.Attributes,
		}),

		connectChan:        make(chan *appRun),
		collectorErrorChan: make(chan error),
		dataChan:           make(chan appData, appDataChanSize),
		client: &http.Client{
			Transport: c.Transport,
			Timeout:   collectorTimeout,
		},
	}

	log.Info("application created", log.Context{
		"app":         app.String(),
		"version":     version.Version,
		"development": app.config.Development,
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

// ExpectApp exposes captured data for testing in the internal/test package.
type ExpectApp interface {
	Expect
	api.Application
}

// NewTestApp returns an ExpectApp for testing in the internal/test package.
func NewTestApp(replyfn func(*ConnectReply), cfg api.Config) (ExpectApp, error) {
	cfg.Development = true
	application, err := NewApp(cfg)
	if nil != err {
		return nil, err
	}
	app := application.(*App)
	if nil != replyfn {
		reply := connectReplyDefaults()
		replyfn(reply)
		app.setRun(&appRun{ConnectReply: reply})
	}

	app.testHarvest = newHarvest(time.Now())

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

// StartTransaction implements newrelic.Application's StartTransaction.
func (app *App) StartTransaction(name string, w http.ResponseWriter, r *http.Request) api.Transaction {
	run := app.getRun()
	return upgradeTxn(newTxn(txnInput{
		Config:     app.config,
		Reply:      run.ConnectReply,
		Request:    r,
		W:          w,
		Consumer:   app,
		attrConfig: app.attrConfig,
	}, name))
}

var (
	// ErrHighSecurityEnabled is returned by app.RecordCustomEvent if high
	// security mode is enabled.
	ErrHighSecurityEnabled = errors.New("high security enabled")
	// ErrCustomEventsDisabled is returned by app.RecordCustomEvent if
	// custom events have been disabled in te api.Config structure.
	ErrCustomEventsDisabled = errors.New("custom events disabled")
	// ErrCustomEventsRemoteDisabled is returned by app.RecordCustomEvent if
	// custom events have been disabled by the collector response.
	ErrCustomEventsRemoteDisabled = errors.New("custom events disabled by server")
)

// RecordCustomEvent implements newrelic.Application's RecordCustomEvent.
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

	app.consume(run.RunID, event)

	return nil
}

func (app *App) consume(id AgentRunID, data harvestable) {
	if nil != app.testHarvest {
		data.mergeIntoHarvest(app.testHarvest)
		return
	}

	if "" == id {
		return
	}

	app.dataChan <- appData{id, data}
}
