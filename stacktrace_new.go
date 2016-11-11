// +build go1.7

package newrelic

import (
	"runtime"

	"github.com/newrelic/go-agent/internal"
)

// A StackTracer is an error type that can return information about the
// stacktrace at which it was created.
//
// The stackTracer interface is not exported by this package, but is considered
// to be part of the stable public API.
type stackTracer interface {
	StackTrace() []runtime.Frame
}

// stackTrace extracts the stack trace from err if it implements the stackTracer
// interface or the current stack trace. Since the interface requires go 1.7 or
// higher there is an alternative implementation of this function in this package
// (stacktrace_old.go) that will just return the current stack trace.
func stackTrace(err error, skipFrames int) internal.StackTrace {
	t, ok := err.(stackTracer)
	if !ok {
		return internal.GetStackTrace(skipFrames + 1)
	}

	var stack internal.StackTrace
	for _, s := range t.StackTrace() {
		stack = append(stack, internal.StackFrame{
			File:     s.File,
			Line:     s.Line,
			Function: s.Function,
		})
	}

	return stack
}
