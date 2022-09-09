package logWriter

import (
	"context"
	"io"

	"github.com/newrelic/go-agent/v3/internal/logcontext/nrwriter"
	"github.com/newrelic/go-agent/v3/newrelic"
)

type LogWriter struct {
	w nrwriter.LogWriter
}

func init() { internal.TrackUsage("integration", "logcontext-v2", "logWriter") }

// New creates a new LogWriter
// output is the io.Writer destination that you want your log to be written to
// app must be a vaild, non nil new relic Application
func New(output io.Writer, app *newrelic.Application) LogWriter {
	return LogWriter{
		w: nrwriter.New(output, app),
	}
}

// DebugLogging toggles whether error information should be printed to console. By default, this service
// will fail silently. Enabling debug logging will print error messages on a new line after your log message.
func (lw *LogWriter) DebugLogging(enabled bool) { lw.w.DebugLogging(enabled) }

// WithTransaction creates a new LogWriter for a specific transactions
func (lw *LogWriter) WithTransaction(txn *newrelic.Transaction) LogWriter {
	return LogWriter{w: lw.w.WithTransaction(txn)}
}

// WithContext creates a new LogWriter for the transaction inside of a context
func (lw *LogWriter) WithContext(ctx context.Context) LogWriter {
	return LogWriter{w: lw.w.WithContext(ctx)}
}

// Write is a valid io.Writer method that will write the content of an enriched log to the output io.Writer
func (lw LogWriter) Write(p []byte) (n int, err error) {
	enrichedLog := lw.w.EnrichLog(newrelic.LogData{Message: string(p)}, p)
	return lw.w.Write(enrichedLog)
}
