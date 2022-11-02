package zerologWriter

import (
	"context"
	"io"
	"strings"
	"time"
	"unicode"

	"github.com/newrelic/go-agent/v3/integrations/logcontext-v2/nrwriter"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/rs/zerolog"
)

type ZerologWriter struct {
	w nrwriter.LogWriter
}

func init() { internal.TrackUsage("integration", "logcontext-v2", "zerologWriter") }

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
	data.Timestamp = time.Now().UnixMilli()

	for i := 0; i < len(log)-1; {
		// get key; always a string field
		key, valStart := getKey(log, i)
		var next int

		// NOTE: depending on the key, the type of field the value is can differ
		switch key {
		case zerolog.LevelFieldName:
			data.Severity, next = getStringValue(log, valStart+1)
		case zerolog.ErrorStackFieldName:
			_, next = getStackTrace(log, valStart)
		default:
			if i >= len(log)-1 {
				return data
			}
			// TODO: once we update the logging spec to support custom attributes, capture these
			if isStringValue(log, valStart) {
				_, next = getStringValue(log, valStart+1)
			} else if isNumberValue(log, valStart) {
				_, next = getNumberValue(log, valStart)
			} else {
				return data
			}
		}

		if next == -1 {
			return data
		}
		i = next
	}

	return data
}

func isStringValue(p []byte, indx int) bool {
	return p[indx] == '"'
}

func isNumberValue(p []byte, indx int) bool {
	return unicode.IsDigit(rune(p[indx]))
}

// zerolog keys are always JSON strings
func getKey(p []byte, indx int) (string, int) {
	value := strings.Builder{}
	i := indx

	// find start of string field
	for ; i < len(p); i++ {
		if p[i] == '"' {
			i += 1
			break
		}
	}

	// parse value of string field
	for ; i < len(p); i++ {
		if p[i] == '"' && i+1 < len(p) && p[i+1] == ':' {
			return value.String(), i + 2
		} else {
			value.WriteByte(p[i])
		}
	}

	return "", -1
}

func isEOL(p []byte, i int) bool {
	return p[i] == '}' && i+2 == len(p)
}

func getStringValue(p []byte, indx int) (string, int) {
	value := strings.Builder{}

	// parse value of string field
	for i := indx; i < len(p); i++ {
		if p[i] == '"' && i+1 < len(p) {
			if p[i+1] == ',' && i+2 < len(p) && p[i+2] == '"' {
				return value.String(), i + 2
			} else if isEOL(p, i+1) {
				return value.String(), -1
			}
		}
		value.WriteByte(p[i])
	}

	return "", -1
}

func getNumberValue(p []byte, indx int) (string, int) {
	value := strings.Builder{}

	// parse value of string field
	for i := indx; i < len(p); i++ {
		if p[i] == ',' && i+1 < len(p) && p[i+1] == '"' {
			return value.String(), i + 1
		} else if isEOL(p, i) {
			return value.String(), -1
		} else {
			value.WriteByte(p[i])
		}
	}

	return "", -1
}

func getStackTrace(p []byte, indx int) (string, int) {
	value := strings.Builder{}

	// parse value of string field
	for i := indx; i < len(p); i++ {
		if p[i] == ']' {
			value.WriteByte(p[i])

			if i+1 < len(p) {
				if isEOL(p, i+1) {
					return value.String(), -1
				}
				if p[i+1] == ',' && i+2 < len(p) && p[i+2] == '"' {
					return value.String(), i + 2
				}
			}
		} else {
			value.WriteByte(p[i])
		}
	}

	return value.String(), -1
}
