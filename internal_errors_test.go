// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"encoding/json"
	"runtime"
	"strconv"
	"testing"

	"github.com/newrelic/go-agent/internal"
)

type myError struct{}

func (e myError) Error() string { return "my msg" }

func TestNoticeErrorBackground(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.NoticeError(myError{})
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "OtherTransaction/Go/hello",
		Msg:     "my msg",
		Klass:   "newrelic.myError",
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "newrelic.myError",
			"error.message":   "my msg",
			"transactionName": "OtherTransaction/Go/hello",
		},
	}})
	app.ExpectMetrics(t, backgroundErrorMetrics)
}

func TestNoticeErrorWeb(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, helloRequest)
	err := txn.NoticeError(myError{})
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "WebTransaction/Go/hello",
		Msg:     "my msg",
		Klass:   "newrelic.myError",
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "newrelic.myError",
			"error.message":   "my msg",
			"transactionName": "WebTransaction/Go/hello",
		},
		AgentAttributes: helloRequestAttributes,
	}})
	app.ExpectMetrics(t, webErrorMetrics)
}

func TestNoticeErrorTxnEnded(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	txn.End()
	err := txn.NoticeError(myError{})
	if err != errAlreadyEnded {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{})
	app.ExpectErrorEvents(t, []internal.WantEvent{})
	app.ExpectMetrics(t, backgroundMetrics)
}

func TestNoticeErrorHighSecurity(t *testing.T) {
	cfgFn := func(cfg *Config) { cfg.HighSecurity = true }
	app := testApp(nil, cfgFn, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.NoticeError(myError{})
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "OtherTransaction/Go/hello",
		Msg:     highSecurityErrorMsg,
		Klass:   "newrelic.myError",
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "newrelic.myError",
			"error.message":   highSecurityErrorMsg,
			"transactionName": "OtherTransaction/Go/hello",
		},
	}})
	app.ExpectMetrics(t, backgroundErrorMetrics)
}

func TestNoticeErrorMessageSecurityPolicy(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) { reply.SecurityPolicies.AllowRawExceptionMessages.SetEnabled(false) }
	app := testApp(replyfn, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.NoticeError(myError{})
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "OtherTransaction/Go/hello",
		Msg:     securityPolicyErrorMsg,
		Klass:   "newrelic.myError",
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "newrelic.myError",
			"error.message":   securityPolicyErrorMsg,
			"transactionName": "OtherTransaction/Go/hello",
		},
	}})
	app.ExpectMetrics(t, backgroundErrorMetrics)
}

func TestNoticeErrorLocallyDisabled(t *testing.T) {
	cfgFn := func(cfg *Config) { cfg.ErrorCollector.Enabled = false }
	app := testApp(nil, cfgFn, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.NoticeError(myError{})
	if errorsDisabled != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{})
	app.ExpectErrorEvents(t, []internal.WantEvent{})
	app.ExpectMetrics(t, backgroundMetrics)
}

func TestErrorsDisabledByServerSideConfig(t *testing.T) {
	// Test that errors can be disabled by server-side-config.
	cfgFn := func(cfg *Config) {}
	replyfn := func(reply *internal.ConnectReply) {
		json.Unmarshal([]byte(`{"agent_config":{"error_collector.enabled":false}}`), reply)
	}
	app := testApp(replyfn, cfgFn, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.NoticeError(myError{})
	if errorsDisabled != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{})
	app.ExpectErrorEvents(t, []internal.WantEvent{})
	app.ExpectMetrics(t, backgroundMetrics)
}

func TestErrorsEnabledByServerSideConfig(t *testing.T) {
	// Test that errors can be enabled by server-side-config.
	cfgFn := func(cfg *Config) {
		cfg.ErrorCollector.Enabled = false
	}
	replyfn := func(reply *internal.ConnectReply) {
		json.Unmarshal([]byte(`{"agent_config":{"error_collector.enabled":true}}`), reply)
	}
	app := testApp(replyfn, cfgFn, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.NoticeError(myError{})
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "OtherTransaction/Go/hello",
		Msg:     "my msg",
		Klass:   "newrelic.myError",
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "newrelic.myError",
			"error.message":   "my msg",
			"transactionName": "OtherTransaction/Go/hello",
		},
		UserAttributes:  map[string]interface{}{},
		AgentAttributes: map[string]interface{}{},
	}})
	app.ExpectMetrics(t, backgroundErrorMetrics)
}

func TestNoticeErrorTracedErrorsRemotelyDisabled(t *testing.T) {
	// This tests that the connect reply field "collect_errors" controls the
	// collection of traced-errors, not error-events.
	replyfn := func(reply *internal.ConnectReply) { reply.CollectErrors = false }
	app := testApp(replyfn, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.NoticeError(myError{})
	if err != nil {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "newrelic.myError",
			"error.message":   "my msg",
			"transactionName": "OtherTransaction/Go/hello",
		},
		UserAttributes:  map[string]interface{}{},
		AgentAttributes: map[string]interface{}{},
	}})
	app.ExpectMetrics(t, backgroundErrorMetrics)
}

func TestNoticeErrorNil(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.NoticeError(nil)
	if errNilError != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{})
	app.ExpectErrorEvents(t, []internal.WantEvent{})
	app.ExpectMetrics(t, backgroundMetrics)
}

func TestNoticeErrorEventsLocallyDisabled(t *testing.T) {
	cfgFn := func(cfg *Config) { cfg.ErrorCollector.CaptureEvents = false }
	app := testApp(nil, cfgFn, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.NoticeError(myError{})
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "OtherTransaction/Go/hello",
		Msg:     "my msg",
		Klass:   "newrelic.myError",
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{})
	app.ExpectMetrics(t, backgroundErrorMetrics)
}

func TestNoticeErrorEventsRemotelyDisabled(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) { reply.CollectErrorEvents = false }
	app := testApp(replyfn, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.NoticeError(myError{})
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "OtherTransaction/Go/hello",
		Msg:     "my msg",
		Klass:   "newrelic.myError",
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{})
	app.ExpectMetrics(t, backgroundErrorMetrics)
}

type errorWithClass struct{ class string }

func (e errorWithClass) Error() string      { return "my msg" }
func (e errorWithClass) ErrorClass() string { return e.class }

func TestErrorWithClasser(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.NoticeError(errorWithClass{class: "zap"})
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "OtherTransaction/Go/hello",
		Msg:     "my msg",
		Klass:   "zap",
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "zap",
			"error.message":   "my msg",
			"transactionName": "OtherTransaction/Go/hello",
		},
	}})
	app.ExpectMetrics(t, backgroundErrorMetrics)
}

func TestErrorWithClasserReturnsEmpty(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.NoticeError(errorWithClass{class: ""})
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "OtherTransaction/Go/hello",
		Msg:     "my msg",
		Klass:   "newrelic.errorWithClass",
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "newrelic.errorWithClass",
			"error.message":   "my msg",
			"transactionName": "OtherTransaction/Go/hello",
		},
	}})
	app.ExpectMetrics(t, backgroundErrorMetrics)
}

type withStackTrace struct{ trace []uintptr }

func makeErrorWithStackTrace() error {
	callers := make([]uintptr, 20)
	written := runtime.Callers(1, callers)
	return withStackTrace{
		trace: callers[0:written],
	}
}

func (e withStackTrace) Error() string         { return "my msg" }
func (e withStackTrace) StackTrace() []uintptr { return e.trace }

func TestErrorWithStackTrace(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	e := makeErrorWithStackTrace()
	err := txn.NoticeError(e)
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "OtherTransaction/Go/hello",
		Msg:     "my msg",
		Klass:   "newrelic.withStackTrace",
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "newrelic.withStackTrace",
			"error.message":   "my msg",
			"transactionName": "OtherTransaction/Go/hello",
		},
	}})
	app.ExpectMetrics(t, backgroundErrorMetrics)
}

func TestErrorWithStackTraceReturnsNil(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	e := withStackTrace{trace: nil}
	err := txn.NoticeError(e)
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "OtherTransaction/Go/hello",
		Msg:     "my msg",
		Klass:   "newrelic.withStackTrace",
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "newrelic.withStackTrace",
			"error.message":   "my msg",
			"transactionName": "OtherTransaction/Go/hello",
		},
	}})
	app.ExpectMetrics(t, backgroundErrorMetrics)
}

func TestNewrelicErrorNoAttributes(t *testing.T) {
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.NoticeError(Error{
		Message: "my msg",
		Class:   "my class",
	})
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "OtherTransaction/Go/hello",
		Msg:     "my msg",
		Klass:   "my class",
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "my class",
			"error.message":   "my msg",
			"transactionName": "OtherTransaction/Go/hello",
		},
	}})
	app.ExpectMetrics(t, backgroundErrorMetrics)
}

func TestNewrelicErrorValidAttributes(t *testing.T) {
	extraAttributes := map[string]interface{}{
		"zip": "zap",
	}
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.NoticeError(Error{
		Message:    "my msg",
		Class:      "my class",
		Attributes: extraAttributes,
	})
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{{
		TxnName:        "OtherTransaction/Go/hello",
		Msg:            "my msg",
		Klass:          "my class",
		UserAttributes: extraAttributes,
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "my class",
			"error.message":   "my msg",
			"transactionName": "OtherTransaction/Go/hello",
		},
		UserAttributes: extraAttributes,
	}})
	app.ExpectMetrics(t, backgroundErrorMetrics)
}

func TestNewrelicErrorAttributesHighSecurity(t *testing.T) {
	extraAttributes := map[string]interface{}{
		"zip": "zap",
	}
	cfgFn := func(cfg *Config) { cfg.HighSecurity = true }
	app := testApp(nil, cfgFn, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.NoticeError(Error{
		Message:    "my msg",
		Class:      "my class",
		Attributes: extraAttributes,
	})
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{{
		TxnName:        "OtherTransaction/Go/hello",
		Msg:            "message removed by high security setting",
		Klass:          "my class",
		UserAttributes: map[string]interface{}{},
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "my class",
			"error.message":   "message removed by high security setting",
			"transactionName": "OtherTransaction/Go/hello",
		},
		UserAttributes: map[string]interface{}{},
	}})
	app.ExpectMetrics(t, backgroundErrorMetrics)
}

func TestNewrelicErrorAttributesSecurityPolicy(t *testing.T) {
	extraAttributes := map[string]interface{}{
		"zip": "zap",
	}
	replyfn := func(reply *internal.ConnectReply) { reply.SecurityPolicies.CustomParameters.SetEnabled(false) }
	app := testApp(replyfn, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.NoticeError(Error{
		Message:    "my msg",
		Class:      "my class",
		Attributes: extraAttributes,
	})
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{{
		TxnName:        "OtherTransaction/Go/hello",
		Msg:            "my msg",
		Klass:          "my class",
		UserAttributes: map[string]interface{}{},
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "my class",
			"error.message":   "my msg",
			"transactionName": "OtherTransaction/Go/hello",
		},
		UserAttributes: map[string]interface{}{},
	}})
	app.ExpectMetrics(t, backgroundErrorMetrics)
}

func TestNewrelicErrorAttributeOverridesNormalAttribute(t *testing.T) {
	extraAttributes := map[string]interface{}{
		"zip": "zap",
	}
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	if err := txn.AddAttribute("zip", 123); nil != err {
		t.Error(err)
	}
	err := txn.NoticeError(Error{
		Message:    "my msg",
		Class:      "my class",
		Attributes: extraAttributes,
	})
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{{
		TxnName:        "OtherTransaction/Go/hello",
		Msg:            "my msg",
		Klass:          "my class",
		UserAttributes: extraAttributes,
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "my class",
			"error.message":   "my msg",
			"transactionName": "OtherTransaction/Go/hello",
		},
		UserAttributes: extraAttributes,
	}})
	app.ExpectMetrics(t, backgroundErrorMetrics)
}

func TestNewrelicErrorInvalidAttributes(t *testing.T) {
	extraAttributes := map[string]interface{}{
		"zip":     "zap",
		"INVALID": struct{}{},
	}
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.NoticeError(Error{
		Message:    "my msg",
		Class:      "my class",
		Attributes: extraAttributes,
	})
	if _, ok := err.(internal.ErrInvalidAttributeType); !ok {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{})
	app.ExpectErrorEvents(t, []internal.WantEvent{})
	app.ExpectMetrics(t, backgroundMetrics)
}

func TestExtraErrorAttributeRemovedThroughConfiguration(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.ErrorCollector.Attributes.Exclude = []string{"IGNORE_ME"}
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.NoticeError(Error{
		Message: "my msg",
		Class:   "my class",
		Attributes: map[string]interface{}{
			"zip":       "zap",
			"IGNORE_ME": 123,
		},
	})
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{{
		TxnName:        "OtherTransaction/Go/hello",
		Msg:            "my msg",
		Klass:          "my class",
		UserAttributes: map[string]interface{}{"zip": "zap"},
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "my class",
			"error.message":   "my msg",
			"transactionName": "OtherTransaction/Go/hello",
		},
		UserAttributes: map[string]interface{}{"zip": "zap"},
	}})
	app.ExpectMetrics(t, backgroundErrorMetrics)

}

func TestTooManyExtraErrorAttributes(t *testing.T) {
	attrs := make(map[string]interface{})
	for i := 0; i <= internal.AttributeErrorLimit; i++ {
		attrs[strconv.Itoa(i)] = i
	}
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.NoticeError(Error{
		Message:    "my msg",
		Class:      "my class",
		Attributes: attrs,
	})
	if errTooManyErrorAttributes != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{})
	app.ExpectErrorEvents(t, []internal.WantEvent{})
	app.ExpectMetrics(t, backgroundMetrics)
}

type basicError struct{}

func (e basicError) Error() string { return "something went wrong" }

type withClass struct{ class string }

func (e withClass) Error() string      { return "something went wrong" }
func (e withClass) ErrorClass() string { return e.class }

type withClassAndCause struct {
	cause error
	class string
}

func (e withClassAndCause) Error() string      { return e.cause.Error() }
func (e withClassAndCause) Unwrap() error      { return e.cause }
func (e withClassAndCause) ErrorClass() string { return e.class }

type withCause struct{ cause error }

func (e withCause) Error() string { return e.cause.Error() }
func (e withCause) Unwrap() error { return e.cause }

func errWithClass(class string) error           { return withClass{class: class} }
func wrapWithClass(e error, class string) error { return withClassAndCause{cause: e, class: class} }
func wrapError(e error) error                   { return withCause{cause: e} }

func TestErrorClass(t *testing.T) {
	// First choice is any ErrorClass() of the immediate error.
	// Second choice is any ErrorClass() of the error's cause.
	// Final choice is the reflect type of the error's cause.
	testcases := []struct {
		Error  error
		Expect string
	}{
		{Error: basicError{}, Expect: "newrelic.basicError"},
		{Error: errWithClass("zap"), Expect: "zap"},
		{Error: errWithClass(""), Expect: "newrelic.withClass"},
		{Error: wrapWithClass(errWithClass("zap"), "zip"), Expect: "zip"},
		{Error: wrapWithClass(errWithClass("zap"), ""), Expect: "zap"},
		{Error: wrapWithClass(errWithClass(""), ""), Expect: "newrelic.withClass"},
		{Error: wrapError(basicError{}), Expect: "newrelic.basicError"},
		{Error: wrapError(errWithClass("zap")), Expect: "zap"},
	}

	for idx, tc := range testcases {
		data, err := errDataFromError(tc.Error)
		if err != nil {
			t.Errorf("testcase %d: got error: %v", idx, err)
			continue
		}
		if data.Klass != tc.Expect {
			t.Errorf("testcase %d: expected %s got %s", idx, tc.Expect, data.Klass)
		}
	}
}
