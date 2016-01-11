package internal

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"
)

type panicError struct {
	v interface{}
}

func (p panicError) Error() string {
	return fmt.Sprintf("%v", p.v)
}

func PanicValueToError(v interface{}) error {
	switch val := v.(type) {
	case error:
		return val
	default:
		return panicError{v: v}
	}
}

type txnError struct {
	when  time.Time
	stack *StackTrace
	msg   string
	klass string
}

const (
	HighSecurityErrorMsg = "message removed by high security setting"
)

func newTxnError(highSecurity bool, err error, stack *StackTrace, now time.Time) *txnError {
	var msg string
	if highSecurity {
		msg = HighSecurityErrorMsg
	} else {
		msg = err.Error()
	}

	return &txnError{
		when:  now,
		stack: stack,
		msg:   msg,
		klass: reflect.TypeOf(err).String(),
	}
}

type txnErrors []*txnError

func newTxnErrors(max int) txnErrors {
	return make([]*txnError, 0, max)
}

func (errors *txnErrors) Add(e *txnError) {
	// QUESTION: Which errors should we take?  Last in wins?  Priority
	// system?
	if len(*errors) < cap(*errors) {
		*errors = append(*errors, e)
	}
}

func (h *harvestError) MarshalJSON() ([]byte, error) {
	return json.Marshal(
		[]interface{}{
			timeToFloatMilliseconds(h.when),
			h.txnName,
			h.msg,
			h.klass,
			struct {
				Stack      *StackTrace `json:"stack_trace"`
				Agent      struct{}    `json:"agentAttributes"`
				User       struct{}    `json:"userAttributes"`
				Intrinsics struct{}    `json:"intrinsics"`
				RequestURI string      `json:"request_uri,omitempty"`
			}{
				Stack:      h.stack,
				RequestURI: h.requestURI,
			},
		})
}

func (e *txnError) toHarvestError(txnName string, requestURI string) *harvestError {
	return &harvestError{
		txnError:   *e,
		txnName:    txnName,
		requestURI: requestURI,
	}
}

type harvestError struct {
	txnError
	txnName    string
	requestURI string
}

type harvestErrors struct {
	errors []*harvestError
}

func newHarvestErrors(max int) *harvestErrors {
	return &harvestErrors{
		errors: make([]*harvestError, 0, max),
	}
}

func (errors *harvestErrors) merge(errs txnErrors, txnName string, requestURI string) {
	for _, e := range errs {
		if len(errors.errors) == cap(errors.errors) {
			return
		}

		errors.errors = append(errors.errors, e.toHarvestError(txnName, requestURI))
	}
}

func (errors *harvestErrors) Data(agentRunID string, harvestStart time.Time) ([]byte, error) {
	if 0 == len(errors.errors) {
		return nil, nil
	}
	return json.Marshal([]interface{}{agentRunID, errors.errors})
}

func (errors *harvestErrors) MergeIntoHarvest(h *Harvest) {}
