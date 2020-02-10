package newrelic

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"sync"
	"time"

	"github.com/newrelic/go-agent/v3/internal"
)

type txn struct {
	app *app
	*appRun

	// This mutex is required since the consumer may call the public API
	// interface functions from different routines.
	sync.Mutex
	// finished indicates whether or not End() has been called.  After
	// finished has been set to true, no recording should occur.
	finished           bool
	numPayloadsCreated uint32
	sampledCalculated  bool

	ignore bool

	// wroteHeader prevents capturing multiple response code errors if the
	// user erroneously calls WriteHeader multiple times.
	wroteHeader bool

	internal.TxnData

	mainThread   internal.Thread
	asyncThreads []*internal.Thread
}

type thread struct {
	*txn
	// thread does not have locking because it should only be accessed while
	// the txn is locked.
	thread *internal.Thread
}

func (txn *txn) markStart(now time.Time) {
	txn.Start = now
	// The mainThread is considered active now.
	txn.mainThread.RecordActivity(now)

}

func (txn *txn) markEnd(now time.Time, thread *internal.Thread) {
	txn.Stop = now
	// The thread on which End() was called is considered active now.
	thread.RecordActivity(now)
	txn.Duration = txn.Stop.Sub(txn.Start)

	// TotalTime is the sum of "active time" across all threads.  A thread
	// was active when it started the transaction, stopped the transaction,
	// started a segment, or stopped a segment.
	txn.TotalTime = txn.mainThread.TotalTime()
	for _, thd := range txn.asyncThreads {
		txn.TotalTime += thd.TotalTime()
	}
	// Ensure that TotalTime is at least as large as Duration so that the
	// graphs look sensible.  This can happen under the following situation:
	// goroutine1: txn.start----|segment1|
	// goroutine2:                                   |segment2|----txn.end
	if txn.Duration > txn.TotalTime {
		txn.TotalTime = txn.Duration
	}
}

func newTxn(app *app, run *appRun, name string) *thread {
	txn := &txn{
		app:    app,
		appRun: run,
	}
	txn.markStart(time.Now())

	txn.Name = name
	txn.Attrs = internal.NewAttributes(run.AttributeConfig)

	if run.Config.DistributedTracer.Enabled {
		txn.BetterCAT.Enabled = true
		txn.TraceIDGenerator = run.Reply.TraceIDGenerator
		txn.BetterCAT.SetTraceAndTxnIDs(txn.TraceIDGenerator.GenerateTraceID())
		txn.BetterCAT.Priority = txn.TraceIDGenerator.GeneratePriority()
		txn.ShouldCollectSpanEvents = txn.shouldCollectSpanEvents
	}

	txn.Attrs.Agent.Add(internal.AttributeHostDisplayName, txn.Config.HostDisplayName, nil)
	txn.TxnTrace.Enabled = txn.Config.TransactionTracer.Enabled
	txn.TxnTrace.SegmentThreshold = txn.Config.TransactionTracer.Segments.Threshold
	txn.StackTraceThreshold = txn.Config.TransactionTracer.Segments.StackTraceThreshold
	txn.SlowQueriesEnabled = txn.Config.DatastoreTracer.SlowQuery.Enabled
	txn.SlowQueryThreshold = txn.Config.DatastoreTracer.SlowQuery.Threshold

	// Synthetics support is tied up with a transaction's Old CAT field,
	// CrossProcess. To support Synthetics with either BetterCAT or Old CAT,
	// Initialize the CrossProcess field of the transaction, passing in
	// the top-level configuration.
	doOldCAT := txn.Config.CrossApplicationTracer.Enabled
	noGUID := txn.Config.DistributedTracer.Enabled
	txn.CrossProcess.Init(doOldCAT, noGUID, run.Reply)

	return &thread{
		txn:    txn,
		thread: &txn.mainThread,
	}
}

func (thd *thread) logAPIError(err error, operation string, extraDetails map[string]interface{}) {
	if nil == thd {
		return
	}
	if nil == err {
		return
	}
	if extraDetails == nil {
		extraDetails = make(map[string]interface{}, 1)
	}
	extraDetails["reason"] = err.Error()
	thd.Config.Logger.Error("unable to "+operation, extraDetails)
}

func (txn *txn) shouldCollectSpanEvents() bool {
	if !txn.Config.DistributedTracer.Enabled {
		return false
	}
	if !txn.Config.SpanEvents.Enabled {
		return false
	}
	return txn.lazilyCalculateSampled()
}

// lazilyCalculateSampled calculates and returns whether or not the transaction
// should be sampled.  Sampled is not computed at the beginning of the
// transaction because we want to calculate Sampled only for transactions that
// do not accept an inbound payload.
func (txn *txn) lazilyCalculateSampled() bool {
	if !txn.BetterCAT.Enabled {
		return false
	}
	if txn.sampledCalculated {
		return txn.BetterCAT.Sampled
	}
	txn.BetterCAT.Sampled = txn.appRun.adaptiveSampler.computeSampled(txn.BetterCAT.Priority.Float32(), time.Now())
	if txn.BetterCAT.Sampled {
		txn.BetterCAT.Priority += 1.0
	}
	txn.sampledCalculated = true
	return txn.BetterCAT.Sampled
}

func (txn *txn) SetWebRequest(r WebRequest) error {
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return errAlreadyEnded
	}

	// Any call to SetWebRequest should indicate a web transaction.
	txn.IsWeb = true

	h := r.Header
	if nil != h {
		txn.Queuing = queueDuration(h, txn.Start)
		txn.acceptDistributedTraceHeadersLocked(r.Transport, h)
		txn.CrossProcess.InboundHTTPRequest(h)
	}

	internal.RequestAgentAttributes(txn.Attrs, r.Method, h, r.URL)

	return nil
}

type dummyResponseWriter struct{}

func (rw dummyResponseWriter) Header() http.Header { return nil }

func (rw dummyResponseWriter) Write(b []byte) (int, error) { return 0, nil }

func (rw dummyResponseWriter) WriteHeader(code int) {}

func (thd *thread) SetWebResponse(w http.ResponseWriter) http.ResponseWriter {
	txn := thd.txn
	txn.Lock()
	defer txn.Unlock()

	if w == nil {
		// Accepting a nil parameter makes it easy for consumers to add
		// a response code to the transaction without a response
		// writer:
		//
		//    txn.SetWebResponse(nil).WriteHeader(500)
		//
		w = dummyResponseWriter{}
	}

	return upgradeResponseWriter(&replacementResponseWriter{
		txn:      txn,
		original: w,
	})
}

func (txn *txn) freezeName() {
	if txn.ignore || ("" != txn.FinalName) {
		return
	}
	txn.FinalName = txn.appRun.createTransactionName(txn.Name, txn.IsWeb)
	if "" == txn.FinalName {
		txn.ignore = true
	}
}

func (txn *txn) getsApdex() bool {
	return txn.IsWeb
}

func (txn *txn) shouldSaveTrace() bool {
	if !txn.Config.TransactionTracer.Enabled {
		return false
	}
	if txn.CrossProcess.IsSynthetics() {
		return true
	}
	return txn.Duration >= txn.txnTraceThreshold(txn.ApdexThreshold)
}

func (txn *txn) MergeIntoHarvest(h *internal.Harvest) {

	var priority internal.Priority
	if txn.BetterCAT.Enabled {
		priority = txn.BetterCAT.Priority
	} else {
		priority = internal.NewPriority()
	}

	internal.CreateTxnMetrics(&txn.TxnData, h.Metrics)
	internal.MergeBreakdownMetrics(&txn.TxnData, h.Metrics)

	if txn.Config.TransactionEvents.Enabled {
		// Allocate a new TxnEvent to prevent a reference to the large transaction.
		alloc := new(internal.TxnEvent)
		*alloc = txn.TxnData.TxnEvent
		h.TxnEvents.AddTxnEvent(alloc, priority)
	}

	if txn.Reply.CollectErrors {
		internal.MergeTxnErrors(&h.ErrorTraces, txn.Errors, txn.TxnEvent)
	}

	if txn.Config.ErrorCollector.CaptureEvents {
		for _, e := range txn.Errors {
			errEvent := &internal.ErrorEvent{
				ErrorData: *e,
				TxnEvent:  txn.TxnEvent,
			}
			// Since the stack trace is not used in error events, remove the reference
			// to minimize memory.
			errEvent.Stack = nil
			h.ErrorEvents.Add(errEvent, priority)
		}
	}

	if txn.shouldSaveTrace() {
		h.TxnTraces.Witness(internal.HarvestTrace{
			TxnEvent: txn.TxnEvent,
			Trace:    txn.TxnTrace,
		})
	}

	if nil != txn.SlowQueries {
		h.SlowSQLs.Merge(txn.SlowQueries, txn.TxnEvent)
	}

	if txn.shouldCollectSpanEvents() {
		h.SpanEvents.MergeSpanEvents(txn.TxnData.SpanEvents)
	}
}

func headersJustWritten(txn *txn, code int, hdr http.Header) {
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return
	}
	if txn.wroteHeader {
		return
	}
	txn.wroteHeader = true

	internal.ResponseHeaderAttributes(txn.Attrs, hdr)
	internal.ResponseCodeAttribute(txn.Attrs, code)

	if txn.appRun.responseCodeIsError(code) {
		e := internal.TxnErrorFromResponseCode(time.Now(), code)
		e.Stack = internal.GetStackTrace()
		txn.noticeErrorInternal(e)
	}
}

func (txn *txn) responseHeader(hdr http.Header) http.Header {
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return nil
	}
	if txn.wroteHeader {
		return nil
	}
	if !txn.CrossProcess.Enabled {
		return nil
	}
	if !txn.CrossProcess.IsInbound() {
		return nil
	}
	txn.freezeName()
	contentLength := internal.GetContentLengthFromHeader(hdr)

	appData, err := txn.CrossProcess.CreateAppData(txn.FinalName, txn.Queuing, time.Since(txn.Start), contentLength)
	if err != nil {
		txn.Config.Logger.Debug("error generating outbound response header", map[string]interface{}{
			"error": err,
		})
		return nil
	}
	return internal.AppDataToHTTPHeader(appData)
}

func addCrossProcessHeaders(txn *txn, hdr http.Header) {
	// responseHeader() checks the wroteHeader field and returns a nil map if the
	// header has been written, so we don't need a check here.
	if nil != hdr {
		for key, values := range txn.responseHeader(hdr) {
			for _, value := range values {
				hdr.Add(key, value)
			}
		}
	}
}

func (thd *thread) End(recovered interface{}) error {
	txn := thd.txn
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return errAlreadyEnded
	}

	txn.finished = true

	if nil != recovered {
		e := internal.TxnErrorFromPanic(time.Now(), recovered)
		e.Stack = internal.GetStackTrace()
		txn.noticeErrorInternal(e)
	}

	txn.markEnd(time.Now(), thd.thread)
	txn.freezeName()
	// Make a sampling decision if there have been no segments or outbound
	// payloads.
	txn.lazilyCalculateSampled()

	// Finalise the CAT state.
	if err := txn.CrossProcess.Finalise(txn.Name, txn.Config.AppName); err != nil {
		txn.Config.Logger.Debug("error finalising the cross process state", map[string]interface{}{
			"error": err,
		})
	}

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
			"name":          txn.FinalName,
			"duration_ms":   txn.Duration.Seconds() * 1000.0,
			"ignored":       txn.ignore,
			"app_connected": "" != txn.Reply.RunID,
		})
	}

	if txn.shouldCollectSpanEvents() {
		root := &internal.SpanEvent{
			GUID:         txn.GetRootSpanID(),
			Timestamp:    txn.Start,
			Duration:     txn.Duration,
			Name:         txn.FinalName,
			Category:     internal.SpanCategoryGeneric,
			IsEntrypoint: true,
		}
		if nil != txn.BetterCAT.Inbound {
			root.ParentID = txn.BetterCAT.Inbound.ID
			root.TrustedParentID = txn.BetterCAT.Inbound.TrustedParentID
			root.TracingVendors = txn.BetterCAT.Inbound.TracingVendors
		}
		txn.SpanEvents = append(txn.SpanEvents, root)

		// Add transaction tracing fields to span events at the end of
		// the transaction since we could accept payload after the early
		// segments occur.
		for _, evt := range txn.SpanEvents {
			evt.TraceID = txn.BetterCAT.TraceID
			evt.TransactionID = txn.BetterCAT.TxnID
			evt.Sampled = txn.BetterCAT.Sampled
			evt.Priority = txn.BetterCAT.Priority
		}
	}

	if !txn.ignore {
		txn.app.Consume(txn.Reply.RunID, txn)
	}

	// Note that if a consumer uses `panic(nil)`, the panic will not
	// propagate.
	if nil != recovered {
		panic(recovered)
	}

	return nil
}

func (txn *txn) AddAttribute(name string, value interface{}) error {
	txn.Lock()
	defer txn.Unlock()

	if txn.Config.HighSecurity {
		return errHighSecurityEnabled
	}

	if !txn.Reply.SecurityPolicies.CustomParameters.Enabled() {
		return errSecurityPolicy
	}

	if txn.finished {
		return errAlreadyEnded
	}

	return internal.AddUserAttribute(txn.Attrs, name, value, internal.DestAll)
}

var (
	errorsDisabled        = errors.New("errors disabled")
	errNilError           = errors.New("nil error")
	errAlreadyEnded       = errors.New("transaction has already ended")
	errSecurityPolicy     = errors.New("disabled by security policy")
	errTransactionIgnored = errors.New("transaction has been ignored")
	errBrowserDisabled    = errors.New("browser disabled by local configuration")
)

const (
	highSecurityErrorMsg   = "message removed by high security setting"
	securityPolicyErrorMsg = "message removed by security policy"
)

func (txn *txn) noticeErrorInternal(err internal.ErrorData) error {
	if !txn.Config.ErrorCollector.Enabled {
		return errorsDisabled
	}

	if nil == txn.Errors {
		txn.Errors = internal.NewTxnErrors(internal.MaxTxnErrors)
	}

	if txn.Config.HighSecurity {
		err.Msg = highSecurityErrorMsg
	}

	if !txn.Reply.SecurityPolicies.AllowRawExceptionMessages.Enabled() {
		err.Msg = securityPolicyErrorMsg
	}

	txn.Errors.Add(err)
	txn.TxnData.TxnEvent.HasError = true //mark transaction as having an error
	return nil
}

var (
	errTooManyErrorAttributes = fmt.Errorf("too many extra attributes: limit is %d",
		internal.AttributeErrorLimit)
)

// errorCause returns the error's deepest wrapped ancestor.
func errorCause(err error) error {
	for {
		if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
			if next := unwrapper.Unwrap(); nil != next {
				err = next
				continue
			}
		}
		return err
	}
}

func errorClassMethod(err error) string {
	if ec, ok := err.(errorClasser); ok {
		return ec.ErrorClass()
	}
	return ""
}

func errorStackTraceMethod(err error) internal.StackTrace {
	if st, ok := err.(stackTracer); ok {
		return st.StackTrace()
	}
	return nil
}

func errorAttributesMethod(err error) map[string]interface{} {
	if st, ok := err.(errorAttributer); ok {
		return st.ErrorAttributes()
	}
	return nil
}

func errDataFromError(input error) (data internal.ErrorData, err error) {
	cause := errorCause(input)

	data = internal.ErrorData{
		When: time.Now(),
		Msg:  input.Error(),
	}

	if c := errorClassMethod(input); "" != c {
		// If the error implements ErrorClasser, use that.
		data.Klass = c
	} else if c := errorClassMethod(cause); "" != c {
		// Otherwise, if the error's cause implements ErrorClasser, use that.
		data.Klass = c
	} else {
		// As a final fallback, use the type of the error's cause.
		data.Klass = reflect.TypeOf(cause).String()
	}

	if st := errorStackTraceMethod(input); nil != st {
		// If the error implements StackTracer, use that.
		data.Stack = st
	} else if st := errorStackTraceMethod(cause); nil != st {
		// Otherwise, if the error's cause implements StackTracer, use that.
		data.Stack = st
	} else {
		// As a final fallback, generate a StackTrace here.
		data.Stack = internal.GetStackTrace()
	}

	var unvetted map[string]interface{}
	if ats := errorAttributesMethod(input); nil != ats {
		// If the error implements ErrorAttributer, use that.
		unvetted = ats
	} else {
		// Otherwise, if the error's cause implements ErrorAttributer, use that.
		unvetted = errorAttributesMethod(cause)
	}
	if unvetted != nil {
		if len(unvetted) > internal.AttributeErrorLimit {
			err = errTooManyErrorAttributes
			return
		}

		data.ExtraAttributes = make(map[string]interface{})
		for key, val := range unvetted {
			val, err = internal.ValidateUserAttribute(key, val)
			if nil != err {
				return
			}
			data.ExtraAttributes[key] = val
		}
	}

	return data, nil
}

func (txn *txn) NoticeError(input error) error {
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return errAlreadyEnded
	}

	if nil == input {
		return errNilError
	}

	data, err := errDataFromError(input)
	if nil != err {
		return err
	}

	if txn.Config.HighSecurity || !txn.Reply.SecurityPolicies.CustomParameters.Enabled() {
		data.ExtraAttributes = nil
	}

	return txn.noticeErrorInternal(data)
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

func (thd *thread) StartSegmentNow() SegmentStartTime {
	var s internal.SegmentStartTime
	txn := thd.txn
	txn.Lock()
	if !txn.finished {
		s = internal.StartSegment(&txn.TxnData, thd.thread, time.Now())
	}
	txn.Unlock()
	return SegmentStartTime{
		segment: segment{
			start:  s,
			thread: thd,
		},
	}
}

const (
	// Browser fields are encoded using the first digits of the license
	// key.
	browserEncodingKeyLimit = 13
)

func browserEncodingKey(licenseKey string) []byte {
	key := []byte(licenseKey)
	if len(key) > browserEncodingKeyLimit {
		key = key[0:browserEncodingKeyLimit]
	}
	return key
}

func (txn *txn) BrowserTimingHeader() (*BrowserTimingHeader, error) {
	txn.Lock()
	defer txn.Unlock()

	if !txn.Config.BrowserMonitoring.Enabled {
		return nil, errBrowserDisabled
	}

	if txn.Reply.AgentLoader == "" {
		// If the loader is empty, either browser has been disabled
		// by the server or the application is not yet connected.
		return nil, nil
	}

	if txn.finished {
		return nil, errAlreadyEnded
	}

	txn.freezeName()

	// Freezing the name might cause the transaction to be ignored, so check
	// this after txn.freezeName().
	if txn.ignore {
		return nil, errTransactionIgnored
	}

	encodingKey := browserEncodingKey(txn.Config.License)

	attrs, err := internal.Obfuscate(internal.BrowserAttributes(txn.Attrs), encodingKey)
	if err != nil {
		return nil, fmt.Errorf("error getting browser attributes: %v", err)
	}

	name, err := internal.Obfuscate([]byte(txn.FinalName), encodingKey)
	if err != nil {
		return nil, fmt.Errorf("error obfuscating name: %v", err)
	}

	return &BrowserTimingHeader{
		agentLoader: txn.Reply.AgentLoader,
		info: browserInfo{
			Beacon:                txn.Reply.Beacon,
			LicenseKey:            txn.Reply.BrowserKey,
			ApplicationID:         txn.Reply.AppID,
			TransactionName:       name,
			QueueTimeMillis:       txn.Queuing.Nanoseconds() / (1000 * 1000),
			ApplicationTimeMillis: time.Now().Sub(txn.Start).Nanoseconds() / (1000 * 1000),
			ObfuscatedAttributes:  attrs,
			ErrorBeacon:           txn.Reply.ErrorBeacon,
			Agent:                 txn.Reply.JSAgentFile,
		},
	}, nil
}

func createThread(txn *txn) *internal.Thread {
	newThread := internal.NewThread(&txn.TxnData)
	txn.asyncThreads = append(txn.asyncThreads, newThread)
	return newThread
}

func (thd *thread) NewGoroutine() *Transaction {
	txn := thd.txn
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		// If the transaction has finished, return the same thread.
		return newTransaction(thd)
	}
	return newTransaction(&thread{
		thread: createThread(txn),
		txn:    txn,
	})
}

type segment struct {
	start  internal.SegmentStartTime
	thread *thread
}

func endSegment(s *Segment) error {
	thd := s.StartTime.thread
	if nil == thd {
		return nil
	}
	txn := thd.txn
	var err error
	txn.Lock()
	if txn.finished {
		err = errAlreadyEnded
	} else {
		err = internal.EndBasicSegment(&txn.TxnData, thd.thread, s.StartTime.start, time.Now(), s.Name)
	}
	txn.Unlock()
	return err
}

func endDatastore(s *DatastoreSegment) error {
	thd := s.StartTime.thread
	if nil == thd {
		return nil
	}
	txn := thd.txn
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
	if txn.Reply.SecurityPolicies.RecordSQL.IsSet() {
		s.QueryParameters = nil
		if !txn.Reply.SecurityPolicies.RecordSQL.Enabled() {
			s.ParameterizedQuery = ""
		}
	}
	if !txn.Config.DatastoreTracer.DatabaseNameReporting.Enabled {
		s.DatabaseName = ""
	}
	if !txn.Config.DatastoreTracer.InstanceReporting.Enabled {
		s.Host = ""
		s.PortPathOrID = ""
	}
	return internal.EndDatastoreSegment(internal.EndDatastoreParams{
		TxnData:            &txn.TxnData,
		Thread:             thd.thread,
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
		ThisHost:           txn.appRun.Config.hostname,
	})
}

func externalSegmentMethod(s *ExternalSegment) string {
	if "" != s.Procedure {
		return s.Procedure
	}
	r := s.Request
	if nil != s.Response && nil != s.Response.Request {
		r = s.Response.Request
	}

	if nil != r {
		if "" != r.Method {
			return r.Method
		}
		// Golang's http package states that when a client's Request has
		// an empty string for Method, the method is GET.
		return "GET"
	}

	return ""
}

func externalSegmentURL(s *ExternalSegment) (*url.URL, error) {
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

func endExternal(s *ExternalSegment) error {
	thd := s.StartTime.thread
	if nil == thd {
		return nil
	}
	txn := thd.txn
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return errAlreadyEnded
	}
	u, err := externalSegmentURL(s)
	if nil != err {
		return err
	}
	return internal.EndExternalSegment(internal.EndExternalParams{
		TxnData:  &txn.TxnData,
		Thread:   thd.thread,
		Start:    s.StartTime.start,
		Now:      time.Now(),
		Logger:   txn.Config.Logger,
		Response: s.Response,
		URL:      u,
		Host:     s.Host,
		Library:  s.Library,
		Method:   externalSegmentMethod(s),
	})
}

func endMessage(s *MessageProducerSegment) error {
	thd := s.StartTime.thread
	if nil == thd {
		return nil
	}
	txn := thd.txn
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return errAlreadyEnded
	}

	if "" == s.DestinationType {
		s.DestinationType = MessageQueue
	}

	return internal.EndMessageSegment(internal.EndMessageParams{
		TxnData:         &txn.TxnData,
		Thread:          thd.thread,
		Start:           s.StartTime.start,
		Now:             time.Now(),
		Library:         s.Library,
		Logger:          txn.Config.Logger,
		DestinationName: s.DestinationName,
		DestinationType: string(s.DestinationType),
		DestinationTemp: s.DestinationTemporary,
	})
}

// oldCATOutboundHeaders generates the Old CAT and Synthetics headers, depending
// on whether Old CAT is enabled or any Synthetics functionality has been
// triggered in the agent.
func oldCATOutboundHeaders(txn *txn) http.Header {
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return http.Header{}
	}

	metadata, err := txn.CrossProcess.CreateCrossProcessMetadata(txn.Name, txn.Config.AppName)
	if err != nil {
		txn.Config.Logger.Debug("error generating outbound headers", map[string]interface{}{
			"error": err,
		})

		// It's possible for CreateCrossProcessMetadata() to error and still have a
		// Synthetics header, so we'll still fall through to returning headers
		// based on whatever metadata was returned.
	}

	return internal.MetadataToHTTPHeader(metadata)
}

func outboundHeaders(s *ExternalSegment) http.Header {
	thd := s.StartTime.thread

	if nil == thd {
		return http.Header{}
	}
	txn := thd.txn
	hdr := oldCATOutboundHeaders(txn)

	// hdr may be empty, or it may contain headers.  If DistributedTracer
	// is enabled, add more to the existing hdr
	thd.CreateDistributedTracePayload(hdr)

	return hdr
}

const (
	maxSampledDistributedPayloads = 35
)

func (thd *thread) CreateDistributedTracePayload(hdrs http.Header) {
	txn := thd.txn
	txn.Lock()
	defer txn.Unlock()

	if !txn.BetterCAT.Enabled {
		return
	}

	support := &txn.DistributedTracingSupport

	excludeNRHeader := thd.Config.DistributedTracer.ExcludeNewRelicHeader
	if txn.finished {
		support.TraceContextCreateException = true
		if !excludeNRHeader {
			support.CreatePayloadException = true
		}
		return
	}

	if "" == txn.Reply.AccountID || "" == txn.Reply.TrustedAccountKey {
		// We can't create a payload:  The application is not yet
		// connected or serverless distributed tracing configuration was
		// not provided.
		return
	}

	txn.numPayloadsCreated++

	p := &internal.Payload{}

	// Calculate sampled first since this also changes the value for the
	// priority
	sampled := txn.lazilyCalculateSampled()
	if txn.shouldCollectSpanEvents() {
		p.ID = txn.CurrentSpanIdentifier(thd.thread)
	}

	p.Type = internal.CallerTypeApp
	p.Account = txn.Reply.AccountID
	p.App = txn.Reply.PrimaryAppID
	p.TracedID = txn.BetterCAT.TraceID
	p.Priority = txn.BetterCAT.Priority
	p.Timestamp.Set(txn.Reply.DistributedTraceTimestampGenerator())
	p.TrustedAccountKey = txn.Reply.TrustedAccountKey
	p.TransactionID = txn.BetterCAT.TxnID // Set the transaction ID to the transaction guid.
	if nil != txn.BetterCAT.Inbound {
		p.NonTrustedTraceState = txn.BetterCAT.Inbound.NonTrustedTraceState
		p.OriginalTraceState = txn.BetterCAT.Inbound.OriginalTraceState
	}

	// limit the number of outbound sampled=true payloads to prevent too
	// many downstream sampled events.
	p.SetSampled(false)
	if txn.numPayloadsCreated < maxSampledDistributedPayloads {
		p.SetSampled(sampled)
	}

	support.TraceContextCreateSuccess = true

	if !excludeNRHeader {
		hdrs.Set(internal.DistributedTraceNewRelicHeader, p.NRHTTPSafe())
		support.CreatePayloadSuccess = true
	}

	// ID must be present in the Traceparent header when span events are
	// enabled, even if the transaction is not sampled.  Note that this
	// assignment occurs after setting the Newrelic header since the ID
	// field of the Newrelic header should be empty if span events are
	// disabled or the transaction is not sampled.
	if p.ID == "" {
		p.ID = txn.CurrentSpanIdentifier(thd.thread)
	}
	hdrs.Set(internal.DistributedTraceW3CTraceParentHeader, p.W3CTraceParent())

	if !txn.Config.SpanEvents.Enabled {
		p.ID = ""
	}
	if !txn.Config.TransactionEvents.Enabled {
		p.TransactionID = ""
	}
	hdrs.Set(internal.DistributedTraceW3CTraceStateHeader, p.W3CTraceState())
}

var (
	errOutboundPayloadCreated   = errors.New("outbound payload already created")
	errAlreadyAccepted          = errors.New("AcceptDistributedTraceHeaders has already been called")
	errInboundPayloadDTDisabled = errors.New("DistributedTracer must be enabled to accept an inbound payload")
	errTrustedAccountKey        = errors.New("trusted account key missing or does not match")
)

func (txn *txn) AcceptDistributedTraceHeaders(t TransportType, hdrs http.Header) error {
	txn.Lock()
	defer txn.Unlock()

	return txn.acceptDistributedTraceHeadersLocked(t, hdrs)
}

func (txn *txn) acceptDistributedTraceHeadersLocked(t TransportType, hdrs http.Header) error {

	if !txn.BetterCAT.Enabled {
		return errInboundPayloadDTDisabled
	}

	if txn.finished {
		return errAlreadyEnded
	}

	support := &txn.DistributedTracingSupport

	if txn.numPayloadsCreated > 0 {
		support.AcceptPayloadCreateBeforeAccept = true
		return errOutboundPayloadCreated
	}

	if txn.BetterCAT.Inbound != nil {
		support.AcceptPayloadIgnoredMultiple = true
		return errAlreadyAccepted
	}

	if nil == hdrs {
		support.AcceptPayloadNullPayload = true
		return nil
	}

	if "" == txn.Reply.AccountID || "" == txn.Reply.TrustedAccountKey {
		// We can't accept a payload:  The application is not yet
		// connected or serverless distributed tracing configuration was
		// not provided.
		return nil
	}

	txn.BetterCAT.TransportType = t.toString()

	payload, err := internal.AcceptPayload(hdrs, txn.Reply.TrustedAccountKey, support)
	if nil != err {
		return err
	}

	if nil == payload {
		return nil
	}

	// and let's also do our trustedKey check
	receivedTrustKey := payload.TrustedAccountKey
	if "" == receivedTrustKey {
		receivedTrustKey = payload.Account
	}

	// If the trust key doesn't match but we don't have any New Relic trace info, this means
	// we just got the TraceParent header, and we still need to save that info to BetterCAT
	// farther down.
	if receivedTrustKey != txn.Reply.TrustedAccountKey && payload.HasNewRelicTraceInfo {
		support.AcceptPayloadUntrustedAccount = true
		return errTrustedAccountKey
	}

	if 0 != payload.Priority {
		txn.BetterCAT.Priority = payload.Priority
	}

	// a nul payload.Sampled means the a field wasn't provided
	if nil != payload.Sampled {
		txn.BetterCAT.Sampled = *payload.Sampled
		txn.sampledCalculated = true
	}

	txn.BetterCAT.Inbound = payload
	txn.BetterCAT.TraceID = payload.TracedID

	if tm := payload.Timestamp.Time(); txn.Start.After(tm) {
		txn.BetterCAT.Inbound.TransportDuration = txn.Start.Sub(tm)
	}

	return nil
}

func (txn *txn) Application() *Application {
	return newApplication(txn.app)
}

func (thd *thread) AddAgentSpanAttribute(key internal.SpanAttribute, val string) {
	thd.thread.AddAgentSpanAttribute(key, val)
}

var (
	// Ensure that txn implements AddAgentAttributer to avoid breaking
	// integration package type assertions.
	_ internal.AddAgentAttributer = &txn{}
)

func (txn *txn) AddAgentAttribute(id internal.AgentAttributeID, stringVal string, otherVal interface{}) {
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return
	}
	txn.Attrs.Agent.Add(id, stringVal, otherVal)
}

func (thd *thread) GetTraceMetadata() (metadata TraceMetadata) {
	txn := thd.txn
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return
	}

	if txn.BetterCAT.Enabled {
		metadata.TraceID = txn.BetterCAT.TraceID
		if txn.shouldCollectSpanEvents() {
			metadata.SpanID = txn.CurrentSpanIdentifier(thd.thread)
		}
	}

	return
}

func (thd *thread) GetLinkingMetadata() (metadata LinkingMetadata) {
	txn := thd.txn
	metadata.EntityName = txn.appRun.firstAppName
	metadata.EntityType = "SERVICE"
	metadata.EntityGUID = txn.appRun.Reply.EntityGUID
	metadata.Hostname = txn.appRun.Config.hostname

	md := thd.GetTraceMetadata()
	metadata.TraceID = md.TraceID
	metadata.SpanID = md.SpanID

	return
}

func (txn *txn) IsSampled() bool {
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return false
	}

	return txn.lazilyCalculateSampled()
}
