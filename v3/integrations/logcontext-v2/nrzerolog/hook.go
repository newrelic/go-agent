package nrzerolog

import (
	"context"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/rs/zerolog"
)

func init() { internal.TrackUsage("integration", "logcontext", "zerolog") }

type NewRelicHook struct {
	App     *newrelic.Application
	Context context.Context
}

func (h NewRelicHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	logLevel := ""
	if level == zerolog.NoLevel {
		logLevel = newrelic.LogSeverityUnknown
	} else {
		logLevel = level.String()
	}
	data := newrelic.LogData{
		Severity: logLevel,
		Message:  msg,
		Context:  h.Context,
	}

	h.App.RecordLog(data)
}