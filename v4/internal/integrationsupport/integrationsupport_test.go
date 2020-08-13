// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package integrationsupport

import (
	"sync"
	"testing"

	"github.com/newrelic/go-agent/v4/internal"
	"github.com/newrelic/go-agent/v4/newrelic"
)

func TestNilTransaction(t *testing.T) {
	var txn *newrelic.Transaction

	AddAgentAttribute(txn, newrelic.AttributeHostDisplayName, "hostname", nil)
	AddAgentSpanAttribute(txn, newrelic.SpanAttributeAWSOperation, "operation")
}

func TestEmptyTransaction(t *testing.T) {
	txn := &newrelic.Transaction{}

	AddAgentAttribute(txn, newrelic.AttributeHostDisplayName, "hostname", nil)
	AddAgentSpanAttribute(txn, newrelic.SpanAttributeAWSOperation, "operation")
}

func TestSuccess(t *testing.T) {
	app := NewTestApp(nil)
	txn := app.StartTransaction("hello")
	AddAgentAttribute(txn, newrelic.AttributeHostDisplayName, "hostname", nil)
	segment := txn.StartSegment("mySegment")
	AddAgentSpanAttribute(txn, newrelic.SpanAttributeAWSOperation, "operation")
	segment.End()
	txn.End()

	app.ExpectSpanEvents(t, []internal.WantSpan{
		{
			Name:          "mySegment",
			ParentID:      internal.MatchAnyParent,
			SkipAttrsTest: true,
			Attributes: map[string]interface{}{
				"parentId":                         internal.MatchAnything,
				"category":                         "generic",
				newrelic.SpanAttributeAWSOperation: "operation",
			},
		},
		{
			Name:          "hello",
			ParentID:      internal.MatchNoParent,
			SkipAttrsTest: true,
			Attributes: map[string]interface{}{
				"transaction.name": "OtherTransaction/Go/hello",
				"category":         "generic",
				"nr.entryPoint":    true,
				"host.displayName": "hostname",
			},
		},
	})
}

func TestConcurrentCalls(t *testing.T) {
	// This test will fail with a data race if the txn is not properly locked
	app := NewTestApp(nil)
	txn := app.StartTransaction("hello")
	defer txn.End()
	defer txn.StartSegment("mySegment").End()

	var wg sync.WaitGroup
	addAttr := func() {
		AddAgentSpanAttribute(txn, newrelic.SpanAttributeAWSOperation, "operation")
		wg.Done()
	}

	wg.Add(1)
	go addAttr()
	wg.Wait()
}
