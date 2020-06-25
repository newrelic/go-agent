// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package logcontext facilitates adding New Relic context to your logs.
//
// Adding New Relic context to your logs will allow you to see links between
// your events and traces in APM and your logs.  If you are using a logging
// framework that does not already have a New Relic plugin for log decoration,
// use this package to manually add logging context.
//
// See https://github.com/newrelic/newrelic-exporter-specs/tree/master/logging
// for a complete specification.
package logcontext

import newrelic "github.com/newrelic/go-agent/v3/newrelic"

// Keys used for logging context JSON.
const (
	KeyFile       = "file.name"
	KeyLevel      = "log.level"
	KeyLine       = "line.number"
	KeyMessage    = "message"
	KeyMethod     = "method.name"
	KeyTimestamp  = "timestamp"
	KeyTraceID    = "trace.id"
	KeySpanID     = "span.id"
	KeyEntityName = "entity.name"
	KeyEntityType = "entity.type"
	KeyEntityGUID = "entity.guid"
	KeyHostname   = "hostname"
)

func metadataMapField(m map[string]interface{}, key, val string) {
	if val != "" {
		m[key] = val
	}
}

// AddLinkingMetadata adds the LinkingMetadata into a map.  Only non-empty
// string fields are included in the map.  The specific key names facilitate
// agent logs in context.  These keys are: "trace.id", "span.id",
// "entity.name", "entity.type", "entity.guid", and "hostname".
func AddLinkingMetadata(m map[string]interface{}, md newrelic.LinkingMetadata) {
	metadataMapField(m, KeyTraceID, md.TraceID)
	metadataMapField(m, KeySpanID, md.SpanID)
	metadataMapField(m, KeyEntityName, md.EntityName)
	metadataMapField(m, KeyEntityType, md.EntityType)
	metadataMapField(m, KeyEntityGUID, md.EntityGUID)
	metadataMapField(m, KeyHostname, md.Hostname)
}
