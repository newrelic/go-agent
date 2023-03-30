// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/newrelic/go-agent/v3/internal/stacktracetest"
)

func TestGetStackTrace(t *testing.T) {
	stack := getStackTrace()
	js, err := json.Marshal(stack)
	if nil != err {
		t.Fatal(err)
	}
	if nil == js {
		t.Fatal(string(js))
	}
}

func TestLongStackTraceLimitsFrames(t *testing.T) {
	st := stacktracetest.CountedCall(maxStackTraceFrames+20, func() []uintptr {
		return getStackTrace()
	})
	if len(st) != maxStackTraceFrames {
		t.Error("Unexpected size of stacktrace", maxStackTraceFrames, len(st))
	}
	l := len(stackTrace(st).frames())
	if l != maxStackTraceFrames {
		t.Error("Unexpected number of frames", maxStackTraceFrames, l)
	}
}

func TestManyStackTraceFramesLimitsOutput(t *testing.T) {
	frames := make([]StacktraceFrame, maxStackTraceFrames+20)
	expect := `[
	{},{},{},{},{},{},{},{},{},{},
	{},{},{},{},{},{},{},{},{},{},
	{},{},{},{},{},{},{},{},{},{},
	{},{},{},{},{},{},{},{},{},{},
	{},{},{},{},{},{},{},{},{},{},
	{},{},{},{},{},{},{},{},{},{},
	{},{},{},{},{},{},{},{},{},{},
	{},{},{},{},{},{},{},{},{},{},
	{},{},{},{},{},{},{},{},{},{},
	{},{},{},{},{},{},{},{},{},{}
	]`
	estimate := 256 * len(frames)
	output := bytes.NewBuffer(make([]byte, 0, estimate))
	writeFrames(output, frames)
	if compactJSONString(expect) != output.String() {
		t.Error("Unexpected JSON output", compactJSONString(expect), output.String())
	}
}

func TestStacktraceFrames(t *testing.T) {
	// This stacktrace taken from Go 1.13
	inputFrames := []StacktraceFrame{
		{
			File: "/Users/will/Desktop/gopath/src/github.com/newrelic/go-agent/v3/internal/stacktrace.go",
			Name: "github.com/newrelic/go-agent/v3/internal.GetStackTrace",
			Line: 18,
		},
		{
			File: "/Users/will/Desktop/gopath/src/github.com/newrelic/go-agent/v3/newrelic/internal_txn.go",
			Name: "github.com/newrelic/go-agent/v3/newrelic.errDataFromError",
			Line: 533,
		},
		{
			File: "/Users/will/Desktop/gopath/src/github.com/newrelic/go-agent/v3/newrelic/internal_txn.go",
			Name: "github.com/newrelic/go-agent/v3/newrelic.(*txn).NoticeError",
			Line: 575,
		},
		{
			File: "/Users/will/Desktop/gopath/src/github.com/newrelic/go-agent/v3/newrelic/transaction.go",
			Name: "github.com/newrelic/go-agent/v3/newrelic.(*Transaction).NoticeError",
			Line: 90,
		},
		{
			File: "/Users/will/Desktop/gopath/src/github.com/newrelic/go-agent/v3/examples/server/main.go",
			Name: "main.noticeError",
			Line: 30,
		},
		{
			File: "/Users/will/.gvm/gos/go1.13/src/net/http/server.go",
			Name: "net/http.HandlerFunc.ServeHTTP",
			Line: 2007,
		},
		{
			File: "/Users/will/Desktop/gopath/src/github.com/newrelic/go-agent/v3/newrelic/instrumentation.go",
			Name: "github.com/newrelic/go-agent/v3/newrelic.WrapHandle.func1",
			Line: 41,
		},
		{
			File: "/Users/will/.gvm/gos/go1.13/src/net/http/server.go",
			Name: "net/http.HandlerFunc.ServeHTTP",
			Line: 2007,
		},
		{
			File: "/Users/will/Desktop/gopath/src/github.com/newrelic/go-agent/v3/newrelic/instrumentation.go",
			Name: "github.com/newrelic/go-agent/v3/newrelic.WrapHandleFunc.func1",
			Line: 71,
		},
		{
			File: "/Users/will/.gvm/gos/go1.13/src/net/http/server.go",
			Name: "net/http.HandlerFunc.ServeHTTP",
			Line: 2007,
		},
		{
			File: "/Users/will/.gvm/gos/go1.13/src/net/http/server.go",
			Name: "net/http.(*ServeMux).ServeHTTP",
			Line: 2387,
		},
		{
			File: "/Users/will/.gvm/gos/go1.13/src/net/http/server.go",
			Name: "net/http.serverHandler.ServeHTTP",
			Line: 2802,
		},
		{
			File: "/Users/will/.gvm/gos/go1.13/src/net/http/server.go",
			Name: "net/http.(*conn).serve",
			Line: 1890,
		},
		{
			File: "/Users/will/.gvm/gos/go1.13/src/runtime/asm_amd64.s",
			Name: "runtime.goexit",
			Line: 1357,
		},
	}
	buf := &bytes.Buffer{}
	writeFrames(buf, inputFrames)
	expectedJSON := `[
		{
			"name":"main.noticeError",
			"filepath":"/Users/will/Desktop/gopath/src/github.com/newrelic/go-agent/v3/examples/server/main.go",
			"line":30
		},
		{
			"name":"http.HandlerFunc.ServeHTTP",
			"filepath":"/Users/will/.gvm/gos/go1.13/src/net/http/server.go",
			"line":2007
		},
		{
			"name":"newrelic.WrapHandle.func1",
			"filepath":"/Users/will/Desktop/gopath/src/github.com/newrelic/go-agent/v3/newrelic/instrumentation.go",
			"line":41
		},
		{
			"name":"http.HandlerFunc.ServeHTTP",
			"filepath":"/Users/will/.gvm/gos/go1.13/src/net/http/server.go",
			"line":2007
		},
		{
			"name":"newrelic.WrapHandleFunc.func1",
			"filepath":"/Users/will/Desktop/gopath/src/github.com/newrelic/go-agent/v3/newrelic/instrumentation.go",
			"line":71
		},
		{
			"name":"http.HandlerFunc.ServeHTTP",
			"filepath":"/Users/will/.gvm/gos/go1.13/src/net/http/server.go",
			"line":2007
		},
		{
			"name":"http.(*ServeMux).ServeHTTP",
			"filepath":"/Users/will/.gvm/gos/go1.13/src/net/http/server.go",
			"line":2387
		},
		{
			"name":"http.serverHandler.ServeHTTP",
			"filepath":"/Users/will/.gvm/gos/go1.13/src/net/http/server.go",
			"line":2802
		},
		{
			"name":"http.(*conn).serve",
			"filepath":"/Users/will/.gvm/gos/go1.13/src/net/http/server.go",
			"line":1890
		},
		{
			"name":"runtime.goexit",
			"filepath":"/Users/will/.gvm/gos/go1.13/src/runtime/asm_amd64.s",
			"line":1357
		}]`
	testExpectedJSON(t, expectedJSON, buf.String())
}

func TestStackTraceTopFrame(t *testing.T) {
	// This test uses a separate package since the stacktrace code removes
	// the top stack frames which are in packages "newrelic" and "internal".
	stackJSON := stacktracetest.TopStackFrame(func() []byte {
		st := getStackTrace()
		js, _ := json.Marshal(st)
		return js
	})

	stack := []struct {
		Name     string `json:"name"`
		FilePath string `json:"filepath"`
		Line     int    `json:"line"`
	}{}
	if err := json.Unmarshal(stackJSON, &stack); err != nil {
		t.Fatal(err)
	}
	if len(stack) < 2 {
		t.Fatal(string(stackJSON))
	}
	if stack[0].Name != "stacktracetest.TopStackFrame" {
		t.Error(string(stackJSON))
	}
	if stack[0].Line != 9 {
		t.Error(string(stackJSON))
	}
	if !strings.Contains(stack[0].FilePath, "go-agent/v3/internal/stacktracetest/stacktracetest.go") {
		t.Error(string(stackJSON))
	}
}

func TestFramesCount(t *testing.T) {
	st := stacktracetest.CountedCall(3, func() []uintptr {
		return getStackTrace()
	})
	frames := stackTrace(st).frames()
	if len(st) != len(frames) {
		t.Error("Invalid # of frames", len(st), len(frames))
	}
}
