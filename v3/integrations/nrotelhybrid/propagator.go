package nrotelhybrid

import (
	"context"
	"net/http"

	"github.com/newrelic/go-agent/v3/newrelic"
	"go.opentelemetry.io/otel/propagation"
)

type hybridPropagator struct {
	original propagation.TraceContext
}

func NewHybridPropagator() propagation.TextMapPropagator {
	return hybridPropagator{
		original: propagation.TraceContext{},
	}
}

func (h hybridPropagator) Fields() []string {
	return h.original.Fields()
}

func (h hybridPropagator) Extract(ctx context.Context, c propagation.TextMapCarrier) context.Context {
	return h.original.Extract(ctx, c)
}

// Inject writes New Relic's W3C trace context headers for the transaction in
// ctx. Without a transaction, or if the agent writes no headers (for example
// distributed tracing is disabled), it falls back to standard OTel behavior.
func (h hybridPropagator) Inject(ctx context.Context, c propagation.TextMapCarrier) {
	txn := h.txnFromSpanContext(ctx)
	if txn == nil {
		h.original.Inject(ctx, c)
		return
	}
	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)
	if len(hdrs) == 0 {
		h.original.Inject(ctx, c)
		return
	}
	for _, k := range h.Fields() {
		if v := hdrs.Get(k); v != "" {
			c.Set(k, v)
		}
	}
}

func (h hybridPropagator) txnFromSpanContext(ctx context.Context) *newrelic.Transaction {
	txn := newrelic.FromContext(ctx)
	if txn != nil {
		return txn
	}
	return nil
}
