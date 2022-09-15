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

func TestParseLogData(t *testing.T) {
	type logTest struct {
		log      string
		levelKey string
		expect   newrelic.LogData
	}
	tests := []logTest{
		{
			`{"time":1516134303,"level":"debug","message":"hello world"}`,
			"level",
			newrelic.LogData{
				Message:  `{"time":1516134303,"level":"debug","message":"hello world"}`,
				Severity: "debug",
			},
		},
		{
			`{"time":1516134303,"level":"info","message":"hello world"}`,
			"level",
			newrelic.LogData{
				Message:  `{"time":1516134303,"level":"info","message":"hello world"}`,
				Severity: "info",
			},
		},
		{
			`{"time":1516133263,"level":"fatal","error":"A repo man spends his life getting into tense situations","service":"myservice","message":"Cannot start myservice"}`,
			"level",
			newrelic.LogData{
				Message:  `{"time":1516133263,"level":"fatal","error":"A repo man spends his life getting into tense situations","service":"myservice","message":"Cannot start myservice"}`,
				Severity: "fatal",
			},
		},
		{
			`{"time":1516134303,"hi":"info","message":"hello world"}`,
			"hi",
			newrelic.LogData{
				Message:  `{"time":1516134303,"hi":"info","message":"hello world"}`,
				Severity: "info",
			},
		},
	}
	for _, test := range tests {
		if test.levelKey != "" {
			zerolog.LevelFieldName = test.levelKey
		}
		val := parseJSONLogData([]byte(test.log))

		if val.Message != test.expect.Message {
			parserTestError(t, "Message", val.Message, test.expect.Message)
		}
		if val.Severity != test.expect.Severity {
			parserTestError(t, "Severity", val.Severity, test.expect.Severity)
		}

		zerolog.LevelFieldName = "level"
	}
}

func parserTestError(t *testing.T, field, actual, expect string) {
	t.Errorf("The parsed %s does not match the expected message: parsed \"%s\" expected \"%s\"", field, actual, expect)
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
		parseJSONLogData(log)
	}
}
