package nrotelhybrid

import (
	"context"
	"net/url"
	"sync"

	"github.com/newrelic/go-agent/v3/newrelic"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type txnMapEntry struct {
	txn    *newrelic.Transaction
	spanID oteltrace.SpanID
}

type nrotelhybridProcessor struct {
	app        *newrelic.Application
	mu         sync.Mutex
	txnMap     map[oteltrace.TraceID]txnMapEntry // Trace ID -> Transaction
	txnChecker func(txnMap map[oteltrace.TraceID]txnMapEntry, traceID oteltrace.TraceID, spanID oteltrace.SpanID) bool
}

func NewHybridProcessor(app *newrelic.Application) *nrotelhybridProcessor {
	return &nrotelhybridProcessor{
		app:        app,
		txnMap:     map[oteltrace.TraceID]txnMapEntry{},
		txnChecker: isWithinTransaction,
	}
}

func (p *nrotelhybridProcessor) OnStart(ctx context.Context, s trace.ReadWriteSpan) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// for now pretending like everything is enabled
	// first check for case when span has a remote parent.  In this case, it must create a new transaction

	// check if remote parent
	// should be a valid span context and be marked as remote
	// this begins a transaction
	if isTxn, isWeb := p.isTransaction(s.SpanKind(), s.SpanContext(), s.Parent()); isTxn {
		txn := p.app.StartTransaction(s.Name())
		if isWeb {
			attrs := s.Attributes()
			var fullURL string
			for _, attr := range attrs {
				if attr.Key == semconv.URLFullKey {
					fullURL = attr.Value.AsString()
				}
			}
			nrURL, err := url.Parse(fullURL)
			if err != nil {
				// debug log
			}
			txn.SetWebRequest(newrelic.WebRequest{
				URL: nrURL,
			})
		}
		p.txnMap[s.SpanContext().TraceID()] = txnMapEntry{txn, s.SpanContext().SpanID()}
	}

}

func (p *nrotelhybridProcessor) OnEnd(s trace.ReadOnlySpan) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// use the trace id from trace.ReadOnlySpan to end the transaction
	if isTxn, _ := p.isTransaction(s.SpanKind(), s.SpanContext(), s.Parent()); isTxn {
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

// isTransaction reports whether the span should start/continue a transaction (isTxn),
// and whether that transaction is a web transaction (isWeb).
func (p *nrotelhybridProcessor) isTransaction(kind oteltrace.SpanKind, current oteltrace.SpanContext, parent oteltrace.SpanContext) (isTxn, isWeb bool) {
	if parent.IsRemote() {
		// any span with a remote parent is a transaction
		switch kind {
		case oteltrace.SpanKindServer, oteltrace.SpanKindClient:
			return true, true
		default:
			return true, false
		}
	}
	switch kind {
	case oteltrace.SpanKindServer:
		return !p.txnChecker(p.txnMap, current.TraceID(), current.SpanID()), true
	case oteltrace.SpanKindClient:
		return false, true
	case oteltrace.SpanKindConsumer:
		return !p.txnChecker(p.txnMap, current.TraceID(), current.SpanID()), false
	default:
		return false, false
	}
}

func isWithinTransaction(txnMap map[oteltrace.TraceID]txnMapEntry, traceID oteltrace.TraceID, spanID oteltrace.SpanID) bool {
	// if the transaction exists and is not the same span id, it is within an existing transaction
	if val, ok := txnMap[traceID]; ok {
		return val.spanID != spanID
	}
	return false
}
