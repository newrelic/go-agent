// Package integrationsupport exists to expose functionality to integration
// packages without adding noise to the public API.
package integrationsupport

import (
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/internal"
)

// AddAgentAttribute allows instrumentation packages to add agent attributes.
func AddAgentAttribute(txn newrelic.Transaction, id internal.AgentAttributeID, stringVal string, otherVal interface{}) {
	if aa, ok := txn.(internal.AddAgentAttributer); ok {
		aa.AddAgentAttribute(id, stringVal, otherVal)
	}
}

// AddAgentSpanAttribute allows instrumentation packages to add span attributes.
func AddAgentSpanAttribute(txn newrelic.Transaction, key internal.SpanAttribute, val string) {
	internal.AddAgentSpanAttribute(txn, key, val)
}
