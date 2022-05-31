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

	// Versions of go prior to 1.17 do not have a built in function for Unix Milli time.
	// For go versions 1.17+ use time.Now().UnixMilli() to generate timestamps
	timestamp := time.Now().UnixNano() / int64(time.Millisecond)

	data := newrelic.LogData{
		Timestamp: timestamp,
		Severity:  logLevel,
		Message:   msg,
		Context:   h.Context,
	}

	h.App.RecordLog(data)
}
