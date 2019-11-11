package integrationsupport

import (
	"testing"

	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/internal"
)

func TestNilTransaction(t *testing.T) {
	var txn *newrelic.Transaction

	AddAgentAttribute(txn, internal.AttributeHostDisplayName, "hostname", nil)
	AddAgentSpanAttribute(txn, internal.SpanAttributeAWSOperation, "operation")
}

func TestEmptyTransaction(t *testing.T) {
	txn := &newrelic.Transaction{}

	AddAgentAttribute(txn, internal.AttributeHostDisplayName, "hostname", nil)
	AddAgentSpanAttribute(txn, internal.SpanAttributeAWSOperation, "operation")
}

func TestSuccess(t *testing.T) {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("appname"),
		newrelic.ConfigLicense("0123456789012345678901234567890123456789"),
		newrelic.ConfigEnabled(false),
		newrelic.ConfigDistributedTracerEnabled(true),
	)
	if nil != err {
		t.Fatal(err)
	}
	replyfn := func(reply *internal.ConnectReply) {
		reply.AdaptiveSampler = internal.SampleEverything{}
	}
	internal.HarvestTesting(app.Private, replyfn)

	txn := app.StartTransaction("hello")
	AddAgentAttribute(txn, internal.AttributeHostDisplayName, "hostname", nil)
	segment := txn.StartSegment("mySegment")
	AddAgentSpanAttribute(txn, internal.SpanAttributeAWSOperation, "operation")
	segment.End()
	txn.End()

	app.Private.(internal.Expect).ExpectTxnEvents(t, []internal.WantEvent{
		{
			AgentAttributes: map[string]interface{}{
				newrelic.AttributeHostDisplayName: "hostname",
			},
		},
	})
	app.Private.(internal.Expect).ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":          "OtherTransaction/Go/hello",
				"category":      "generic",
				"nr.entryPoint": true,
			},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"name":     "Custom/mySegment",
				"parentId": internal.MatchAnything,
				"category": "generic",
			},
			AgentAttributes: map[string]interface{}{
				newrelic.SpanAttributeAWSOperation: "operation",
			},
		},
	})
}
