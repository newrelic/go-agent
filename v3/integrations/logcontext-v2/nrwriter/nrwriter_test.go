package nrwriter

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	"github.com/newrelic/go-agent/v3/internal/logcontext"
	"github.com/newrelic/go-agent/v3/internal/sysinfo"
	"github.com/newrelic/go-agent/v3/newrelic"
)

var (
	host, _ = sysinfo.Hostname()
)

const (
	benchmarkMsg = "This is a test log message"
)

func BenchmarkEnrichLog(b *testing.B) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn, newrelic.ConfigAppLogDecoratingEnabled(true))
	a := New(io.Discard, app.Application)
	a.DebugLogging(true)

	buf := bytes.NewBuffer([]byte(benchmarkMsg))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.EnrichLog(newrelic.LogData{}, buf.Bytes())
	}
}

const (
	logMessageWithNewline                = "This is a log message with a newline\n"
	logMessageWithoutNewline             = "This is a log message without a newline"
	logMessageWithSpace                  = "This is a log message with a space at the end \n"
	logMessageWithoutNewlineAndWithSpace = "This is a log message without a newline "
	nrlinking                            = "NR-LINKING"
)

func TestLogSpacingAndNewlines(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn, newrelic.ConfigAppLogDecoratingEnabled(true))
	buf := bytes.NewBuffer([]byte{})
	a := New(buf, app.Application)
	a.DebugLogging(true)

	lines := []string{
		logMessageWithNewline,
		logMessageWithSpace,
		logMessageWithoutNewline,
		logMessageWithoutNewlineAndWithSpace,
	}

	for _, line := range lines {
		buf.Write(a.EnrichLog(newrelic.LogData{}, []byte(line)))
		log := buf.String()
		// verify there is a single newline at the end of the log line
		if strings.Count(log, "\n") != 1 {
			t.Errorf("Expected a single log line ending with one newline, instead got: %s", log)
		}

		substrings := strings.Split(log, nrlinking)
		if len(substrings) != 2 {
			t.Errorf("Expected %s metadata but log line was not decorated: %s", nrlinking, log)
		} else {
			whitespace := countTrailingWhitespace(substrings[0])
			if whitespace != 1 {
				t.Errorf("Expecting a single whitespace separating log line from %s, got %d: %s", nrlinking, whitespace, log)
			}
		}

		buf.Reset()
	}
}

func countTrailingWhitespace(str string) int {
	count := 0
	for i := len(str) - 1; i >= 0; i-- {
		if str[i] == ' ' {
			count++
		} else {
			break
		}
	}

	return count
}

func TestBackgroundLog(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn, newrelic.ConfigAppLogDecoratingEnabled(true))
	buf := bytes.NewBuffer([]byte{})
	a := New(buf, app.Application)
	a.DebugLogging(true)

	a.Write(a.EnrichLog(newrelic.LogData{}, []byte(logMessageWithNewline)))
	logcontext.ValidateDecoratedOutput(t, buf, &logcontext.DecorationExpect{
		EntityGUID: integrationsupport.TestEntityGUID,
		Hostname:   host,
		EntityName: integrationsupport.SampleAppName,
	})
}

func TestTransactionLog(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn, newrelic.ConfigAppLogDecoratingEnabled(true))
	buf := bytes.NewBuffer([]byte{})
	a := New(buf, app.Application)
	a.DebugLogging(true)

	txn := app.StartTransaction("test transaction")
	b := a.WithTransaction(txn)

	b.Write(b.EnrichLog(newrelic.LogData{}, []byte(logMessageWithNewline)))
	logcontext.ValidateDecoratedOutput(t, buf, &logcontext.DecorationExpect{
		EntityGUID: integrationsupport.TestEntityGUID,
		Hostname:   host,
		EntityName: integrationsupport.SampleAppName,
		TraceID:    txn.GetLinkingMetadata().TraceID,
		SpanID:     txn.GetLinkingMetadata().SpanID,
	})
	txn.End()
}

func TestContextLog(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn, newrelic.ConfigAppLogDecoratingEnabled(true))
	buf := bytes.NewBuffer([]byte{})
	a := New(buf, app.Application)
	a.DebugLogging(true)

	txn := app.StartTransaction("test transaction")
	ctx := newrelic.NewContext(context.Background(), txn)
	b := a.WithContext(ctx)

	b.Write(b.EnrichLog(newrelic.LogData{}, []byte(logMessageWithNewline)))
	logcontext.ValidateDecoratedOutput(t, buf, &logcontext.DecorationExpect{
		EntityGUID: integrationsupport.TestEntityGUID,
		Hostname:   host,
		EntityName: integrationsupport.SampleAppName,
		TraceID:    txn.GetLinkingMetadata().TraceID,
		SpanID:     txn.GetLinkingMetadata().SpanID,
	})
	txn.End()
}

func TestNilContextLog(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn, newrelic.ConfigAppLogDecoratingEnabled(true))
	buf := bytes.NewBuffer([]byte{})
	a := New(buf, app.Application)
	a.DebugLogging(true)

	b := a.WithContext(nil)

	// verify that when context is nil, log is enriched with application data
	b.Write(b.EnrichLog(newrelic.LogData{}, []byte(logMessageWithNewline)))
	logcontext.ValidateDecoratedOutput(t, buf, &logcontext.DecorationExpect{
		EntityGUID: integrationsupport.TestEntityGUID,
		Hostname:   host,
		EntityName: integrationsupport.SampleAppName,
	})

	buf.Reset()

	// verify that when context is empty, log is enriched with application data
	c := a.WithContext(context.Background())
	c.Write(c.EnrichLog(newrelic.LogData{}, []byte(logMessageWithNewline)))
	logcontext.ValidateDecoratedOutput(t, buf, &logcontext.DecorationExpect{
		EntityGUID: integrationsupport.TestEntityGUID,
		Hostname:   host,
		EntityName: integrationsupport.SampleAppName,
	})
}
