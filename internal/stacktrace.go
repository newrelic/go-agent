package internal

import (
	"bytes"
	"path"
	"runtime"
)

// StackTrace is a list of nested StackFrames at a certain point in time.
// The frames in a StackTrace are ordered by nesting depth with the outermost
// function being at the last position of the list.
type StackTrace []StackFrame

// StackFrame represents a function call in a stack trace.
type StackFrame struct {
	File     string
	Line     int
	Function string
}

// GetStackTrace returns a new StackTrace.
func GetStackTrace(skipFrames int) StackTrace {
	frames := make([]uintptr, maxStackTraceFrames)

	// skips runtime.Callers and this function
	n := runtime.Callers(skipFrames+2, frames)

	trace := make([]StackFrame, n)
	for i, pc := range frames[0:n] {
		f := StackFrame{}
		fun, place := pcToFunc(pc)
		if nil != fun {
			f.Function = fun.Name()
			f.File, f.Line = fun.FileLine(place)
		}

		trace[i] = f
	}

	return StackTrace(trace)
}

func pcToFunc(pc uintptr) (*runtime.Func, uintptr) {
	// The Golang runtime package documentation says "To look up the file
	// and line number of the call itself, use pc[i]-1. As an exception to
	// this rule, if pc[i-1] corresponds to the function runtime.sigpanic,
	// then pc[i] is the program counter of a faulting instruction and
	// should be used without any subtraction."
	//
	// TODO: Fully understand when this subtraction is necessary.
	place := pc - 1
	return runtime.FuncForPC(place), place
}

func topCallerNameBase(st StackTrace) string {
	if len(st) == 0 {
		return ""
	}

	return path.Base(st[0].Function)
}

// WriteJSON adds the stack trace to the buffer in the JSON form expected by the
// collector.
func (st StackTrace) WriteJSON(buf *bytes.Buffer) {
	buf.WriteByte('[')
	for i, frame := range st {
		if i > 0 {
			buf.WriteByte(',')
		}

		buf.WriteByte('{')

		w := jsonFieldsWriter{buf: buf}
		w.stringField("filepath", frame.File)
		w.stringField("name", frame.Function)
		w.intField("line", int64(frame.Line))

		buf.WriteByte('}')
	}
	buf.WriteByte(']')
}

// MarshalJSON prepares JSON in the format expected by the collector.
func (st StackTrace) MarshalJSON() ([]byte, error) {
	estimate := 256 * len(st)
	buf := bytes.NewBuffer(make([]byte, 0, estimate))

	st.WriteJSON(buf)

	return buf.Bytes(), nil
}
