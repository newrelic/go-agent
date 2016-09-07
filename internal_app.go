package newrelic

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/newrelic/go-agent/internal"
	"github.com/newrelic/go-agent/internal/logger"
)

var (
	debugLogging = os.Getenv("NEW_RELIC_DEBUG_LOGGING")
	redirectHost = func() string {
		if s := os.Getenv("NEW_RELIC_HOST"); "" != s {
			return s
		}
		return "collector.newrelic.com"
	}()
)

type dataConsumer interface {
	Consume(internal.AgentRunID, internal.Harvestable)
}

type appData struct {
	id   internal.AgentRunID
	data internal.Harvestable
}

type app struct {
	config      Config
	attrConfig  *internal.AttributeConfig
	rpmControls internal.RpmControls
	testHarvest *internal.Harvest

	harvestTicker      *time.Ticker
	harvestChan        <-chan time.Time
	dataChan           chan appData
	collectorErrorChan chan error
	connectChan        chan *internal.AppRun

	// run is non-nil when the app is successfully connected.  It is
	// immutable.  It is assigned by the processor goroutine and accessed by
	// goroutines calling app API methods.  It should be accessed using
	// getRun and SetRun.
	run *internal.AppRun
	sync.RWMutex
}

var (
	placeholderRun = &internal.AppRun{
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

func (app *app) doHarvest(h *internal.Harvest, harvestStart time.Time, run *internal.AppRun) {
	h.CreateFinalMetrics()
	h.Metrics = h.Metrics.ApplyRules(run.MetricRules)

	payloads := h.Payloads()
	for cmd, p := range payloads {

		data, err := p.Data(run.RunID.String(), harvestStart)

		if nil == data && nil == err {
			continue
		}

		if nil == err {
			call := internal.RpmCmd{
				Collector: run.Collector,
				RunID:     run.RunID.String(),
				Name:      cmd,
				Data:      data,
			}

			// The reply from harvest calls is always unused.
			_, err = internal.CollectorRequest(call, app.rpmControls)
		}

		if nil == err {
			continue
		}

		if isFatalHarvestError(err) {
			app.collectorErrorChan <- err
			return
		}

		app.config.Logger.Warn("harvest failure", map[string]interface{}{
			"cmd":   cmd,
			"error": err.Error(),
		})

		if shouldSaveFailedHarvest(err) {
			app.Consume(run.RunID, p)
		}
	}
}

func connectAttempt(app *app) (*internal.AppRun, error) {
	js, e := configConnectJSON(app.config)
	if nil != e {
		return nil, e
	}
	return internal.ConnectAttempt(js, redirectHost, app.rpmControls)
}

func (app *app) connectRoutine() {
	for {
		run, err := connectAttempt(app)
		if nil == err {
			app.connectChan <- run
			return
		}

		if internal.IsDisconnect(err) || internal.IsLicenseException(err) {
			app.collectorErrorChan <- err
			return
		}

		app.config.Logger.Warn("application connect failure", map[string]interface{}{
			"error": err.Error(),
		})

		time.Sleep(internal.ConnectBackoff)
	}
}

func debug(data internal.Harvestable, lg Logger) {
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
			lg.Debug("integration", map[string]interface{}{
				"cmd":   cmd,
				"error": err.Error(),
			})
			continue
		}
		lg.Debug("integration", map[string]interface{}{
			"cmd":  cmd,
			"data": internal.JSONString(d),
		})
	}
}

func processConnectMessages(run *internal.AppRun, lg Logger) {
	for _, msg := range run.Messages {
		event := "collector message"
		cn := map[string]interface{}{"msg": msg.Message}

		switch strings.ToLower(msg.Level) {
		case "error":
			lg.Error(event, cn)
		case "warn":
			lg.Warn(event, cn)
		case "info":
			lg.Info(event, cn)
		case "debug", "verbose":
			lg.Debug(event, cn)
		}
	}
}

func (app *app) process() {
	var h *internal.Harvest

	for {
		select {
		case <-app.harvestChan:
			run := app.getRun()
			if "" != run.RunID && nil != h {
				now := time.Now()
				go app.doHarvest(h, now, run)
				h = internal.NewHarvest(now)
			}
		case d := <-app.dataChan:
			run := app.getRun()
			if "" != d.id && nil != h && run.RunID == d.id {
				d.data.MergeIntoHarvest(h)
			}

		case err := <-app.collectorErrorChan:
			h = nil
			app.setRun(nil)

			switch {
			case internal.IsDisconnect(err):
				app.config.Logger.Error("application disconnected by New Relic", map[string]interface{}{
					"app": app.config.AppName,
				})
			case internal.IsLicenseException(err):
				app.config.Logger.Error("invalid license", map[string]interface{}{
					"app":     app.config.AppName,
					"license": app.config.License,
				})
			case internal.IsRestartException(err):
				app.config.Logger.Info("application restarted", map[string]interface{}{
					"app": app.config.AppName,
				})
				go app.connectRoutine()
			}
		case r := <-app.connectChan:
			h = internal.NewHarvest(time.Now())
			app.setRun(r)
			app.config.Logger.Info("application connected", map[string]interface{}{
				"app": app.config.AppName,
				"run": r.RunID.String(),
			})
			processConnectMessages(r, app.config.Logger)
		}
	}
}

func makeSHA256(key string) string {
	sum := sha256.Sum256([]byte(key))
	return base64.StdEncoding.EncodeToString(sum[:])
}

const (
	expectedTokenHash = "vZi2AtjcnOh2fbhrybZsDIeJa8JfJiWWEOK6zXhPG2E="
)

func convertAttributeDestinationConfig(c AttributeDestinationConfig) internal.AttributeDestinationConfig {
	return internal.AttributeDestinationConfig{
		Enabled: c.Enabled,
		Include: c.Include,
		Exclude: c.Exclude,
	}
}

func runSampler(app *app, period time.Duration) {
	previous := internal.GetSample(time.Now(), app.config.Logger)

	for now := range time.Tick(period) {
		current := internal.GetSample(now, app.config.Logger)

		run := app.getRun()
		app.Consume(run.RunID, internal.GetStats(internal.Samples{
			Previous: previous,
			Current:  current,
		}))
		previous = current
	}
}

func newApp(c Config) (Application, error) {
	c = copyConfigReferenceFields(c)
	if err := c.Validate(); nil != err {
		return nil, err
	}
	if nil == c.Logger {
		c.Logger = logger.ShimLogger{}
	}
	app := &app{
		config: c,
		attrConfig: internal.CreateAttributeConfig(internal.AttributeConfigInput{
			Attributes:        convertAttributeDestinationConfig(c.Attributes),
			ErrorCollector:    convertAttributeDestinationConfig(c.ErrorCollector.Attributes),
			TransactionEvents: convertAttributeDestinationConfig(c.TransactionEvents.Attributes),
			TransactionTracer: convertAttributeDestinationConfig(c.TransactionTracer.Attributes),
		}),

		connectChan:        make(chan *internal.AppRun),
		collectorErrorChan: make(chan error),
		dataChan:           make(chan appData, internal.AppDataChanSize),
		rpmControls: internal.RpmControls{
			UseTLS:  c.UseTLS,
			License: c.License,
			Client: &http.Client{
				Transport: c.Transport,
				Timeout:   internal.CollectorTimeout,
			},
			Logger:       c.Logger,
			AgentVersion: Version,
		},
	}

	app.config.Logger.Info("application created", map[string]interface{}{
		"app":     app.config.AppName,
		"version": Version,
		"enabled": app.config.Enabled,
	})

	if !app.config.Enabled {
		return app, nil
	}

	app.harvestTicker = time.NewTicker(internal.HarvestPeriod)
	app.harvestChan = app.harvestTicker.C

	go app.process()
	go app.connectRoutine()

	if app.config.RuntimeSampler.Enabled {
		go runSampler(app, internal.RuntimeSamplerPeriod)
	}

	return app, nil
}

type expectApp interface {
	internal.Expect
	Application
}

func newTestApp(replyfn func(*internal.ConnectReply), cfg Config) (expectApp, error) {
	cfg.Enabled = false
	application, err := newApp(cfg)
	if nil != err {
		return nil, err
	}
	app := application.(*app)
	if nil != replyfn {
		reply := internal.ConnectReplyDefaults()
		replyfn(reply)
		app.setRun(&internal.AppRun{ConnectReply: reply})
	}

	app.testHarvest = internal.NewHarvest(time.Now())

	return app, nil
}

func (app *app) getRun() *internal.AppRun {
	app.RLock()
	defer app.RUnlock()

	if nil == app.run {
		return placeholderRun
	}
	return app.run
}

func (app *app) setRun(run *internal.AppRun) {
	app.Lock()
	defer app.Unlock()

	app.run = run
}

// StartTransaction implements newrelic.Application's StartTransaction.
func (app *app) StartTransaction(name string, w http.ResponseWriter, r *http.Request) Transaction {
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
	errHighSecurityEnabled        = errors.New("high security enabled")
	errCustomEventsDisabled       = errors.New("custom events disabled")
	errCustomEventsRemoteDisabled = errors.New("custom events disabled by server")
)

// RecordCustomEvent implements newrelic.Application's RecordCustomEvent.
func (app *app) RecordCustomEvent(eventType string, params map[string]interface{}) error {
	if app.config.HighSecurity {
		return errHighSecurityEnabled
	}

	if !app.config.CustomInsightsEvents.Enabled {
		return errCustomEventsDisabled
	}

	event, e := internal.CreateCustomEvent(eventType, params, time.Now())
	if nil != e {
		return e
	}

	run := app.getRun()
	if !run.CollectCustomEvents {
		return errCustomEventsRemoteDisabled
	}

	app.Consume(run.RunID, event)

	return nil
}

func (app *app) Consume(id internal.AgentRunID, data internal.Harvestable) {
	if "" != debugLogging {
		debug(data, app.config.Logger)
	}

	if nil != app.testHarvest {
		data.MergeIntoHarvest(app.testHarvest)
		return
	}

	if "" == id {
		return
	}

	app.dataChan <- appData{id, data}
}

type addValidatorField struct {
	field    interface{}
	original internal.Validator
}

func (a addValidatorField) Error(fields ...interface{}) {
	fields = append([]interface{}{a.field}, fields...)
	a.original.Error(fields...)
}

func (app *app) ExpectCustomEvents(t internal.Validator, want []internal.WantCustomEvent) {
	internal.ExpectCustomEvents(addValidatorField{`custom events:`, t}, app.testHarvest.CustomEvents, want)
}

func (app *app) ExpectErrors(t internal.Validator, want []internal.WantError) {
	internal.ExpectErrors(addValidatorField{`traced errors:`, t}, app.testHarvest.ErrorTraces, want)
}

func (app *app) ExpectErrorEvents(t internal.Validator, want []internal.WantErrorEvent) {
	internal.ExpectErrorEvents(addValidatorField{`error events:`, t}, app.testHarvest.ErrorEvents, want)
}

func (app *app) ExpectTxnEvents(t internal.Validator, want []internal.WantTxnEvent) {
	internal.ExpectTxnEvents(addValidatorField{`txn events:`, t}, app.testHarvest.TxnEvents, want)
}

func (app *app) ExpectMetrics(t internal.Validator, want []internal.WantMetric) {
	internal.ExpectMetrics(addValidatorField{`metrics:`, t}, app.testHarvest.Metrics, want)
}

func (app *app) ExpectTxnTraces(t internal.Validator, want []internal.WantTxnTrace) {
	internal.ExpectTxnTraces(addValidatorField{`txn traces:`, t}, app.testHarvest.TxnTraces, want)
}
