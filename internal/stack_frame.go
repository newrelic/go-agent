// +build go1.7

package internal

import "runtime"

func (st StackTrace) frames() []stacktraceFrame {
	fs := make([]stacktraceFrame, maxStackTraceFrames)
	frames := runtime.CallersFrames(st)
	i := 0
	for frame, more := frames.Next(); more && (i < maxStackTraceFrames); frame, more = frames.Next() {
		fs[i] = stacktraceFrame{
			Name: frame.Function,
			File: frame.File,
			Line: int64(frame.Line),
		}
		i++
	}
	return fs[0:i]
}
