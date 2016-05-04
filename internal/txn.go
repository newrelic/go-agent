package internal

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"go.datanerd.us/p/will/go-sdk/api"
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

// TODO: Investigate this:  Don't some agents have apdex for background
// transactions?
func (txn *txn) getsApdex() bool {
	return txn.isWeb
}

func (txn *txn) end() {
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
	h.MergeErrors(txn.errors, txn.finalName, requestURI)

	if txn.errorEventsEnabled() {
		h.CreateErrorEvents(txn.errors, txn.finalName, txn.duration)
	}
}

func (txn *txn) Header() http.Header         { return txn.Writer.Header() }
func (txn *txn) Write(b []byte) (int, error) { return txn.Writer.Write(b) }
func (txn *txn) WriteHeader(code int)        { txn.Writer.WriteHeader(code) }

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
		stack := GetStackTrace(0)
		err := PanicValueToError(r)
		txn.noticeErrorInternal(err, stack)
	}

	txn.end()

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

func (txn *txn) noticeErrorInternal(err error, stack *StackTrace) error {
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

	txn.errors.Add(newTxnError(txn.Config.HighSecurity, err, stack, time.Now()))

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

	stack := GetStackTrace(1)

	return txn.noticeErrorInternal(err, stack)
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
