package nrotelhybrid

import (
	"context"
	"net/http"

	"github.com/newrelic/go-agent/v3/newrelic"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type hybridPropagator struct {
	original  propagation.TraceContext
	processor *nrotelhybridProcessor
}

func NewHybridPropagator(processor *nrotelhybridProcessor) propagation.TextMapPropagator {
	return hybridPropagator{
		original:  propagation.TraceContext{},
		processor: processor,
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
	h.processor.mu.Lock()
	defer h.processor.mu.Unlock()
	spanContext := oteltrace.SpanContextFromContext(ctx)
	// look up segment first then look for txn
	if segment, ok := h.processor.segmentMap[spanContext.SpanID()]; ok {
		return segment.txn
	}
	if txnEntries, ok := h.processor.txnMap[spanContext.TraceID()]; ok {
		for _, entry := range txnEntries {
			if spanContext.SpanID() == entry.spanID {
				return entry.txn
			}
		}
	}
	return nil
}
