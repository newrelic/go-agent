package nrwriter

import (
	"bytes"
	"context"
	"io"

	"github.com/newrelic/go-agent/v3/newrelic"
)

// LogWriter is an io.Writer that captures log data for use with New Relic Logs in Context
type LogWriter struct {
	debug bool
	out   io.Writer
	app   *newrelic.Application
	txn   *newrelic.Transaction
}

// New creates a new NewRelicWriter Object
// output is the io.Writer destination that you want your log to be written to
// app must be a vaild, non nil new relic Application
func New(output io.Writer, app *newrelic.Application) LogWriter {
	return LogWriter{
		out: output,
		app: app,
	}
}

// DebugLogging enables or disables debug error messages being written in the IO output.
// By default, the nrwriter debug logging is set to false and will fail silently
func (b *LogWriter) DebugLogging(enabled bool) {
	b.debug = enabled
}

// WithTransaction duplicates the current NewRelicWriter and sets the transaction to txn
func (b *LogWriter) WithTransaction(txn *newrelic.Transaction) LogWriter {
	return LogWriter{
		out:   b.out,
		app:   b.app,
		debug: b.debug,
		txn:   txn,
	}
}

// WithTransaction duplicates the current NewRelicWriter and sets the transaction to the transaction parsed from ctx
func (b *LogWriter) WithContext(ctx context.Context) LogWriter {
	txn := newrelic.FromContext(ctx)
	return LogWriter{
		out:   b.out,
		app:   b.app,
		debug: b.debug,
		txn:   txn,
	}
}

// EnrichLog attempts to enrich a log with New Relic linking metadata. If it fails,
// it will return the original log line unless debug=true, otherwise it will print
// an error on a following line.
func (b *LogWriter) EnrichLog(data newrelic.LogData, p []byte) []byte {
	logLine := bytes.TrimRight(p, "\n")
	buf := bytes.NewBuffer(logLine)

	var enrichErr error
	if b.txn != nil {
		b.txn.RecordLog(data)
		enrichErr = newrelic.EnrichLog(buf, newrelic.FromTxn(b.txn))
	} else {
		b.app.RecordLog(data)
		enrichErr = newrelic.EnrichLog(buf, newrelic.FromApp(b.app))
	}

	if b.debug && enrichErr != nil {
		buf.WriteString("\n")
		buf.WriteString(enrichErr.Error())
	}

	buf.WriteString("\n")
	return buf.Bytes()
}

// Write implements io.Write
func (b LogWriter) Write(p []byte) (n int, err error) {
	return b.out.Write(p)
}
