package nrotelhybrid

import (
	"context"

	"github.com/newrelic/go-agent/v3/newrelic"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type nrotelhybridProcessor struct {
	app    *newrelic.Application
	txnMap map[oteltrace.TraceID]struct {
		txn    *newrelic.Transaction
		spanID oteltrace.SpanID
	} // Trace ID -> Transaction
}

func (p *nrotelhybridProcessor) OnStart(ctx context.Context, s trace.ReadWriteSpan) {
	// for now pretending like everything is enabled
	// first check for case when span has a remote parent.  In this case, it must create a new transaction

	// check if remote parent
	// should be a valid span context and be marked as remote
	// this begins a transaction
	if p.isTransaction(s.SpanKind(), s.SpanContext(), s.Parent()) {
		txn := p.app.StartTransaction(s.Name())
		p.txnMap[s.SpanContext().TraceID()] = struct {
			txn    *newrelic.Transaction
			spanID oteltrace.SpanID
		}{txn, s.SpanContext().SpanID()}
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
		// if valid remote parent
		return parent.IsValid()
	}
	// check if within a transaction
	switch kind {
	case oteltrace.SpanKindUnspecified:
		return false
	case oteltrace.SpanKindInternal:
		return false
	case oteltrace.SpanKindServer:
		return !p.isWithinTransaction(current.TraceID(), current.SpanID())
	case oteltrace.SpanKindClient:
		return false
	case oteltrace.SpanKindProducer:
		return false
	case oteltrace.SpanKindConsumer:
		return !p.isWithinTransaction(current.TraceID(), current.SpanID())
	default:
		return false

	}
}

func (p *nrotelhybridProcessor) isWithinTransaction(traceID oteltrace.TraceID, spanID oteltrace.SpanID) bool {
	if val, ok := p.txnMap[traceID]; ok {
		return val.spanID != spanID
	}
	return false
}
