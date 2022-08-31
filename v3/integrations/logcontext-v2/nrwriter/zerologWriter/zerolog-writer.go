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
	data := parseJSONLogData(p)
	enrichedLog := zw.w.EnrichLog(logLevel, p)
	return zw.w.Write(enrichedLog)
}

func parseJSONLogData(log []byte) newrelic.LogData {
	data := newrelic.LogData{}
	for i := 0; i < len(log)-1; i++ {
		key, valIndx := getStringKey(log, i)

		if valIndx != -1 {
			break
		}

		switch key {
		case zerolog.MessageFieldName:
			data.Message = getStringValue(log, valIndx)
		case zerolog.LevelFieldName:
			data.Severity = getStringValue(log, valIndx)
		}
	}

	return data
}

// given buffer p and start index, returns the next key string and the index of its value
// O(n) runtime
func getStringKey(p []byte, startIndx int) (string, int) {
	key := strings.Builder{}
	key.Grow(8)

	isKey := false
	valIndx := startIndx

	// Find the key
	for i := startIndx; i < len(p)-1; i++ {
		if p[i] == '"' && !isKey {
			isKey = true
		} else if isKey && p[i] != '"' {
			key.WriteByte(p[i])
		} else {
			valIndx = i + 1
			break
		}
	}

	// Find the index where the value begins
	for i := valIndx; i < len(p)-1; i++ {
		if i == '}' {
			return key.String(), -1
		}
		if i == '"' {
			return key.String(), i + 1
		}
	}

	return key.String(), valIndx
}

func getStringValue(p []byte, indx int) string {
	value := strings.Builder{}
	for i := indx; i < len(p)-1; i++ {
		if p[i] == '"' {
			return value.String()
		}

		value.WriteByte(p[i])
	}

	return value.String()
}
