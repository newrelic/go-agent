package internal

import (
	"encoding/json"
	"fmt"
	"path"
	"runtime"
)

type StackTrace struct {
	callers []uintptr
	written int
}

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
	// TODO: Fully understand when this substraction is necessary.
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

// MarshalJSON prepares JSON in the format expected by the collector.
func (st *StackTrace) MarshalJSON() ([]byte, error) {
	lines := make([]string, 0, len(st.callers))

	for _, pc := range st.callers {
		f, place := pcToFunc(pc)
		str := "in unknown"
		if nil != f {
			name := f.Name()
			file, line := f.FileLine(place)
			str = fmt.Sprintf(" in %s called at %s (%d)", name, file, line)
		}

		lines = append(lines, str)
	}

	return json.Marshal(lines)
}
