package errors

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type StackTracer struct {
	error
	trace errors.StackTrace
}

// stackTracer is an error that also knows about its StackTrace.
// All wrapped errors from github.com/pkg/errors implement this interface.
type stackTracer interface {
	StackTrace() errors.StackTrace
}

// causer is an error that wraps another error.
type causer interface {
	Cause() error
}

// WithStackTrace searches for the last cause of err (deepest in the stack) that implements
// the stackTracer interface. If no such cause exists, err is returned unmodified.
func WithStackTrace(err error) error {
	originalErr := err

	var lastStackTracer stackTracer
	for err != nil {
		if err, ok := err.(stackTracer); ok {
			lastStackTracer = err
		}

		cause, ok := err.(causer)
		if !ok {
			break
		}
		err = cause.Cause()
	}

	if lastStackTracer == nil {
		return originalErr
	}

	return &StackTracer{
		error: originalErr,
		trace: lastStackTracer.StackTrace(),
	}
}

// ErrorClass returns the error class that should be displayed in newrelic.
func (t *StackTracer) ErrorClass() string {
	if err, ok := t.error.(interface {
		ErrorClass() string
	}); ok {
		return err.ErrorClass()
	}

	return fmt.Sprintf("%T", t.error)
}

// StackTrace returns t.trace as stack trace that is understood by the newrelic agent.
func (t *StackTracer) StackTrace() []runtime.Frame {
	var trace []runtime.Frame
	for _, f := range t.trace {
		// we are jumping through some ropes here since there is no simpler way
		// of extracting this info from the errors.StackTrace
		funcFile := strings.SplitN(fmt.Sprintf("%+s", f), "\n", 2)
		file := strings.TrimSpace(funcFile[1])
		line, _ := strconv.ParseInt(fmt.Sprintf("%d", f), 10, 32)
		trace = append(trace, runtime.Frame{
			File:     file,
			Line:     int(line),
			Function: fmt.Sprintf("%n", f),
		})
	}

	return trace
}
