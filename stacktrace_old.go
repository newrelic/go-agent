// +build !go1.7

package newrelic

import "github.com/newrelic/go-agent/internal"

// stackTrace returns the current stack trace. There is an alternative
// implementation of this function that is used in go1.7 or newer which
// supports custom stack traces recorded within the error if it implements
// a specific interface (see stacktrace_new.go).
func stackTrace(err error, skipFrames int) internal.StackTrace {
	return internal.GetStackTrace(skipFrames + 1)
}
