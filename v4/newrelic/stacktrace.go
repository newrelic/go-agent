// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"runtime"
)

const (
	maxStackTraceFrames = 100
)

// stackTrace is a stack trace.
type stackTrace []uintptr

// getStackTrace returns a new stackTrace.
func getStackTrace() stackTrace {
	skip := 1 // skip runtime.Callers
	callers := make([]uintptr, maxStackTraceFrames)
	written := runtime.Callers(skip, callers)
	return callers[:written]
}
