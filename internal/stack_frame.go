// +build go1.7

package internal

import "runtime"

func (st StackTrace) frames() []stacktraceFrame {
	fs := make([]stacktraceFrame, len(st))
	frames := runtime.CallersFrames(st)
	for i := range st {
		frame, more := frames.Next()
		fun := runtime.FuncForPC(frame.PC)
		if nil == fun {
			fs[i] = stacktraceFrame{}
		} else {
			fs[i] = stacktraceFrame{
				Name: fun.Name(),
				File: frame.File,
				Line: int64(frame.Line),
			}
		}
		if !more {
			break
		}
	}
	return fs
}
