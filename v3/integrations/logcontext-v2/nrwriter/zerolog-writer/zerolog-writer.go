package zerologWriter

import (
	"context"
	"io"
	"strings"

	"github.com/newrelic/go-agent/v3/integrations/logcontext-v2/nrwriter"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/rs/zerolog"
)

type ZerologWriter struct {
	w nrwriter.LogWriter
}

// New creates a new NewRelicWriter Object
// output is the io.Writer destination that you want your log to be written to
// app must be a vaild, non nil new relic Application
func New(output io.Writer, app *newrelic.Application) ZerologWriter {
	return ZerologWriter{
		w: nrwriter.New(output, app),
	}
}

// DebugLogging toggles whether error information should be printed to console. By default, this service
// will fail silently. Enabling debug logging will print error messages on a new line after your log message.
func (zw *ZerologWriter) DebugLogging(enabled bool) { zw.w.DebugLogging(enabled) }

// WithTransaction creates a new ZerologWriter for a specific transactions
func (zw *ZerologWriter) WithTransaction(txn *newrelic.Transaction) ZerologWriter {
	return ZerologWriter{w: zw.w.WithTransaction(txn)}
}

// WithContext creates a new ZerologWriter for the transaction inside of a context
func (zw *ZerologWriter) WithContext(ctx context.Context) ZerologWriter {
	return ZerologWriter{w: zw.w.WithContext(ctx)}
}

// Write is a valid io.Writer method that will write the content of an enriched log to the output io.Writer
func (zw ZerologWriter) Write(p []byte) (n int, err error) {
	logLevel := parseLogLevel(p)
	enrichedLog := zw.w.EnrichLog(logLevel, p)
	return zw.w.Write(enrichedLog)
}

func parseLogLevel(log []byte) string {
	levelKey := zerolog.LevelFieldName
	keyIndx := 0

	matchKey := false
	value := strings.Builder{}
	value.Grow(8)
	for i := 0; i < len(log); i++ {
		if !matchKey {
			if log[i] == levelKey[keyIndx] {
				keyIndx++
				if keyIndx == len(levelKey) && log[i+1] == '"' {
					matchKey = true
					i += 3
				}
			} else {
				keyIndx = 0
			}
		} else {
			if log[i] == '"' {
				break
			}
			value.WriteByte(log[i])
		}
	}

	return value.String()
}
