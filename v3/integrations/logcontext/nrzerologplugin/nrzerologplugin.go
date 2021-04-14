package nrzerologplugin

import (
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/newrelic/go-agent/v3/integrations/logcontext"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/rs/zerolog"
)

func init() { internal.TrackUsage("integration", "logcontext", "zerolog") }

// Middleware adds the necessary date to enable logs in context
func Middleware(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {

		updateContextFn := func(c zerolog.Context) zerolog.Context {
			if txn := newrelic.FromContext(r.Context()); nil != txn {
				data := map[string]interface{}{
					logcontext.KeyMessage: zerolog.MessageFieldName,
					logcontext.KeyLevel:   zerolog.LevelFieldName,
				}

				logcontext.AddLinkingMetadata(data, txn.GetLinkingMetadata())

				c = c.Fields(data)
			}

			return c
		}

		zerolog.Ctx(r.Context()).UpdateContext(updateContextFn)
		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

// Hook Zerolog hook to add information regarding the file to the logs
func Hook(e *zerolog.Event, level zerolog.Level, msg string) {
	if e.Enabled() {
		// We cannot get the timestamp from zerolog, so we use the current one
		e.Uint64(logcontext.KeyTimestamp, uint64(time.Now().UnixNano())/uint64(1000*1000))

		// 6 is a magic number, that's the number of frames in the stacktrace we need to
		// go up until we get to the function/method who actually called the logger.
		// It depends on how many function calls we introduced and how many are
		// introduced by zerolog.
		file, line, method := traceinfo(6)
		e.Str(logcontext.KeyFile, file)
		e.Str(logcontext.KeyLine, strconv.Itoa(line))
		e.Str(logcontext.KeyMethod, method)
		e.Str(logcontext.KeyLevel, level.String())
	}
}

// traceinfo returns the file, line number and function name.
// calldepth is the number of frames to be ignored.
func traceinfo(calldepth int) (string, int, string) {
	pc := make([]uintptr, 1)
	n := runtime.Callers(calldepth, pc)
	frames := runtime.CallersFrames(pc[:n])
	f, _ := frames.Next()
	return f.File, f.Line, f.Function
}

// trimPath shortens given path leaving at most given number of right hand
// segments.
func trimPath(filepath string, segments int) string {
	var chunks int
	for i := len(filepath) - 1; i > 0; i-- {
		if filepath[i] == '/' {
			chunks++
			if chunks >= segments {
				return filepath[i+1:]
			}
		}
	}
	return filepath
}
