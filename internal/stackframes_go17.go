// +build go1.7

package internal

import (
	"runtime"
)

func (st StackTrace) frames() []stacktraceFrame {
	frames := make([]stacktraceFrame, 0, len(st))
	cf := runtime.CallersFrames(st)
	for {
		f, more := cf.Next()
		if !more {
			break
		}
		frames = append(frames, stacktraceFrame{
			Name: f.Func.Name(),
			File: f.File,
			Line: int64(f.Line),
		})
	}
	return frames
}
