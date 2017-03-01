package newrelic

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"sync"
	"time"

	"github.com/newrelic/go-agent/internal"
)

type txnInput struct {
	W          http.ResponseWriter
	Config     Config
	Reply      *internal.ConnectReply
	Consumer   dataConsumer
	attrConfig *internal.AttributeConfig
}

func newTxnID() string {
	bits := internal.RandUint64()
	return fmt.Sprintf("%x", bits)
}

type txn struct {
	txnInput
	// This mutex is required since the consumer may call the public API
	// interface functions from different routines.
	sync.Mutex
	// finished indicates whether or not End() has been called.  After
	// finished has been set to true, no recording should occur.
	finished       bool
	sequenceNum    uint32
	payloadCreated bool

	Name   string // Work in progress name
	ignore bool

	// wroteHeader prevents capturing multiple response code errors if the
	// user erroneously calls WriteHeader multiple times.
	wroteHeader bool

	internal.TxnData
}

func newTxn(input txnInput, req *http.Request, name string) *txn {
	txn := &txn{
		txnInput: input,
	}
	txn.Start = time.Now()
	txn.Name = name
	txn.IsWeb = nil != req
	txn.Attrs = internal.NewAttributes(input.attrConfig)
	txn.Priority = internal.NewPriority(input.Reply.EncodingKeyHash)
	txn.ID = newTxnID()
	if nil != req {
		txn.Proxies = internal.NewProxies(req.Header, txn.Start)
		internal.RequestAgentAttributes(txn.Attrs, req)
	}
	txn.Attrs.Agent.HostDisplayName = txn.Config.HostDisplayName
	txn.TxnTrace.Enabled = txn.txnTracesEnabled()
	txn.TxnTrace.SegmentThreshold = txn.Config.TransactionTracer.SegmentThreshold
	txn.StackTraceThreshold = txn.Config.TransactionTracer.StackTraceThreshold
	txn.SlowQueriesEnabled = txn.slowQueriesEnabled()
	txn.SlowQueryThreshold = txn.Config.DatastoreTracer.SlowQuery.Threshold
	if nil != req && nil != req.URL {
		txn.CleanURL = internal.SafeURL(req.URL)
	}

	return txn
}

func (txn *txn) slowQueriesEnabled() bool {
	return txn.Config.DatastoreTracer.SlowQuery.Enabled &&
		txn.Reply.CollectTraces
}

func (txn *txn) txnTracesEnabled() bool {
	return txn.Config.TransactionTracer.Enabled &&
		txn.Reply.CollectTraces
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
	if txn.ignore || ("" != txn.FinalName) {
		return
	}

	txn.FinalName = internal.CreateFullTxnName(txn.Name, txn.Reply, txn.IsWeb)
	if "" == txn.FinalName {
		txn.ignore = true
	}
}

func (txn *txn) getsApdex() bool {
	return txn.IsWeb
}

func (txn *txn) txnTraceThreshold() time.Duration {
	if txn.Config.TransactionTracer.Threshold.IsApdexFailing {
		return internal.ApdexFailingThreshold(txn.ApdexThreshold)
	}
	return txn.Config.TransactionTracer.Threshold.Duration
}

func (txn *txn) shouldSaveTrace() bool {
	return txn.txnTracesEnabled() &&
		(txn.Duration >= txn.txnTraceThreshold())
}

func (txn *txn) MergeIntoHarvest(h *internal.Harvest) {
	internal.CreateTxnMetrics(&txn.TxnData, h.Metrics)
	internal.MergeBreakdownMetrics(&txn.TxnData, h.Metrics)

	if txn.txnEventsEnabled() {
		// Allocate a new TxnEvent to prevent a reference to the large transaction.
		alloc := new(internal.TxnEvent)
		*alloc = txn.TxnData.TxnEvent
		h.TxnEvents.AddTxnEvent(alloc, txn.Priority.Value())
	}

	internal.MergeTxnErrors(&h.ErrorTraces, txn.Errors, txn.TxnEvent)

	if txn.errorEventsEnabled() {
		for _, e := range txn.Errors {
			errEvent := &internal.ErrorEvent{
				ErrorData: *e,
				TxnEvent:  txn.TxnEvent,
			}
			// Since the stack trace is not used in error events, remove the reference
			// to minimize memory.
			errEvent.Stack = nil
			h.ErrorEvents.Add(errEvent, txn.Priority.Value())
		}
	}

	if txn.shouldSaveTrace() {
		h.TxnTraces.Witness(internal.HarvestTrace{
			TxnEvent: txn.TxnEvent,
			Trace:    txn.TxnTrace,
		})
	}

	if nil != txn.SlowQueries {
		h.SlowSQLs.Merge(txn.SlowQueries, txn.FinalName, txn.CleanURL)
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

	internal.ResponseHeaderAttributes(txn.Attrs, txn.W.Header())
	internal.ResponseCodeAttribute(txn.Attrs, code)

	if responseCodeIsError(&txn.Config, code) {
		e := internal.TxnErrorFromResponseCode(time.Now(), code)
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
		e := internal.TxnErrorFromPanic(time.Now(), r)
		e.Stack = internal.GetStackTrace(0)
		txn.noticeErrorInternal(e)
	}

	txn.Stop = time.Now()
	txn.Duration = txn.Stop.Sub(txn.Start)
	if children := internal.TracerRootChildren(&txn.TxnData); txn.Duration > children {
		txn.Exclusive = txn.Duration - children
	}

	txn.freezeName()

	// Assign apdexThreshold regardless of whether or not the transaction
	// gets apdex since it may be used to calculate the trace threshold.
	txn.ApdexThreshold = internal.CalculateApdexThreshold(txn.Reply, txn.FinalName)

	if txn.getsApdex() {
		if txn.HasErrors() {
			txn.Zone = internal.ApdexFailing
		} else {
			txn.Zone = internal.CalculateApdexZone(txn.ApdexThreshold, txn.Duration)
		}
	} else {
		txn.Zone = internal.ApdexNone
	}

	if txn.Config.Logger.DebugEnabled() {
		txn.Config.Logger.Debug("transaction ended", map[string]interface{}{
			"name":        txn.FinalName,
			"duration_ms": txn.Duration.Seconds() * 1000.0,
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

	return internal.AddUserAttribute(txn.Attrs, name, value, internal.DestAll)
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

func (txn *txn) noticeErrorInternal(err internal.ErrorData) error {
	if !txn.Config.ErrorCollector.Enabled {
		return errorsLocallyDisabled
	}

	if !txn.Reply.CollectErrors {
		return errorsRemotelyDisabled
	}

	if nil == txn.Errors {
		txn.Errors = internal.NewTxnErrors(internal.MaxTxnErrors)
	}

	if txn.Config.HighSecurity {
		err.Msg = highSecurityErrorMsg
	}

	txn.Errors.Add(err)

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

	e := internal.ErrorData{
		When: time.Now(),
		Msg:  err.Error(),
	}
	if ec, ok := err.(ErrorClasser); ok {
		e.Klass = ec.ErrorClass()
	}
	if "" == e.Klass {
		e.Klass = reflect.TypeOf(err).String()
	}
	if st, ok := err.(StackTracer); ok {
		e.Stack = st.StackTrace()
		// Note that if the provided stack trace is excessive in length,
		// it will be truncated during JSON creation.
	}
	if nil == e.Stack {
		e.Stack = internal.GetStackTrace(2)
	}

	return txn.noticeErrorInternal(e)
}

func (txn *txn) SetName(name string) error {
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return errAlreadyEnded
	}

	txn.Name = name
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
	var s internal.SegmentStartTime
	txn.Lock()
	if !txn.finished {
		s = internal.StartSegment(&txn.TxnData, time.Now())
	}
	txn.Unlock()
	return SegmentStartTime{
		segment: segment{
			start: s,
			txn:   txn,
		},
	}
}

type segment struct {
	start internal.SegmentStartTime
	txn   *txn
}

func endSegment(s Segment) error {
	txn := s.StartTime.txn
	if nil == txn {
		return nil
	}
	var err error
	txn.Lock()
	if txn.finished {
		err = errAlreadyEnded
	} else {
		err = internal.EndBasicSegment(&txn.TxnData, s.StartTime.start, time.Now(), s.Name)
	}
	txn.Unlock()
	return err
}

func endDatastore(s DatastoreSegment) error {
	txn := s.StartTime.txn
	if nil == txn {
		return nil
	}
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return errAlreadyEnded
	}
	if txn.Config.HighSecurity {
		s.QueryParameters = nil
	}
	if !txn.Config.DatastoreTracer.QueryParameters.Enabled {
		s.QueryParameters = nil
	}
	if !txn.Config.DatastoreTracer.DatabaseNameReporting.Enabled {
		s.DatabaseName = ""
	}
	if !txn.Config.DatastoreTracer.InstanceReporting.Enabled {
		s.Host = ""
		s.PortPathOrID = ""
	}
	return internal.EndDatastoreSegment(internal.EndDatastoreParams{
		Tracer:             &txn.TxnData,
		Start:              s.StartTime.start,
		Now:                time.Now(),
		Product:            string(s.Product),
		Collection:         s.Collection,
		Operation:          s.Operation,
		ParameterizedQuery: s.ParameterizedQuery,
		QueryParameters:    s.QueryParameters,
		Host:               s.Host,
		PortPathOrID:       s.PortPathOrID,
		Database:           s.DatabaseName,
	})
}

func externalSegmentURL(s ExternalSegment) (*url.URL, error) {
	if "" != s.URL {
		return url.Parse(s.URL)
	}
	r := s.Request
	if nil != s.Response && nil != s.Response.Request {
		r = s.Response.Request
	}
	if r != nil {
		return r.URL, nil
	}
	return nil, nil
}

func endExternal(s ExternalSegment) error {
	txn := s.StartTime.txn
	if nil == txn {
		return nil
	}
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return errAlreadyEnded
	}
	u, err := externalSegmentURL(s)
	if nil != err {
		return err
	}
	return internal.EndExternalSegment(&txn.TxnData, s.StartTime.start, time.Now(), u)
}

func (txn *txn) sequence() int {
	s := txn.sequenceNum
	txn.sequenceNum++
	return int(s)
}

type shimPayload struct{}

func (s shimPayload) Text() string     { return "" }
func (s shimPayload) HTTPSafe() string { return "" }

func (txn *txn) CreateDistributedTracePayload(u *url.URL) DistributedTracePayload {
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return shimPayload{}
	}

	if "" == txn.Reply.AppID {
		// Return a shimPayload if the application is not yet connected.
		return shimPayload{}
	}

	txn.payloadCreated = true

	var p internal.PayloadV1
	p.Type = internal.CallerType
	p.Account = txn.Reply.AccountID
	p.App = txn.Reply.AppID
	p.ID = txn.ID
	p.Trip = txn.TripID()
	p.Priority = txn.Priority.Input()
	p.Sequence = txn.sequence()
	p.Depth = txn.Depth() + 1
	p.Time = time.Now()
	p.TimeMS = internal.TimeToUnixMilliseconds(p.Time)
	if u != nil {
		p.Host = internal.HostFromURL(u)
	}

	return p
}

var (
	errOutboundPayloadCreated = errors.New("outbound payload already created")
	errPayloadUnknownType     = errors.New("payload of unknown type")
)

func (txn *txn) AcceptDistributedTracePayload(t TransportType, p interface{}) error {
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return errAlreadyEnded
	}

	if txn.payloadCreated {
		return errOutboundPayloadCreated
	}

	payload, err := internal.AcceptPayload(p)
	if nil != payload {
		if e := internal.ValidateInboundAccountID(txn.Reply, payload.Account); nil != e {
			return e
		}
		if payload.Priority != "" {
			priority, e := internal.PriorityFromInput(txn.Reply.EncodingKeyHash,
				payload.Priority)
			if nil != e {
				return e
			}
			txn.Priority = priority
		}
		txn.Inbound = payload
		txn.Inbound.TransportType = string(t)

		if txn.Start.After(payload.Time) {
			txn.Inbound.TransportDuration = txn.Start.Sub(payload.Time)
		}
	}
	return err
}
