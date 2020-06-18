// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package stacktracetest helps test stack trace behavior.
package stacktracetest

// TopStackFrame is a function should will appear in the stacktrace.
func TopStackFrame(generateStacktrace func() []byte) []byte {
	return generateStacktrace()
}

// CountedCall is a function that allows you to generate a stack trace with this function being called a particular
// number of times. The parameter f should be a function that returns a StackTrace (but it is referred to as []uintptr
// in order to not create a circular dependency on the internal package)
func CountedCall(i int, f func() []uintptr) []uintptr {
	if i > 0 {
		return CountedCall(i-1, f)
	}
	return f()
}
