package internal

import (
	"bytes"
	"fmt"
	"path"
	"runtime"
	"strings"

	"github.com/newrelic/go-agent/internal/jsonx"
)

// StackTrace is a stack trace.
type StackTrace struct {
	callers []uintptr
	written int
}

// GetStackTrace returns a new StackTrace.
func GetStackTrace(skipFrames int) *StackTrace {
	st := &StackTrace{}

	skip := 2 // skips runtime.Callers and this function
	skip += skipFrames

	st.callers = make([]uintptr, maxStackTraceFrames)
	st.written = runtime.Callers(skip, st.callers)
	st.callers = st.callers[0:st.written]

	return st
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

func topCallerNameBase(st *StackTrace) string {
	f, _ := pcToFunc(st.callers[0])
	if nil == f {
		return ""
	}
	return path.Base(f.Name())
}

// simplifyStackTraceFilename makes stack traces smaller and more readable by
// removing anything preceding the first occurrence of `/src/`.  This is
// intended to remove the $GOROOT and the $GOPATH.
func simplifyStackTraceFilename(raw string) string {
	idx := strings.Index(raw, "/src/")
	if idx < 0 {
		return raw
	}
	return raw[idx+5:]
}

const (
	unknownStackTraceFunc = "unknown"
)

// WriteJSON adds the stack trace to the buffer in the JSON form expected by the
// collector.
func (st *StackTrace) WriteJSON(buf *bytes.Buffer) {
	buf.WriteByte('[')
	for i, pc := range st.callers {
		if i > 0 {
			buf.WriteByte(',')
		}
		f, place := pcToFunc(pc)
		str := unknownStackTraceFunc
		if nil != f {
			// Format designed to match the Ruby agent.
			name := path.Base(f.Name())
			file, line := f.FileLine(place)
			str = fmt.Sprintf("%s:%d:in `%s'",
				simplifyStackTraceFilename(file), line, name)
		}
		jsonx.AppendString(buf, str)
	}
	buf.WriteByte(']')
}

// MarshalJSON prepares JSON in the format expected by the collector.
func (st *StackTrace) MarshalJSON() ([]byte, error) {
	estimate := 256 * len(st.callers)
	buf := bytes.NewBuffer(make([]byte, 0, estimate))

	st.WriteJSON(buf)

	return buf.Bytes(), nil
}
