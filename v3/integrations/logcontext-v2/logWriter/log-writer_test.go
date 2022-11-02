package logWriter

import (
	"bytes"
	"log"
	"testing"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	"github.com/newrelic/go-agent/v3/internal/logcontext"
	"github.com/newrelic/go-agent/v3/internal/sysinfo"
	"github.com/newrelic/go-agent/v3/newrelic"
)

var (
	host, _ = sysinfo.Hostname()
)

func TestE2E(t *testing.T) {
	app := integrationsupport.NewTestApp(
		integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)

	// Capture output in a buffer for testing
	buf := bytes.NewBuffer([]byte{})

	// set up logger
	writer := New(buf, app.Application)
	logger := log.New(&writer, "My Prefix: ", log.Lshortfile)

	// configure log writer
	writer.DebugLogging(true)

	// create a log message
	logger.Print("Hello World!")

	logcontext.ValidateDecoratedOutput(t, buf, &logcontext.DecorationExpect{
		EntityGUID: integrationsupport.TestEntityGUID,
		Hostname:   host,
		EntityName: integrationsupport.SampleAppName,
	})

	app.ExpectLogEvents(t, []internal.WantLog{
		{
			Severity:  logcontext.LogSeverityUnknown,
			Message:   "My Prefix: log-writer_test.go:37: Hello World!",
			Timestamp: internal.MatchAnyUnixMilli,
		},
	})
}

func BenchmarkWrite(b *testing.B) {
	app := integrationsupport.NewTestApp(
		integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)
	buf := bytes.NewBuffer([]byte{})
	a := New(buf, app.Application)

	log := []byte(`{"time":1516134303,"level":"debug","message":"hello world"}`)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		a.Write(log)
	}
}
