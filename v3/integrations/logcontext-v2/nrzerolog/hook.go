package nrzerolog

import (
	"context"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/rs/zerolog"
)

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
		Timestamp: time.Now().UnixMilli(),
		Severity:  logLevel,
		Message:   msg,
		Context:   h.Context,
	}

	h.App.RecordLog(&data)
}
