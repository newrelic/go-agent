package nrotelhybrid

import (
	"context"

	"github.com/newrelic/go-agent/v3/newrelic"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type nrotelhybridProcessor struct {
	app    *newrelic.Application
	txnMap map[oteltrace.TraceID]*newrelic.Transaction // Trace ID -> Transaction
}

func (p *nrotelhybridProcessor) OnStart(ctx context.Context, s trace.ReadWriteSpan) {
	// for now pretending like everything is enabled
	// first check for case when span has a remote parent.  In this case, it must create a new transaction

	// check if remote parent
	// should be a valid span context and be marked as remote
	// this begins a transaction
	if isTransaction(s.Parent()) {
		txn := p.app.StartTransaction(s.Name())
		p.txnMap[s.SpanContext().TraceID()] = txn
	}

}

func (p *nrotelhybridProcessor) OnEnd(s trace.ReadOnlySpan) {
	// use the trace id from trace.ReadOnlySpan to end the transaction
	if isTransaction(s.Parent()) {
		txn := p.txnMap[s.SpanContext().TraceID()]
		if txn != nil {
			txn.End()
		}
	}
}

func (p *nrotelhybridProcessor) Shutdown(ctx context.Context) error {
	return nil
}

func (p *nrotelhybridProcessor) ForceFlush(ctx context.Context) error {
	return nil
}

func isTransaction(parent oteltrace.SpanContext) bool {
	return !parent.IsValid() || parent.IsRemote()
}
