// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"runtime"
	"strings"
	"testing"
)

// The use of runtime.CallersFrames requires Go 1.7+.

func topFrameFunction(stack []uintptr) string {
	var frame runtime.Frame
	frames := runtime.CallersFrames(stack)
	if nil != frames {
		frame, _ = frames.Next()
	}
	return frame.Function
}

type withStackAndCause struct {
	cause error
	stack []uintptr
}

type withStack struct {
	stack []uintptr
}

func (e withStackAndCause) Error() string         { return e.cause.Error() }
func (e withStackAndCause) StackTrace() []uintptr { return e.stack }
func (e withStackAndCause) Unwrap() error         { return e.cause }

func (e withStack) Error() string         { return "something went wrong" }
func (e withStack) StackTrace() []uintptr { return e.stack }

func generateStack() []uintptr {
	skip := 2 // skip runtime.Callers and this function.
	callers := make([]uintptr, 20)
	written := runtime.Callers(skip, callers)
	return callers[:written]
}

func alpha() []uintptr { return generateStack() }
func beta() []uintptr  { return generateStack() }

func TestStackTrace(t *testing.T) {
	// First choice is any StackTrace() of the immediate error.
	// Second choice is any StackTrace() of the error's cause.
	// Final choice is stack trace of the current location.
	getStackTraceFrame := "github.com/newrelic/go-agent/v3/newrelic.getStackTrace"
	testcases := []struct {
		Error          error
		ExpectTopFrame string
	}{
		{Error: basicError{}, ExpectTopFrame: getStackTraceFrame},
		{Error: withStack{stack: alpha()}, ExpectTopFrame: "alpha"},
		{Error: withStack{stack: nil}, ExpectTopFrame: getStackTraceFrame},
		{Error: withStackAndCause{stack: alpha(), cause: basicError{}}, ExpectTopFrame: "alpha"},
		{Error: withStackAndCause{stack: nil, cause: withStack{stack: beta()}}, ExpectTopFrame: "beta"},
		{Error: withStackAndCause{stack: nil, cause: withStack{stack: nil}}, ExpectTopFrame: getStackTraceFrame},
	}

	for idx, tc := range testcases {
		data, err := errDataFromError(tc.Error, false)
		if err != nil {
			t.Errorf("testcase %d: got error: %v", idx, err)
			continue
		}
		fn := topFrameFunction(data.Stack)
		if !strings.Contains(fn, tc.ExpectTopFrame) {
			t.Errorf("testcase %d: expected %s got %s",
				idx, tc.ExpectTopFrame, fn)
		}
	}
}
