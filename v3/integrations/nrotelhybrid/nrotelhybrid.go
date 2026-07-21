package nrotelhybrid

import (
	"context"

	"github.com/newrelic/go-agent/v3/newrelic"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type txnMapEntry struct {
	txn    *newrelic.Transaction
	spanID oteltrace.SpanID
}

type nrotelhybridProcessor struct {
	app        *newrelic.Application
	txnMap     map[oteltrace.TraceID]txnMapEntry // Trace ID -> Transaction
	txnChecker func(txnMap map[oteltrace.TraceID]txnMapEntry, traceID oteltrace.TraceID, spanID oteltrace.SpanID) bool
}

func (p *nrotelhybridProcessor) OnStart(ctx context.Context, s trace.ReadWriteSpan) {
	// for now pretending like everything is enabled
	// first check for case when span has a remote parent.  In this case, it must create a new transaction

	// check if remote parent
	// should be a valid span context and be marked as remote
	// this begins a transaction
	if p.isTransaction(s.SpanKind(), s.SpanContext(), s.Parent()) {
		txn := p.app.StartTransaction(s.Name())
		p.txnMap[s.SpanContext().TraceID()] = txnMapEntry{txn, s.SpanContext().SpanID()}
	}

}

func (p *nrotelhybridProcessor) OnEnd(s trace.ReadOnlySpan) {
	// use the trace id from trace.ReadOnlySpan to end the transaction
	if p.isTransaction(s.SpanKind(), s.SpanContext(), s.Parent()) {
		if val, ok := p.txnMap[s.SpanContext().TraceID()]; ok && val.txn != nil {
			val.txn.End()
			delete(p.txnMap, s.SpanContext().TraceID())
		}
	}
}

func (p *nrotelhybridProcessor) Shutdown(ctx context.Context) error {
	return nil
}

func (p *nrotelhybridProcessor) ForceFlush(ctx context.Context) error {
	return nil
}

func (p *nrotelhybridProcessor) isTransaction(kind oteltrace.SpanKind, current oteltrace.SpanContext, parent oteltrace.SpanContext) bool {
	if parent.IsRemote() {
		// any span with a remote parent is a transaction
		return true
	}
	// check if within a transaction
	switch kind {
	case oteltrace.SpanKindUnspecified:
		return false
	case oteltrace.SpanKindInternal:
		return false
	case oteltrace.SpanKindServer:
		return !p.txnChecker(p.txnMap, current.TraceID(), current.SpanID())
	case oteltrace.SpanKindClient:
		return false
	case oteltrace.SpanKindProducer:
		return false
	case oteltrace.SpanKindConsumer:
		return !p.txnChecker(p.txnMap, current.TraceID(), current.SpanID())
	default:
		return false
	}
}

func isWithinTransaction(txnMap map[oteltrace.TraceID]txnMapEntry, traceID oteltrace.TraceID, spanID oteltrace.SpanID) bool {
	// if the transaction exists and is not the same span id, it is within an existing transaction
	if val, ok := txnMap[traceID]; ok {
		return val.spanID != spanID
	}
	return false
}
