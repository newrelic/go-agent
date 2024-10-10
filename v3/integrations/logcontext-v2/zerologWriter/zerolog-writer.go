package zerologWriter

import (
	"bytes"
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

	i := skipPastSpaces(log, 0)
	if i < 0 || i >= len(log) || log[i] != '{' {
		return data
	}
	i++
	for i < len(log)-1 {
		// get key; always a string field
		key, valStart := getKey(log, i)
		if valStart < 0 {
			return data
		}
		var next int

		// NOTE: depending on the key, the type of field the value is can differ
		switch key {
		case zerolog.LevelFieldName:
			data.Severity, next = getStringValue(log, valStart)
		case zerolog.ErrorStackFieldName:
			_, next = getStackTrace(log, valStart)
		default:
			if i >= len(log)-1 {
				return data
			}
			// TODO: once we update the logging spec to support custom attributes, capture these
			if isStringValue(log, valStart) {
				_, next = getStringValue(log, valStart)
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
	if indx = skipPastSpaces(p, indx); indx < 0 {
		return false
	}
	return p[indx] == '"'
}

func isNumberValue(p []byte, indx int) bool {
	if indx = skipPastSpaces(p, indx); indx < 0 {
		return false
	}
	// unicode.IsDigit isn't sufficient here because JSON numbers can start with a sign too
	return unicode.IsDigit(rune(p[indx])) || p[indx] == '-'
}

// zerolog keys are always JSON strings
func getKey(p []byte, indx int) (string, int) {
	value := strings.Builder{}
	i := skipPastSpaces(p, indx)
	if i < 0 || i >= len(p) || p[i] != '"' {
		return "", -1
	}

	// parse value of string field
	literalNext := false
	for i++; i < len(p); i++ {
		if literalNext {
			value.WriteByte(p[i])
			literalNext = false
			continue
		}

		if p[i] == '\\' {
			value.WriteByte(p[i])
			literalNext = true
			continue
		}

		if p[i] == '"' {
			// found end of key. Now look for the colon separator
			for i++; i < len(p); i++ {
				if p[i] == ':' && i+1 < len(p) {
					return value.String(), i + 1
				}
				if p[i] != ' ' && p[i] != '\t' {
					break
				}
			}
			// Oh oh. if we got here, there wasn't a colon, or there wasn't a value after it, or
			// something showed up between the end of the key and the colon that wasn't a space.
			// In any of those cases, we didn't find the key of a key/value pair.
			return "", -1
		} else {
			value.WriteByte(p[i])
		}
	}
	return "", -1
}

/*
func isEOL(p []byte, i int) bool {
	for ; i < len(p); i++ {
		if p[i] == ' ' || p[i] == '\t' {
			continue
		}
		if p[i] == '}' {
			// nothing but space to the end from here?
			for i++; i < len(p); i++ {
				if p[i] != ' ' && p[i] != '\t' && p[i] != '\r' && p[i] != '\n' {
					return false // nope, that wasn't the end of the string
				}
			}
			return true
		}
	}
	return false
}
*/

func skipPastSpaces(p []byte, i int) int {
	for ; i < len(p); i++ {
		if p[i] != ' ' && p[i] != '\t' && p[i] != '\r' && p[i] != '\n' {
			return i
		}
	}
	return -1
}

func getStringValue(p []byte, indx int) (string, int) {
	value := strings.Builder{}

	// skip to start of string
	i := skipPastSpaces(p, indx)
	if i < 0 || i >= len(p) || p[i] != '"' {
		return "", -1
	}

	// parse value of string field
	literalNext := false
	for i++; i < len(p); i++ {
		if literalNext {
			value.WriteByte(p[i])
			literalNext = false
			continue
		}

		if p[i] == '\\' {
			value.WriteByte(p[i])
			literalNext = true
			continue
		}

		if p[i] == '"' {
			// end of string. search past the comma so we can find the following key (if any) later.
			if i = skipPastSpaces(p, i+1); i < 0 || i >= len(p) {
				return value.String(), -1
			}
			if p[i] == ',' {
				if i+1 < len(p) {
					return value.String(), i + 1
				}
				return value.String(), -1
			}
			return value.String(), -1
		}

		value.WriteByte(p[i])
	}

	return "", -1
}

func getNumberValue(p []byte, indx int) (string, int) {
	value := strings.Builder{}

	// parse value of string field
	i := skipPastSpaces(p, indx)
	if i < 0 {
		return "", -1
	}
	// JSON numeric values contain digits, '.', '-', 'e'
	for ; i < len(p) && bytes.IndexByte([]byte("0123456789-+eE."), p[i]) >= 0; i++ {
		value.WriteByte(p[i])
	}

	i = skipPastSpaces(p, i)
	if i > 0 && i+1 < len(p) && p[i] == ',' {
		return value.String(), i + 1
	}
	return value.String(), -1
}

func getStackTrace(p []byte, indx int) (string, int) {
	value := strings.Builder{}

	i := skipPastSpaces(p, indx)
	if i < 0 || i >= len(p) || p[i] != '[' {
		return "", -1
	}
	// the stack trace is everything from '[' to the next ']'.
	// TODO: this is a little naïve and we may need to consider parsing
	// the data inbetween more carefully. To date, we haven't seen a case
	// where that is necessary, and prefer not to impact performance of the
	// system by doing the extra processing, but we can revisit that later
	// if necessary.
	for ; i < len(p); i++ {
		if p[i] == ']' {
			value.WriteByte(p[i])
			i = skipPastSpaces(p, i)
			if i > 0 && i+1 < len(p) && p[i] == ',' {
				return value.String(), i + 1
			}
			return value.String(), -1
		} else {
			value.WriteByte(p[i])
		}
	}

	return value.String(), -1
}
