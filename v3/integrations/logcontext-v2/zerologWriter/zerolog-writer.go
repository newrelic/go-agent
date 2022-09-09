package zerologWriter

import (
	"context"
	"io"
	"strings"

	"github.com/newrelic/go-agent/v3/internal/logcontext/nrwriter"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/rs/zerolog"
)

type ZerologWriter struct {
	w nrwriter.LogWriter
}

func init() { internal.TrackUsage("integration", "logcontext-v2", "zerolog") }

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
	data := parseJSONLogData(p)
	enrichedLog := zw.w.EnrichLog(data, p)
	return zw.w.Write(enrichedLog)
}

func parseJSONLogData(log []byte) newrelic.LogData {
	// For this iteration of the tool, the entire log gets captured as the message
	data := newrelic.LogData{}
	data.Message = string(log)

	for i := 0; i < len(log)-1; {
		// get key; always a string field
		key, keyEnd := getStringField(log, i)

		// find index where value starts
		valStart := getValueIndex(log, keyEnd)
		valEnd := valStart

		// NOTE: depending on the key, the type of field the value is can differ
		switch key {
		case zerolog.LevelFieldName:
			data.Severity, valEnd = getStringField(log, valStart)
		}

		next := nextKeyIndex(log, valEnd)
		if next == -1 {
			return data
		}
		i = next
	}

	return data
}

func getValueIndex(p []byte, indx int) int {
	// Find the index where the value begins
	for i := indx; i < len(p)-1; i++ {
		if p[i] == ':' {
			return i + 1
		}
	}

	return -1
}

func nextKeyIndex(p []byte, indx int) int {
	// Find the index where the key begins
	for i := indx; i < len(p)-1; i++ {
		if p[i] == ',' {
			return i + 1
		}
	}

	return -1
}

func getStringField(p []byte, indx int) (string, int) {
	value := strings.Builder{}
	i := indx

	// find start of string field
	for ; i < len(p)-1; i++ {
		if p[i] == '"' {
			i += 1
			break
		}
	}

	// parse value of string field
	for ; i < len(p)-1; i++ {
		if p[i] == '"' {
			return value.String(), i + 1
		} else {
			value.WriteByte(p[i])
		}

	}

	return "", -1
}
