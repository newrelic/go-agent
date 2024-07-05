package nrslog_test

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/newrelic/go-agent/v3/integrations/nrslog"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func Example() {
	// Get the default logger or create a new one:
	l := slog.Default()

	_, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Example App"),
		newrelic.ConfigLicense("__YOUR_NEWRELIC_LICENSE_KEY__"),
		// Use nrslog to register the logger with the agent:
		nrslog.ConfigLogger(l.WithGroup("newrelic")),
	)
	if err != nil {
		panic(err)
	}
}

func TestLogs(t *testing.T) {
	type args struct {
		message      string
		EnabledLevel slog.Level
	}

	tests := []struct {
		name    string
		args    args
		logFunc func(logger newrelic.Logger, message string, attrs map[string]interface{})
		want    string
	}{
		{
			name: "Error",
			args: args{
				message:      "error message",
				EnabledLevel: slog.LevelError,
			},
			logFunc: newrelic.Logger.Error,
			want:    "level=ERROR msg=\"error message\" key=val\n",
		},
		{
			name: "Warn",
			args: args{
				message:      "warning message",
				EnabledLevel: slog.LevelWarn,
			},
			logFunc: newrelic.Logger.Warn,
			want:    "level=WARN msg=\"warning message\" key=val\n",
		},
		{
			name: "Info",
			args: args{
				message:      "informational message",
				EnabledLevel: slog.LevelInfo,
			},
			logFunc: newrelic.Logger.Info,
			want:    "level=INFO msg=\"informational message\" key=val\n",
		},
		{
			name: "Debug",
			args: args{
				message:      "debug message",
				EnabledLevel: slog.LevelDebug,
			},
			logFunc: newrelic.Logger.Debug,
			want:    "level=DEBUG msg=\"debug message\" key=val\n",
		},
		{
			name: "Disabled",
			args: args{
				message:      "disabled message",
				EnabledLevel: slog.LevelError,
			},
			logFunc: newrelic.Logger.Debug,
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create slog to record logs at the specified level:
			buf := new(bytes.Buffer)
			handler := slog.NewTextHandler(buf, &slog.HandlerOptions{
				Level: tt.args.EnabledLevel,
				ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
					// Remove time from the output for predictable test output.
					if a.Key == slog.TimeKey {
						return slog.Attr{}
					}
					return a
				},
			})
			logger := slog.New(handler)

			// Create test logger using nrslog.Transform:
			testLogger := nrslog.Transform(logger)

			// Define attributes for the test log message:
			attrs := map[string]interface{}{
				"key": "val",
			}

			// Log the message and attributes using the test logger:
			tt.logFunc(testLogger, tt.args.message, attrs)

			assert.Equal(t, tt.want, buf.String())
		})
	}
}

func TestDebugEnabled(t *testing.T) {
	type args struct {
		EnabledLevel slog.Level
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Debug",
			args: args{
				EnabledLevel: slog.LevelDebug,
			},
			want: true,
		},
		{
			name: "Info",
			args: args{
				EnabledLevel: slog.LevelInfo,
			},
			want: false,
		},
		{
			name: "Warn",
			args: args{
				EnabledLevel: slog.LevelWarn,
			},
			want: false,
		},
		{
			name: "Error",
			args: args{
				EnabledLevel: slog.LevelError,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := slog.NewJSONHandler(
				new(bytes.Buffer),
				&slog.HandlerOptions{Level: tt.args.EnabledLevel},
			)
			logger := slog.New(handler)
			testLogger := nrslog.Transform(logger)

			assert.Equal(t, tt.want, testLogger.DebugEnabled())
		})
	}
}
