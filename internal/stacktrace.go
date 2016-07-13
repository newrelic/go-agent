package internal

import (
	"encoding/json"
	"fmt"
	"path"
	"runtime"
)

type stackTrace struct {
	callers []uintptr
	written int
}

func getStackTrace(skipFrames int) *stackTrace {
	st := &stackTrace{}

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

func topCallerNameBase(st *stackTrace) string {
	f, _ := pcToFunc(st.callers[0])
	if nil == f {
		return ""
	}
	return path.Base(f.Name())
}

// MarshalJSON prepares JSON in the format expected by the collector.
func (st *stackTrace) MarshalJSON() ([]byte, error) {
	lines := make([]string, 0, len(st.callers))

	for _, pc := range st.callers {
		f, place := pcToFunc(pc)
		str := "unknown"
		if nil != f {
			// Format designed to match the Ruby agent.
			name := path.Base(f.Name())
			file, line := f.FileLine(place)
			str = fmt.Sprintf("%s:%d:in `%s'", file, line, name)
		}

		lines = append(lines, str)
	}

	return json.Marshal(lines)
}
