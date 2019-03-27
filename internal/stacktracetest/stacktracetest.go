// Package stacktracetest helps test stack trace behavior.
package stacktracetest

// TopStackFrame is a function should will appear in the stacktrace.
func TopStackFrame(generateStacktrace func() []byte) []byte {
	return generateStacktrace()
}
