// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrpkgerrors introduces support for https://github.com/pkg/errors.
//
// This package improves the class and stack-trace fields of pkg/error errors
// when they are recorded with Transaction.NoticeError.
//
package nrpkgerrors

import (
	"fmt"

	"github.com/newrelic/go-agent/v3/internal"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
	"github.com/pkg/errors"
)

func init() { internal.TrackUsage("integration", "pkg-errors") }

// stackTracer is an error that also knows about its StackTrace.
// All wrapped errors from github.com/pkg/errors implement this interface.
type stackTracer interface {
	StackTrace() errors.StackTrace
}

func deepestStackTrace(err error) errors.StackTrace {
	var last stackTracer
	for err != nil {
		if err, ok := err.(stackTracer); ok {
			last = err
		}
		cause, ok := err.(interface {
			Cause() error
		})
		if !ok {
			break
		}
		err = cause.Cause()
	}

	if last == nil {
		return nil
	}
	return last.StackTrace()
}

func transformStackTrace(orig errors.StackTrace) []uintptr {
	st := make([]uintptr, len(orig))
	for i, frame := range orig {
		st[i] = uintptr(frame)
	}
	return st
}

func stackTrace(e error) []uintptr {
	st := deepestStackTrace(e)
	if nil == st {
		return nil
	}
	return transformStackTrace(st)
}

type errorClasser interface {
	ErrorClass() string
}

func errorClass(e error) string {
	if ec, ok := e.(errorClasser); ok {
		return ec.ErrorClass()
	}
	cause := errors.Cause(e)
	if ec, ok := cause.(errorClasser); ok {
		return ec.ErrorClass()
	}
	return fmt.Sprintf("%T", cause)
}

// Wrap wraps a pkg/errors error so that when noticed by
// newrelic.Transaction.NoticeError it gives an improved stacktrace and class
// type.
func Wrap(e error) error {
	attributes := make(map[string]interface{})
	switch error := e.(type) {
	case newrelic.Error:
		// if e is type newrelic.Error, copy attributes into wrapped error
		for key, value := range error.ErrorAttributes() {
			attributes[key] = value
		}
	}
	return newrelic.Error{
		Message:    e.Error(),
		Class:      errorClass(e),
		Stack:      stackTrace(e),
		Attributes: attributes,
	}
}
