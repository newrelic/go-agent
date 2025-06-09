// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package integrationsupport

import (
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
	"testing"
)

type myError struct{}

func (e myError) Error() string { return "My error message" }

func TestNewBasicTestApp(t *testing.T) {
	expectedApp := NewBasicTestApp()
	txn := expectedApp.Application.StartTransaction("My transaction")
	txn.NoticeError(myError{})
	txn.End()
	expectedApp.ExpectErrors(t, []internal.WantError{{
		Msg: "My error message",
	}})
}

func TestDistributedTracingTestApp(t *testing.T) {
	expectedApp := NewTestApp(SampleEverythingReplyFn, DTEnabledCfgFn)
	txn := expectedApp.Application.StartTransaction("My transaction")
	defer txn.End()

	// will add more here
}

func TestAppLogsTestApp(t *testing.T) {
	expectedApp := NewTestApp(SampleEverythingReplyFn, AppLogEnabledCfgFn)
	txn := expectedApp.Application.StartTransaction("My transaction")
	defer txn.End()

	txn.RecordLog(newrelic.LogData{
		Message:   "Transaction Log Message",
		Severity:  "debug",
		Timestamp: 12345,
	})

	expectedApp.RecordLog(newrelic.LogData{
		Message:   "App Log Message",
		Severity:  "info",
		Timestamp: 78910,
	})

	txn.ExpectLogEvents(t, []internal.WantLog{{
		Message:    "App Log Message",
		Severity:   "info",
		Timestamp:  78910,
		SpanID:     expectedApp.GetLinkingMetadata().SpanID,
		TraceID:    expectedApp.GetLinkingMetadata().TraceID,
		Attributes: map[string]interface{}{}}, {
		Message:    "Transaction Log Message",
		Severity:   "debug",
		Timestamp:  12345,
		SpanID:     txn.GetLinkingMetadata().SpanID,
		TraceID:    txn.GetLinkingMetadata().TraceID,
		Attributes: map[string]interface{}{}}})
}

func TestFullTracesTestApp(t *testing.T) {
	expectedApp := NewTestApp(SampleEverythingReplyFn, ConfigFullTraces, newrelic.ConfigCodeLevelMetricsEnabled(false))
	txn := expectedApp.Application.StartTransaction("test")

	AddAgentAttribute(txn, newrelic.AttributeSpanKind, "producer", nil)

	txn.End()

	expectedApp.ExpectTxnTraces(t, []internal.WantTxnTrace{
		{
			AgentAttributes: map[string]interface{}{
				newrelic.AttributeSpanKind: "producer",
			},
		},
	})
}
