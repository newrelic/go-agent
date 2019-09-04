// +build go1.7

package internal

import "runtime"

func (st StackTrace) frames() []stacktraceFrame {
	fs := make([]stacktraceFrame, len(st))
	frames := runtime.CallersFrames(st)
	for i := 0; i < len(st); i++ {
		frame, more := frames.Next()
		f := runtime.FuncForPC(frame.PC)
		if nil == f {
			fs[i] = stacktraceFrame{}
		} else {
			fs[i] = stacktraceFrame{
				Name: f.Name(),
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
