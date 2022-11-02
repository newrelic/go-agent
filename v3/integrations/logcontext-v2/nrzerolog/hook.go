package nrzerolog

import (
	"context"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/rs/zerolog"
)

func init() { internal.TrackUsage("integration", "logcontext-v2", "zerolog") }

type NewRelicHook struct {
	App     *newrelic.Application
	Context context.Context
}

func (h NewRelicHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	var txn *newrelic.Transaction
	if h.Context != nil {
		txn = newrelic.FromContext(h.Context)
	}

	logLevel := ""
	if level != zerolog.NoLevel {
		logLevel = level.String()
	}

	data := newrelic.LogData{
		Severity: logLevel,
		Message:  msg,
	}

	if txn != nil {
		txn.RecordLog(data)
	} else {
		h.App.RecordLog(data)
	}
}
