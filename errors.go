// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import "github.com/newrelic/go-agent/internal"

// StackTracer can be implemented by errors to provide a stack trace when using
// Transaction.NoticeError.
type StackTracer interface {
	StackTrace() []uintptr
}

// ErrorClasser can be implemented by errors to provide a custom class when
// using Transaction.NoticeError.
type ErrorClasser interface {
	ErrorClass() string
}

// ErrorAttributer can be implemented by errors to provide extra context when
// using Transaction.NoticeError.
type ErrorAttributer interface {
	ErrorAttributes() map[string]interface{}
}

// Error is an error that implements ErrorClasser, ErrorAttributer, and
// StackTracer.  Use it with Transaction.NoticeError to directly control error
// message, class, stacktrace, and attributes.
type Error struct {
	// Message is the error message which will be returned by the Error()
	// method.
	Message string
	// Class indicates how the error may be aggregated.
	Class string
	// Attributes are attached to traced errors and error events for
	// additional context.  These attributes are validated just like those
	// added to `Transaction.AddAttribute`.
	Attributes map[string]interface{}
	// Stack is the stack trace.  Assign this field using NewStackTrace,
	// or leave it nil to indicate that Transaction.NoticeError should
	// generate one.
	Stack []uintptr
}

// NewStackTrace generates a stack trace which can be assigned to the Error
// struct's Stack field or returned by an error that implements the ErrorClasser
// interface.
func NewStackTrace() []uintptr {
	st := internal.GetStackTrace()
	return []uintptr(st)
}

func (e Error) Error() string { return e.Message }

// ErrorClass implements the ErrorClasser interface.
func (e Error) ErrorClass() string { return e.Class }

// ErrorAttributes implements the ErrorAttributes interface.
func (e Error) ErrorAttributes() map[string]interface{} { return e.Attributes }

// StackTrace implements the StackTracer interface.
func (e Error) StackTrace() []uintptr { return e.Stack }
