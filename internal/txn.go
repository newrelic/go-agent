package internal

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/newrelic/go-sdk/api"
	"github.com/newrelic/go-sdk/log"
)

type TxnInput struct {
	Writer   http.ResponseWriter
	Request  *http.Request
	Config   api.Config
	Reply    *ConnectReply
	Consumer DataConsumer
}

type txn struct {
	TxnInput
	// This mutex is required since the consumer may call the public API
	// interface functions from different routines.
	sync.Mutex
	// finished indicates whether or not End() has been called.  After
	// finished has been set to true, no recording should occur.
	finished   bool
	start      time.Time
	name       string // Work in progress name
	isWeb      bool
	ignore     bool
	errors     txnErrors // Lazily initialized.
	errorsSeen uint64

	// wroteHeader prevents capturing multiple response code errors if the
	// user erroneously calls WriteHeader multiple times.
	wroteHeader bool

	// Fields assigned at completion
	stop           time.Time
	duration       time.Duration
	finalName      string // Full finalized metric name
	zone           ApdexZone
	apdexThreshold time.Duration
}

func NewTxn(input TxnInput, name string) *txn {
	return &txn{
		TxnInput: input,
		start:    time.Now(),
		name:     name,
		isWeb:    nil != input.Request,
	}
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

	txn.finalName = CreateFullTxnName(txn.name, txn.Reply, txn.isWeb)
	if "" == txn.finalName {
		txn.ignore = true
	}
}

func (txn *txn) getsApdex() bool {
	return txn.isWeb
}

func (txn *txn) MergeIntoHarvest(h *Harvest) {
	h.CreateTxnMetrics(CreateTxnMetricsArgs{
		IsWeb:          txn.isWeb,
		Duration:       txn.duration,
		Name:           txn.finalName,
		Zone:           txn.zone,
		ApdexThreshold: txn.apdexThreshold,
		ErrorsSeen:     txn.errorsSeen,
	})

	if txn.txnEventsEnabled() {
		event := CreateTxnEvent(txn.zone, txn.finalName, txn.duration, txn.start)
		h.AddTxnEvent(event)
	}

	requestURI := ""
	if nil != txn.Request && nil != txn.Request.URL {
		requestURI = safeURL(txn.Request.URL)
	}

	h.MergeErrors(txn.errors, txn.finalName, requestURI)

	if txn.errorEventsEnabled() {
		h.CreateErrorEvents(txn.errors, txn.finalName, txn.duration)
	}
}

func responseCodeIsError(cfg *api.Config, code int) bool {
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

func (txn *txn) Header() http.Header { return txn.Writer.Header() }

func (txn *txn) Write(b []byte) (int, error) {
	n, err := txn.Writer.Write(b)

	txn.Lock()
	defer txn.Unlock()

	if !txn.finished {
		txn.wroteHeader = true
	}

	return n, err
}

func (txn *txn) WriteHeader(code int) {
	txn.Writer.WriteHeader(code)

	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return
	}
	if txn.wroteHeader {
		return
	}
	txn.wroteHeader = true

	if !responseCodeIsError(&txn.Config, code) {
		return
	}

	e := txnErrorFromResponseCode(code)
	e.stack = GetStackTrace(0)
	txn.noticeErrorInternal(e)
}

var (
	AlreadyEndedErr = errors.New("transaction has already ended")
)

func (txn *txn) End() error {
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return AlreadyEndedErr
	}

	txn.finished = true

	r := recover()
	if nil != r {
		e := txnErrorFromPanic(r)
		e.stack = GetStackTrace(0)
		txn.noticeErrorInternal(e)
	}

	txn.stop = time.Now()
	txn.duration = txn.stop.Sub(txn.start)

	txn.freezeName()
	if txn.getsApdex() {
		txn.apdexThreshold = calculateApdexThreshold(txn.Reply, txn.finalName)
		if txn.errorsSeen > 0 {
			txn.zone = ApdexFailing
		} else {
			txn.zone = calculateApdexZone(txn.apdexThreshold, txn.duration)
		}
	} else {
		txn.zone = ApdexNone
	}

	// This logging adds roughly 4 allocations per transaction.
	log.Debug("transaction ended", log.Context{
		"name":        txn.finalName,
		"duration_ms": txn.duration.Seconds() * 1000.0,
	})

	if !txn.ignore {
		txn.Consumer.Consume(txn.Reply.RunID, txn)
	}

	// Note that if a consumer uses `panic(nil)`, the panic will not
	// propogate.
	if nil != r {
		panic(r)
	}

	return nil
}

var (
	ErrorsLocallyDisabled  = errors.New("errors locally disabled")
	ErrorsRemotelyDisabled = errors.New("errors remotely disabled")
	NilError               = errors.New("nil error")
)

const (
	HighSecurityErrorMsg = "message removed by high security setting"
)

func (txn *txn) noticeErrorInternal(err txnError) error {
	// Increment errorsSeen even if errors are disabled:  Error metrics do
	// not depend on whether or not errors are enabled.
	txn.errorsSeen++

	if !txn.Config.ErrorCollector.Enabled {
		return ErrorsLocallyDisabled
	}

	if !txn.Reply.CollectErrors {
		return ErrorsRemotelyDisabled
	}

	if nil == txn.errors {
		txn.errors = newTxnErrors(MaxTxnErrors)
	}

	if txn.Config.HighSecurity {
		err.msg = HighSecurityErrorMsg
	}

	err.when = time.Now()

	txn.errors.Add(&err)

	return nil
}

func (txn *txn) NoticeError(err error) error {
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return AlreadyEndedErr
	}

	if nil == err {
		return NilError
	}

	e := txnErrorFromError(err)
	e.stack = GetStackTrace(1)
	return txn.noticeErrorInternal(e)
}

func (txn *txn) SetName(name string) error {
	txn.Lock()
	defer txn.Unlock()

	if txn.finished {
		return AlreadyEndedErr
	}

	txn.name = name
	return nil
}
