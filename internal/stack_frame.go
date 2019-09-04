// +build go1.7

package internal

import "runtime"

func (st StackTrace) frames() []stacktraceFrame {
	fs := make([]stacktraceFrame, len(st))
	frames := runtime.CallersFrames(st)
	for i := range st {
		frame, more := frames.Next()
		fs[i] = stacktraceFrame{
			Name: frame.Function,
			File: frame.File,
			Line: int64(frame.Line),
		}
		if !more {
			break
		}
	}
	return fs
}
