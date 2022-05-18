// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrpkgerrors

import (
	"runtime"
	"strings"
	"testing"

	newrelic "github.com/newrelic/go-agent/v3/newrelic"
	"github.com/pkg/errors"
)

func topFrameFunction(stack []uintptr) string {
	var frame runtime.Frame
	frames := runtime.CallersFrames(stack)
	if nil != frames {
		frame, _ = frames.Next()
	}
	return frame.Function
}

type basicError struct{}

func (e basicError) Error() string { return "something went wrong" }

func alpha(e error) error { return errors.WithStack(e) }
func beta(e error) error  { return errors.WithStack(e) }
func gamma(e error) error { return errors.WithStack(e) }

func theta(e error) error { return errors.WithMessage(e, "theta") }

func basicNRError(e error) newrelic.Error { return newrelic.Error{Message: e.Error()} }
func withAttributes(e error) newrelic.Error {
	return newrelic.Error{
		Message: e.Error(),
		Attributes: map[string]interface{}{
			"testAttribute": 1,
			"foo":           2,
		},
	}
}

func TestWrappedStackTrace(t *testing.T) {
	testcases := []struct {
		Error            error
		ExpectTopFrame   string
		ExpectAttributes map[string]interface{}
	}{
		{Error: basicError{}, ExpectTopFrame: ""},
		{Error: alpha(basicError{}), ExpectTopFrame: "alpha"},
		{Error: alpha(beta(gamma(basicError{}))), ExpectTopFrame: "gamma"},
		{Error: alpha(theta(basicError{})), ExpectTopFrame: "alpha"},
		{Error: alpha(theta(beta(basicError{}))), ExpectTopFrame: "beta"},
		{Error: alpha(theta(beta(theta(basicError{})))), ExpectTopFrame: "beta"},
		{Error: theta(basicError{}), ExpectTopFrame: ""},
		{Error: basicNRError(basicError{}), ExpectTopFrame: ""},
		{Error: withAttributes(basicError{}), ExpectTopFrame: "", ExpectAttributes: map[string]interface{}{"testAttribute": 1, "foo": 2}},
	}

	for idx, tc := range testcases {
		e := Wrap(tc.Error)
		st := e.(newrelic.Error).StackTrace()
		fn := topFrameFunction(st)
		if !strings.Contains(fn, tc.ExpectTopFrame) {
			t.Errorf("testcase %d: expected %s got %s",
				idx, tc.ExpectTopFrame, fn)
		}
		// check that error attributes are equal if they are expected
		if tc.ExpectAttributes != nil {
			errorAttributes := e.(newrelic.Error).ErrorAttributes()
			if len(tc.ExpectAttributes) != len(errorAttributes) {
				t.Errorf("testcase %d: error attribute size expected %d got %d",
					idx, len(tc.ExpectAttributes), len(errorAttributes))
			}
			for k, v := range errorAttributes {
				if tc.ExpectAttributes[k] != v {
					t.Errorf("testcase %d: expected attribute %s:%v got %s:%v",
						idx, k, tc.ExpectAttributes[k], k, v)
				}
			}
		}
	}
}

type withClass struct{ class string }

func errorWithClass(class string) error { return withClass{class: class} }

func (e withClass) Error() string      { return "something went wrong" }
func (e withClass) ErrorClass() string { return e.class }

type classAndCause struct {
	cause error
	class string
}

func wrapWithClass(e error, class string) error { return classAndCause{cause: e, class: class} }

func (e classAndCause) Error() string      { return e.cause.Error() }
func (e classAndCause) Cause() error       { return e.cause }
func (e classAndCause) ErrorClass() string { return e.class }

func TestWrappedErrorClass(t *testing.T) {
	// First choice is any ErrorClass of the immediate error.
	// Second choice is any ErrorClass of the error's cause.
	// Final choice is the reflect type of the error's cause.
	testcases := []struct {
		Error       error
		ExpectClass string
	}{
		{Error: basicError{}, ExpectClass: "nrpkgerrors.basicError"},
		{Error: errorWithClass("zap"), ExpectClass: "zap"},
		{Error: wrapWithClass(errorWithClass("zap"), "zip"), ExpectClass: "zip"},
		{Error: theta(wrapWithClass(errorWithClass("zap"), "zip")), ExpectClass: "zap"},
		{Error: alpha(basicError{}), ExpectClass: "nrpkgerrors.basicError"},
		{Error: wrapWithClass(basicError{}, "zip"), ExpectClass: "zip"},
		{Error: alpha(wrapWithClass(basicError{}, "zip")), ExpectClass: "nrpkgerrors.basicError"},
	}

	for idx, tc := range testcases {
		e := Wrap(tc.Error)
		class := e.(newrelic.Error).ErrorClass()
		if class != tc.ExpectClass {
			t.Errorf("testcase %d: expected %s got %s",
				idx, tc.ExpectClass, class)
		}
	}
}
