// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package integrationsupport

import (
	"context"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
	"net/http"
	"testing"
	"time"
)

type myError struct{}

func (e myError) Error() string { return "My error message" }

func TestNewBasicTestApp(t *testing.T) {
	expectedApp := NewBasicTestApp()
	txn := expectedApp.Application.StartTransaction("MyTransaction")
	txn.NoticeError(myError{})
	txn.End()
	expectedApp.ExpectErrors(t, []internal.WantError{{
		Msg: "My error message",
	}})
}

func TestDistributedTracingTestApp(t *testing.T) {

	expectedApp := NewTestApp(SampleEverythingReplyFn, DTEnabledCfgFn)
	txn := expectedApp.Application.StartTransaction("MyTransaction")

	client := &http.Client{}
	client.Transport = newrelic.NewRoundTripper(client.Transport)

	request, _ := http.NewRequest("GET", "https://example.com", nil)

	request = request.WithContext(newrelic.NewContext(context.Background(), txn))

	_, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
		return
	}

	time.Sleep(2 * time.Second) // Sleep to exceed trace threshold
	txn.End()

	expectedApp.ExpectMetrics(t, []internal.WantMetric{
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransaction/Go/MyTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/MyTransaction", Scope: "", Forced: false, Data: nil},
		{Name: "External/all", Scope: "", Forced: true, Data: nil},
		{Name: "External/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "External/example.com/all", Scope: "", Forced: false, Data: nil},
		{Name: "External/example.com/http/GET", Scope: "OtherTransaction/Go/MyTransaction", Forced: false, Data: nil},
	})

	expectedApp.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName:  "OtherTransaction/Go/MyTransaction",
		NumSegments: 1,
	}})
}

func TestAppLogsTestApp(t *testing.T) {
	expectedApp := NewTestApp(SampleEverythingReplyFn, AppLogEnabledCfgFn)
	txn := expectedApp.Application.StartTransaction("MyTransaction")
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
	txn := expectedApp.Application.StartTransaction("MyTransaction")

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
