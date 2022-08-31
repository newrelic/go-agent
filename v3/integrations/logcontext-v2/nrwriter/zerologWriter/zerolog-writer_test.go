package zerologWriter

import (
	"bytes"
	"testing"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	"github.com/newrelic/go-agent/v3/internal/logcontext"
	"github.com/newrelic/go-agent/v3/internal/sysinfo"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/rs/zerolog"
)

var (
	host, _ = sysinfo.Hostname()
)

func TestParseLevelValue(t *testing.T) {
	type logTest struct {
		log       string
		expectVal string
		levelKey  string
	}
	tests := []logTest{
		{
			`{"time":1516134303,"level":"debug","message":"hello world"}`,
			"debug",
			"",
		},
		{
			`{"time":1516134303,"level":"info","message":"hello world"}`,
			"info",
			"",
		},
		{
			`{"time":1516133263,"level":"fatal","error":"A repo man spends his life getting into tense situations","service":"myservice","message":"Cannot start myservice"}`,
			"fatal",
			"",
		},
		{
			`{"time":1516134303,"hi":"info","message":"hello world"}`,
			"info",
			"hi",
		},
	}
	for _, test := range tests {
		if test.levelKey != "" {
			zerolog.LevelFieldName = test.levelKey
		}
		val := parseLogLevel([]byte(test.log))
		if val != test.expectVal {
			t.Errorf("incorrect log level; expected: debug, got %s", val)
		}

		zerolog.LevelFieldName = "level"
	}
}

func TestE2E(t *testing.T) {
	app := integrationsupport.NewTestApp(
		integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)
	buf := bytes.NewBuffer([]byte{})
	a := New(buf, app.Application)
	a.DebugLogging(true)

	logger := zerolog.New(a)
	logger.Info().Msg("Hello World!")

	logcontext.ValidateDecoratedOutput(t, buf, &logcontext.DecorationExpect{
		EntityGUID: integrationsupport.TestEntityGUID,
		Hostname:   host,
		EntityName: integrationsupport.SampleAppName,
	})

	app.ExpectLogEvents(t, []internal.WantLog{
		{
			Severity:  zerolog.LevelInfoValue,
			Message:   `{"level":"info","message":"Hello World!"}`,
			Timestamp: internal.MatchAnyUnixMilli,
		},
	})
}

func BenchmarkParseLogLevel(b *testing.B) {
	log := []byte(`{"time":1516134303,"level":"debug","message":"hello world"}`)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		parseLogLevel(log)
	}
}
