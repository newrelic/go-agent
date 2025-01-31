package nrslog

import (
	"log/slog"
	"strings"

	"github.com/newrelic/go-agent/v3/newrelic"
)

const (
	nrlinking = "NR-LINKING"
	key       = "newrelic"
)

func enrichRecord(app *newrelic.Application, record *slog.Record) {
	if !shouldEnrichLog(app) {
		return
	}

	str := nrLinkingString(app.GetLinkingMetadata())
	if str == "" {
		return
	}

	record.AddAttrs(slog.String(key, str))
}

func enrichRecordTxn(txn *newrelic.Transaction, record *slog.Record) {
	if !shouldEnrichLog(txn.Application()) {
		return
	}

	str := nrLinkingString(txn.GetLinkingMetadata())
	if str == "" {
		return
	}

	record.AddAttrs(slog.String(key, str))
}

func shouldEnrichLog(app *newrelic.Application) bool {
	config, ok := app.Config()
	if !ok {
		return false
	}

	return config.ApplicationLogging.Enabled && config.ApplicationLogging.LocalDecorating.Enabled
}

// nrLinkingString returns a string that represents the linking metadata
func nrLinkingString(data newrelic.LinkingMetadata) string {
	if data.EntityGUID == "" {
		return ""
	}

	len := 16 + len(data.EntityGUID) + len(data.Hostname) + len(data.TraceID) + len(data.SpanID) + len(data.EntityName)
	str := strings.Builder{}
	str.Grow(len) // only 1 alloc

	str.WriteString(nrlinking)
	str.WriteByte('|')
	str.WriteString(data.EntityGUID)
	str.WriteByte('|')
	str.WriteString(data.Hostname)
	str.WriteByte('|')
	str.WriteString(data.TraceID)
	str.WriteByte('|')
	str.WriteString(data.SpanID)
	str.WriteByte('|')
	str.WriteString(data.EntityName)
	str.WriteByte('|')

	return str.String()
}
