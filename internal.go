package newrelic

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/newrelic/go-agent/datastore"
	"github.com/newrelic/go-agent/internal"
	"github.com/newrelic/go-agent/internal/logger"
	"github.com/newrelic/go-agent/internal/utilization"
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

const (
	hasC = 1 << iota // CloseNotifier
	hasF             // Flusher
	hasH             // Hijacker
	hasR             // ReaderFrom
)

type wrap struct{ *txn }
type wrapR struct{ *txn }
type wrapH struct{ *txn }
type wrapHR struct{ *txn }
type wrapF struct{ *txn }
type wrapFR struct{ *txn }
type wrapFH struct{ *txn }
type wrapFHR struct{ *txn }
type wrapC struct{ *txn }
type wrapCR struct{ *txn }
type wrapCH struct{ *txn }
type wrapCHR struct{ *txn }
type wrapCF struct{ *txn }
type wrapCFR struct{ *txn }
type wrapCFH struct{ *txn }
type wrapCFHR struct{ *txn }

func (x wrapC) CloseNotify() <-chan bool    { return x.W.(http.CloseNotifier).CloseNotify() }
func (x wrapCR) CloseNotify() <-chan bool   { return x.W.(http.CloseNotifier).CloseNotify() }
func (x wrapCH) CloseNotify() <-chan bool   { return x.W.(http.CloseNotifier).CloseNotify() }
func (x wrapCHR) CloseNotify() <-chan bool  { return x.W.(http.CloseNotifier).CloseNotify() }
func (x wrapCF) CloseNotify() <-chan bool   { return x.W.(http.CloseNotifier).CloseNotify() }
func (x wrapCFR) CloseNotify() <-chan bool  { return x.W.(http.CloseNotifier).CloseNotify() }
func (x wrapCFH) CloseNotify() <-chan bool  { return x.W.(http.CloseNotifier).CloseNotify() }
func (x wrapCFHR) CloseNotify() <-chan bool { return x.W.(http.CloseNotifier).CloseNotify() }

func (x wrapF) Flush()    { x.W.(http.Flusher).Flush() }
func (x wrapFR) Flush()   { x.W.(http.Flusher).Flush() }
func (x wrapFH) Flush()   { x.W.(http.Flusher).Flush() }
func (x wrapFHR) Flush()  { x.W.(http.Flusher).Flush() }
func (x wrapCF) Flush()   { x.W.(http.Flusher).Flush() }
func (x wrapCFR) Flush()  { x.W.(http.Flusher).Flush() }
func (x wrapCFH) Flush()  { x.W.(http.Flusher).Flush() }
func (x wrapCFHR) Flush() { x.W.(http.Flusher).Flush() }

func (x wrapH) Hijack() (net.Conn, *bufio.ReadWriter, error)    { return x.W.(http.Hijacker).Hijack() }
func (x wrapHR) Hijack() (net.Conn, *bufio.ReadWriter, error)   { return x.W.(http.Hijacker).Hijack() }
func (x wrapFH) Hijack() (net.Conn, *bufio.ReadWriter, error)   { return x.W.(http.Hijacker).Hijack() }
func (x wrapFHR) Hijack() (net.Conn, *bufio.ReadWriter, error)  { return x.W.(http.Hijacker).Hijack() }
func (x wrapCH) Hijack() (net.Conn, *bufio.ReadWriter, error)   { return x.W.(http.Hijacker).Hijack() }
func (x wrapCHR) Hijack() (net.Conn, *bufio.ReadWriter, error)  { return x.W.(http.Hijacker).Hijack() }
func (x wrapCFH) Hijack() (net.Conn, *bufio.ReadWriter, error)  { return x.W.(http.Hijacker).Hijack() }
func (x wrapCFHR) Hijack() (net.Conn, *bufio.ReadWriter, error) { return x.W.(http.Hijacker).Hijack() }

func (x wrapR) ReadFrom(r io.Reader) (int64, error)    { return x.W.(io.ReaderFrom).ReadFrom(r) }
func (x wrapHR) ReadFrom(r io.Reader) (int64, error)   { return x.W.(io.ReaderFrom).ReadFrom(r) }
func (x wrapFR) ReadFrom(r io.Reader) (int64, error)   { return x.W.(io.ReaderFrom).ReadFrom(r) }
func (x wrapFHR) ReadFrom(r io.Reader) (int64, error)  { return x.W.(io.ReaderFrom).ReadFrom(r) }
func (x wrapCR) ReadFrom(r io.Reader) (int64, error)   { return x.W.(io.ReaderFrom).ReadFrom(r) }
func (x wrapCHR) ReadFrom(r io.Reader) (int64, error)  { return x.W.(io.ReaderFrom).ReadFrom(r) }
func (x wrapCFR) ReadFrom(r io.Reader) (int64, error)  { return x.W.(io.ReaderFrom).ReadFrom(r) }
func (x wrapCFHR) ReadFrom(r io.Reader) (int64, error) { return x.W.(io.ReaderFrom).ReadFrom(r) }

func upgradeTxn(txn *txn) Transaction {
	x := 0
	if _, ok := txn.W.(http.CloseNotifier); ok {
		x |= hasC
	}
	if _, ok := txn.W.(http.Flusher); ok {
		x |= hasF
	}
	if _, ok := txn.W.(http.Hijacker); ok {
		x |= hasH
	}
	if _, ok := txn.W.(io.ReaderFrom); ok {
		x |= hasR
	}

	switch x {
	default:
		// Wrap the transaction even when there are no methods needed to
		// ensure consistent error stack trace depth.
		return wrap{txn}
	case hasR:
		return wrapR{txn}
	case hasH:
		return wrapH{txn}
	case hasH | hasR:
		return wrapHR{txn}
	case hasF:
		return wrapF{txn}
	case hasF | hasR:
		return wrapFR{txn}
	case hasF | hasH:
		return wrapFH{txn}
	case hasF | hasH | hasR:
		return wrapFHR{txn}
	case hasC:
		return wrapC{txn}
	case hasC | hasR:
		return wrapCR{txn}
	case hasC | hasH:
		return wrapCH{txn}
	case hasC | hasH | hasR:
		return wrapCHR{txn}
	case hasC | hasF:
		return wrapCF{txn}
	case hasC | hasF | hasR:
		return wrapCFR{txn}
	case hasC | hasF | hasH:
		return wrapCFH{txn}
	case hasC | hasF | hasH | hasR:
		return wrapCFHR{txn}
	}
}

type dataConsumer interface {
	Consume(internal.AgentRunID, internal.Harvestable)
}

type txnInput struct {
	W          http.ResponseWriter
	Request    *http.Request
	Config     Config
	Reply      *internal.ConnectReply
	Consumer   dataConsumer
	attrConfig *internal.AttributeConfig
}

type txn struct {
	txnInput
	// This mutex is required since the consumer may call the public API
	// interface functions from different routines.
	sync.Mutex
	// finished indicates whether or not End() has been called.  After
	// finished has been set to true, no recording should occur.
	finished   bool
	queuing    time.Duration
	start      time.Time
	name       string // Work in progress name
	isWeb      bool
	ignore     bool
	errors     internal.TxnErrors // Lazily initialized.
	errorsSeen uint64
	attrs      *internal.Attributes

	// Fields relating to tracing and breakdown metrics/segments.
	tracer internal.Tracer

	// wroteHeader prevents capturing multiple response code errors if the
	// user erroneously calls WriteHeader multiple times.
	wroteHeader bool

	// Fields assigned at completion
	stop           time.Time
	duration       time.Duration
	finalName      string // Full finalized metric name
	zone           internal.ApdexZone
	apdexThreshold time.Duration
}

func newTxn(input txnInput, name string) *txn {
	txn := &txn{
		txnInput: input,
		start:    time.Now(),
		name:     name,
		isWeb:    nil != input.Request,
		attrs:    internal.NewAttributes(input.attrConfig),
	}
	if nil != txn.Request {
		txn.queuing = internal.QueueDuration(input.Request.Header, txn.start)
		internal.RequestAgentAttributes(txn.attrs, input.Request)
	}
	txn.attrs.Agent.HostDisplayName = txn.Config.HostDisplayName

	return txn
}

func (txn *txn) txnEventsEnabled() bool {
	return txn.Config.TransactionEvents.Enabled &&
		txn.Reply.CollectAnalyticsEvents
}

func (txn *txn) errorEventsEnabled() bool {
	return txn.Config.ErrorCollector.CaptureEvents &&
		txn.Reply.CollectErrorEvents
}

func (txn *txn) freezeName() {
	if txn.ignore || ("" != txn.finalName) {
		return
	}

	txn.finalName = internal.CreateFullTxnName(txn.name, txn.Reply, txn.isWeb)
	if "" == txn.finalName {
		txn.ignore = true
	}
}

func (txn *txn) getsApdex() bool {
	return txn.isWeb
}

func (txn *txn) MergeIntoHarvest(h *internal.Harvest) {
	exclusive := time.Duration(0)
	children := internal.TracerRootChildren(&txn.tracer)
	if txn.duration > children {
		exclusive = txn.duration - children
	}

	internal.CreateTxnMetrics(internal.CreateTxnMetricsArgs{
		IsWeb:          txn.isWeb,
		Duration:       txn.duration,
		Exclusive:      exclusive,
		Name:           txn.finalName,
		Zone:           txn.zone,
		ApdexThreshold: txn.apdexThreshold,
		ErrorsSeen:     txn.errorsSeen,
		Queueing:       txn.queuing,
	}, h.Metrics)

	internal.MergeBreakdownMetrics(&txn.tracer, h.Metrics, txn.finalName, txn.isWeb)

	if txn.txnEventsEnabled() {
		h.TxnEvents.AddTxnEvent(&internal.TxnEvent{
			Name:      txn.finalName,
			Timestamp: txn.start,
			Duration:  txn.duration,
			Queuing:   txn.queuing,
			Zone:      txn.zone,
			Attrs:     txn.attrs,
			DatastoreExternalTotals: txn.tracer.DatastoreExternalTotals,
		})
	}

	requestURI := ""
	if nil != txn.Request && nil != txn.Request.URL {
		requestURI = internal.SafeURL(txn.Request.URL)
	}

	internal.MergeTxnErrors(h.ErrorTraces, txn.errors, txn.finalName, requestURI, txn.attrs)

	if txn.errorEventsEnabled() {
		for _, e := range txn.errors {
			h.ErrorEvents.Add(&internal.ErrorEvent{
				Klass:    e.Klass,
				Msg:      e.Msg,
				When:     e.When,
				TxnName:  txn.finalName,
				Duration: txn.duration,
				Queuing:  txn.queuing,
				Attrs:    txn.attrs,
				DatastoreExternalTotals: txn.tracer.DatastoreExternalTotals,
			})
		}
	}
}

func responseCodeIsError(cfg *Config, code int) bool {
	if code < http.StatusBadRequest { // 400
		return false
	}
	for _, ignoreCode := range cfg.ErrorCollector.IgnoreStatusCodes {
		if code == ignoreCode {
			return false
		}
	}
	return true
}

func headersJustWritten(txn *txn, code int) {
	if txn.finished {
		return
	}
	if txn.wroteHeader {
		return
	}
	txn.wroteHeader = true

	internal.ResponseHeaderAttributes(txn.attrs, txn.W.Header())
	internal.ResponseCodeAttribute(txn.attrs, code)

	if responseCodeIsError(&txn.Config, code) {
		e := internal.TxnErrorFromResponseCode(code)
		e.Stack = internal.GetStackTrace(1)
		txn.noticeErrorInternal(e)
	}
}

func (txn *txn) Header() http.Header { return txn.W.Header() }

func (txn *txn) Write(b []byte) (int, error) {
	n, err := txn.W.Write(b)

	txn.Lock()
	defer txn.Unlock()

	headersJustWritten(txn, http.StatusOK)

	return n, err
}

func (txn *txn) WriteHeader(code int) {
	txn.W.WriteHeader(code)

	txn.Lock()
	defer txn.Unlock()

	headersJustWritten(txn, code)
}

func (txn *txn) End() error {
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return errAlreadyEnded
	}

	txn.finished = true

	r := recover()
	if nil != r {
		e := internal.TxnErrorFromPanic(r)
		e.Stack = internal.GetStackTrace(0)
		txn.noticeErrorInternal(e)
	}

	txn.stop = time.Now()
	txn.duration = txn.stop.Sub(txn.start)

	txn.freezeName()
	if txn.getsApdex() {
		txn.apdexThreshold = internal.CalculateApdexThreshold(txn.Reply, txn.finalName)
		if txn.errorsSeen > 0 {
			txn.zone = internal.ApdexFailing
		} else {
			txn.zone = internal.CalculateApdexZone(txn.apdexThreshold, txn.duration)
		}
	} else {
		txn.zone = internal.ApdexNone
	}

	if txn.Config.Logger.DebugEnabled() {
		txn.Config.Logger.Debug("transaction ended", map[string]interface{}{
			"name":        txn.finalName,
			"duration_ms": txn.duration.Seconds() * 1000.0,
			"ignored":     txn.ignore,
			"run":         txn.Reply.RunID,
		})
	}

	if !txn.ignore {
		txn.Consumer.Consume(txn.Reply.RunID, txn)
	}

	// Note that if a consumer uses `panic(nil)`, the panic will not
	// propagate.
	if nil != r {
		panic(r)
	}

	return nil
}

func (txn *txn) AddAttribute(name string, value interface{}) error {
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return errAlreadyEnded
	}

	return internal.AddUserAttribute(txn.attrs, name, value, internal.DestAll)
}

var (
	errorsLocallyDisabled  = errors.New("errors locally disabled")
	errorsRemotelyDisabled = errors.New("errors remotely disabled")
	errNilError            = errors.New("nil error")
	errAlreadyEnded        = errors.New("transaction has already ended")
)

const (
	highSecurityErrorMsg = "message removed by high security setting"
)

func (txn *txn) noticeErrorInternal(err internal.TxnError) error {
	// Increment errorsSeen even if errors are disabled:  Error metrics do
	// not depend on whether or not errors are enabled.
	txn.errorsSeen++

	if !txn.Config.ErrorCollector.Enabled {
		return errorsLocallyDisabled
	}

	if !txn.Reply.CollectErrors {
		return errorsRemotelyDisabled
	}

	if nil == txn.errors {
		txn.errors = internal.NewTxnErrors(internal.MaxTxnErrors)
	}

	if txn.Config.HighSecurity {
		err.Msg = highSecurityErrorMsg
	}

	err.When = time.Now()

	txn.errors.Add(&err)

	return nil
}

func (txn *txn) NoticeError(err error) error {
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return errAlreadyEnded
	}

	if nil == err {
		return errNilError
	}

	e := internal.TxnErrorFromError(err)
	e.Stack = internal.GetStackTrace(2)
	return txn.noticeErrorInternal(e)
}

func (txn *txn) SetName(name string) error {
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return errAlreadyEnded
	}

	txn.name = name
	return nil
}

func (txn *txn) Ignore() error {
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return errAlreadyEnded
	}
	txn.ignore = true
	return nil
}

func (txn *txn) StartSegment() Token {
	token := Token(0)
	txn.Lock()
	if !txn.finished {
		token = Token(internal.StartSegment(&txn.tracer, time.Now()))
	}
	txn.Unlock()
	return token
}

func (txn *txn) EndSegment(token Token, name string) {
	txn.Lock()
	if !txn.finished {
		internal.EndBasicSegment(&txn.tracer, internal.Token(token), time.Now(), name)
	}
	txn.Unlock()
}

func (txn *txn) EndDatastore(token Token, s datastore.Segment) {
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return
	}
	internal.EndDatastoreSegment(&txn.tracer, internal.Token(token), time.Now(), s)
}

func (txn *txn) EndExternal(token Token, url string) {
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return
	}
	internal.EndExternalSegment(&txn.tracer, internal.Token(token), time.Now(), internal.HostFromExternalURL(url))
}

func (txn *txn) PrepareRequest(token Token, request *http.Request) {
	txn.Lock()
	defer txn.Unlock()

	// TODO: handle request CAT headers
}

func hostFromRequestResponse(request *http.Request, response *http.Response) string {
	if nil != response && nil != response.Request {
		request = response.Request
	}
	if nil == request || nil == request.URL {
		return ""
	}
	if "" != request.URL.Opaque {
		return "opaque"
	}
	return request.URL.Host
}

func (txn *txn) EndRequest(token Token, request *http.Request, response *http.Response) {
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return
	}

	// TODO: handle response CAT headers

	host := hostFromRequestResponse(request, response)
	internal.EndExternalSegment(&txn.tracer, internal.Token(token), time.Now(), host)
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
	js, e := configConnectJSON(&app.config)
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

var (
	betaURL               = "http://goo.gl/forms/Rcv1b10Qvt1ENLlr1"
	errMissingBetaToken   = errors.New("missing beta token: please sign the Beta Agreement:  " + betaURL)
	errIncorrectBetaToken = errors.New("incorrect beta token: please contact New Relic")
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

func newAppInternal(c Config) (Application, error) {
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

func newApp(c Config) (Application, error) {
	if "" == c.BetaToken {
		return nil, errMissingBetaToken
	}
	if b := makeSHA256(c.BetaToken); b != expectedTokenHash {
		return nil, errIncorrectBetaToken
	}
	return newAppInternal(c)
}

type expectApp interface {
	internal.Expect
	Application
}

func newTestApp(replyfn func(*internal.ConnectReply), cfg Config) (expectApp, error) {
	cfg.Enabled = false
	application, err := newAppInternal(cfg)
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

func copyDestConfig(c AttributeDestinationConfig) AttributeDestinationConfig {
	cp := c
	if nil != c.Include {
		cp.Include = make([]string, len(c.Include))
		copy(cp.Include, c.Include)
	}
	if nil != c.Exclude {
		cp.Exclude = make([]string, len(c.Exclude))
		copy(cp.Exclude, c.Exclude)
	}
	return cp
}

func copyConfigReferenceFields(cfg Config) Config {
	cp := cfg
	if nil != cfg.Labels {
		cp.Labels = make(map[string]string, len(cfg.Labels))
		for key, val := range cfg.Labels {
			cp.Labels[key] = val
		}
	}
	if nil != cfg.ErrorCollector.IgnoreStatusCodes {
		ignored := make([]int, len(cfg.ErrorCollector.IgnoreStatusCodes))
		copy(ignored, cfg.ErrorCollector.IgnoreStatusCodes)
		cp.ErrorCollector.IgnoreStatusCodes = ignored
	}

	cp.Attributes = copyDestConfig(cfg.Attributes)
	cp.ErrorCollector.Attributes = copyDestConfig(cfg.ErrorCollector.Attributes)
	cp.TransactionEvents.Attributes = copyDestConfig(cfg.TransactionEvents.Attributes)

	return cp
}

const (
	agentLanguage = "go"
)

func transportSetting(t http.RoundTripper) interface{} {
	if nil == t {
		return nil
	}
	return fmt.Sprintf("%T", t)
}

func loggerSetting(lg Logger) interface{} {
	if nil == lg {
		return nil
	}
	if _, ok := lg.(logger.ShimLogger); ok {
		return nil
	}
	return fmt.Sprintf("%T", lg)
}

const (
	// https://source.datanerd.us/agents/agent-specs/blob/master/Custom-Host-Names.md
	hostByteLimit = 255
)

type settings Config

func (s *settings) MarshalJSON() ([]byte, error) {
	c := (*Config)(s)
	js, err := json.Marshal(c)
	if nil != err {
		return nil, err
	}
	fields := make(map[string]interface{})
	err = json.Unmarshal(js, &fields)
	if nil != err {
		return nil, err
	}
	// The License field is not simply ignored by adding the `json:"-"` tag
	// to it since we want to allow consumers to populate Config from JSON.
	delete(fields, `License`)
	fields[`Transport`] = transportSetting(c.Transport)
	fields[`Logger`] = loggerSetting(c.Logger)
	return json.Marshal(fields)
}

func configConnectJSONInternal(c *Config, pid int, util *utilization.Data, e internal.Environment, version string) ([]byte, error) {
	return json.Marshal([]interface{}{struct {
		Pid             int                  `json:"pid"`
		Language        string               `json:"language"`
		Version         string               `json:"agent_version"`
		Host            string               `json:"host"`
		HostDisplayName string               `json:"display_host,omitempty"`
		Settings        interface{}          `json:"settings"`
		AppName         []string             `json:"app_name"`
		HighSecurity    bool                 `json:"high_security"`
		Labels          internal.Labels      `json:"labels,omitempty"`
		Environment     internal.Environment `json:"environment"`
		Identifier      string               `json:"identifier"`
		Util            *utilization.Data    `json:"utilization"`
	}{
		Pid:             pid,
		Language:        agentLanguage,
		Version:         version,
		Host:            internal.StringLengthByteLimit(util.Hostname, hostByteLimit),
		HostDisplayName: internal.StringLengthByteLimit(c.HostDisplayName, hostByteLimit),
		Settings:        (*settings)(c),
		AppName:         strings.Split(c.AppName, ";"),
		HighSecurity:    c.HighSecurity,
		Labels:          internal.Labels(c.Labels),
		Environment:     e,
		// This identifier field is provided to avoid:
		// https://newrelic.atlassian.net/browse/DSCORE-778
		//
		// This identifier is used by the collector to look up the real
		// agent. If an identifier isn't provided, the collector will
		// create its own based on the first appname, which prevents a
		// single daemon from connecting "a;b" and "a;c" at the same
		// time.
		//
		// Providing the identifier below works around this issue and
		// allows users more flexibility in using application rollups.
		Identifier: c.AppName,
		Util:       util,
	}})
}

func configConnectJSON(c *Config) ([]byte, error) {
	env := internal.NewEnvironment()
	util := utilization.Gather(utilization.Config{
		DetectAWS:    c.Utilization.DetectAWS,
		DetectDocker: c.Utilization.DetectDocker,
	}, c.Logger)
	return configConnectJSONInternal(c, os.Getpid(), util, env, Version)
}
