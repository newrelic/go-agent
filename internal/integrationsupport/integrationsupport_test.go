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
