package zerologWriter

import (
	"bytes"
	"context"
	"io"
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
			`{"time":1516134303,"level":"debug","message":"hello world"}` + "\n",
			"level",
			newrelic.LogData{
				Message:  `{"time":1516134303,"level":"debug","message":"hello world"}` + "\n",
				Severity: "debug",
			},
		},
		{
			`{"time":1516134303,"level":"info","message":"hello world"}` + "\n",
			"level",
			newrelic.LogData{
				Message:  `{"time":1516134303,"level":"info","message":"hello world"}` + "\n",
				Severity: "info",
			},
		},
		{
			`{"time":1516133263,"level":"fatal","error":"A repo man spends his life getting into tense situations","service":"myservice","message":"Cannot start myservice"}` + "\n",
			"level",
			newrelic.LogData{
				Message:  `{"time":1516133263,"level":"fatal","error":"A repo man spends his life getting into tense situations","service":"myservice","message":"Cannot start myservice"}` + "\n",
				Severity: "fatal",
			},
		},
		{
			`{"time":1516134303,"hi":"info","message":"hello world"}` + "\n",
			"hi",
			newrelic.LogData{
				Message:  `{"time":1516134303,"hi":"info","message":"hello world"}` + "\n",
				Severity: "info",
			},
		},
		{
			`{"time":1516134303,"level":"debug","message":"hello, world"}` + "\n",
			"level",
			newrelic.LogData{
				Message:  `{"time":1516134303,"level":"debug","message":"hello, world"}` + "\n",
				Severity: "debug",
			},
		},
		{
			`{"time":1516134303,"level":"debug","message":"hello, world { thing }"}` + "\n",
			"level",
			newrelic.LogData{
				Message:  `{"time":1516134303,"level":"debug","message":"hello, world { thing }"}` + "\n",
				Severity: "debug",
			},
		},
		{
			`{"time":1516134303,"level":"debug","message":"hello, world \"{ thing \"}"}` + "\n",
			"level",
			newrelic.LogData{
				Message:  `{"time":1516134303,"level":"debug","message":"hello, world \"{ thing \"}"}` + "\n",
				Severity: "debug",
			},
		},
		{
			`{"message":"hello, world \"{ thing \"}","time":1516134303,"level":"debug"}` + "\n",
			"level",
			newrelic.LogData{
				Message:  `{"message":"hello, world \"{ thing \"}","time":1516134303,"level":"debug"}` + "\n",
				Severity: "debug",
			},
		},
		{
			// basic stack trace test
			`{"level":"error","stack":[{"func":"inner","line":"20","source":"errors.go"},{"func":"middle","line":"24","source":"errors.go"},{"func":"outer","line":"32","source":"errors.go"},{"func":"main","line":"15","source":"errors.go"},{"func":"main","line":"204","source":"proc.go"},{"func":"goexit","line":"1374","source":"asm_amd64.s"}],"error":"seems we have an error here","time":1609086683}` + "\n",
			"level",
			newrelic.LogData{
				Message:  `{"level":"error","stack":[{"func":"inner","line":"20","source":"errors.go"},{"func":"middle","line":"24","source":"errors.go"},{"func":"outer","line":"32","source":"errors.go"},{"func":"main","line":"15","source":"errors.go"},{"func":"main","line":"204","source":"proc.go"},{"func":"goexit","line":"1374","source":"asm_amd64.s"}],"error":"seems we have an error here","time":1609086683}` + "\n",
				Severity: "error",
			},
		},
		{
			// Tests that code can handle a stack trace, even if its at EOL
			`{"level":"error","stack":[{"func":"inner","line":"20","source":"errors.go"},{"func":"middle","line":"24","source":"errors.go"},{"func":"outer","line":"32","source":"errors.go"},{"func":"main","line":"15","source":"errors.go"},{"func":"main","line":"204","source":"proc.go"},{"func":"goexit","line":"1374","source":"asm_amd64.s"}]}` + "\n",
			"level",
			newrelic.LogData{
				Message:  `{"level":"error","stack":[{"func":"inner","line":"20","source":"errors.go"},{"func":"middle","line":"24","source":"errors.go"},{"func":"outer","line":"32","source":"errors.go"},{"func":"main","line":"15","source":"errors.go"},{"func":"main","line":"204","source":"proc.go"},{"func":"goexit","line":"1374","source":"asm_amd64.s"}]}` + "\n",
				Severity: "error",
			},
		},
		{
			`{"level":"debug","Scale":"833 cents","Interval":833.09,"time":1562212768,"message":"Fibonacci is everywhere"}` + "\n",
			"level",
			newrelic.LogData{
				Message:  `{"level":"debug","Scale":"833 cents","Interval":833.09,"time":1562212768,"message":"Fibonacci is everywhere"}` + "\n",
				Severity: "debug",
			},
		},
		{
			`{"Scale":"833 cents","Interval":833.09,"time":1562212768,"message":"Fibonacci is everywhere","level":"debug"}` + "\n",
			"level",
			newrelic.LogData{
				Message:  `{"Scale":"833 cents","Interval":833.09,"time":1562212768,"message":"Fibonacci is everywhere","level":"debug"}` + "\n",
				Severity: "debug",
			},
		},
	}
	for _, test := range tests {
		if test.levelKey != "" {
			zerolog.LevelFieldName = test.levelKey
		}
		val := parseJSONLogData([]byte(test.log))

		if val.Message != test.expect.Message {
			parserTestError(t, "Message", val.Message, test.expect.Message, test.log)
		}
		if val.Severity != test.expect.Severity {
			parserTestError(t, "Severity", val.Severity, test.expect.Severity, test.log)
		}

		zerolog.LevelFieldName = "level"
	}
}

func TestParseLogDataEscapes(t *testing.T) {
	type logTest struct {
		logMessage    string
		levelKey      string
		expectMessage string
	}
	tests := []logTest{
		{
			"escape quote,\"",
			"info",
			`{"level":"info","message":"escape quote,\""}`,
		},
		{
			"escape quote,\", hi",
			"info",
			`{"level":"info","message":"escape quote,\", hi"}`,
		},
		{
			"escape quote,\",\" hi",
			"info",
			`{"level":"info","message":"escape quote,\",\" hi"}`,
		},
		{
			"escape bracket,\"}\n hi",
			"info",
			`{"level":"info","message":"escape bracket,\"}\n hi"}`,
		},
	}

	app := integrationsupport.NewTestApp(
		integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogForwardingEnabled(true),
	)

	writer := New(io.Discard, app.Application)
	writer.DebugLogging(true)
	logger := zerolog.New(writer)

	wantLog := []internal.WantLog{}
	for _, test := range tests {
		logger.Info().Msg(test.logMessage)
		wantLog = append(wantLog, internal.WantLog{
			Severity:  zerolog.LevelInfoValue,
			Message:   test.expectMessage,
			Timestamp: internal.MatchAnyUnixMilli,
		})

	}
	app.ExpectLogEvents(t, wantLog)

}

func parserTestError(t *testing.T, field, actual, expect, input string) {
	t.Errorf("The parsed %s does not match the expected message: parsed \"%s\" expected \"%s\"\nFailed on input: %s", field, actual, expect, input)
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

func TestE2EWithContext(t *testing.T) {
	app := integrationsupport.NewTestApp(
		integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)
	buf := bytes.NewBuffer([]byte{})
	a := New(buf, app.Application)
	a.DebugLogging(true)

	txn := app.Application.StartTransaction("test")

	ctx := newrelic.NewContext(context.Background(), txn)
	txnWriter := a.WithContext(ctx)
	logger := zerolog.New(txnWriter)

	logger.Info().Msg("Hello World!")
	traceID := txn.GetLinkingMetadata().TraceID
	spanID := txn.GetLinkingMetadata().SpanID
	txn.End() // must end txn to dump logs into harvest

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
			TraceID:   traceID,
			SpanID:    spanID,
		},
	})
}

func TestE2EWithTxn(t *testing.T) {
	app := integrationsupport.NewTestApp(
		integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)
	buf := bytes.NewBuffer([]byte{})
	a := New(buf, app.Application)
	a.DebugLogging(true)

	txn := app.Application.StartTransaction("test")

	// create logger with txn context
	txnWriter := a.WithTransaction(txn)
	logger := zerolog.New(txnWriter)

	logger.Info().Msg("Hello World!")
	traceID := txn.GetLinkingMetadata().TraceID
	spanID := txn.GetLinkingMetadata().SpanID
	txn.End() // must end txn to dump logs into harvest

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
			TraceID:   traceID,
			SpanID:    spanID,
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
