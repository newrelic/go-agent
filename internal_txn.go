package newrelic

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/VadimBelov/go-agent/internal"
)

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
		if !txn.Config.ErrorCollector.Enabled {
			txn.ignore = true
		}
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

func (txn *txn) StartSegmentNow() SegmentStartTime {
	token := internal.Token(0)
	txn.Lock()
	if !txn.finished {
		token = internal.StartSegment(&txn.tracer, time.Now())
	}
	txn.Unlock()
	return SegmentStartTime{
		segment: segment{
			token: token,
			txn:   txn,
		},
	}
}

type segment struct {
	token internal.Token
	txn   *txn
}

func endSegment(s Segment) {
	txn := s.StartTime.txn
	if nil == txn {
		return
	}
	txn.Lock()
	if !txn.finished {
		internal.EndBasicSegment(&txn.tracer, s.StartTime.token, time.Now(), s.Name)
	}
	txn.Unlock()
}

func endDatastore(s DatastoreSegment) {
	txn := s.StartTime.txn
	if nil == txn {
		return
	}
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return
	}
	internal.EndDatastoreSegment(&txn.tracer, s.StartTime.token, time.Now(), internal.DatastoreMetricKey{
		Product:    string(s.Product),
		Collection: s.Collection,
		Operation:  s.Operation,
	})
}

func endExternal(s ExternalSegment) {
	txn := s.StartTime.txn
	if nil == txn {
		return
	}
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return
	}
	host := hostFromRequestResponse(s.Request, s.Response)
	if "" != s.URL {
		host = internal.HostFromExternalURL(s.URL)
	}
	internal.EndExternalSegment(&txn.tracer, s.StartTime.token, time.Now(), host)
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
