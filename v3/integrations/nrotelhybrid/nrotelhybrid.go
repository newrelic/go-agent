package nrotelhybrid

import (
	"context"
	"net/url"
	"sync"

	"github.com/newrelic/go-agent/v3/newrelic"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type txnMapEntry struct {
	txn    *newrelic.Transaction
	spanID oteltrace.SpanID
}

type nrSegment interface {
	End()
	AddAttribute(key string, val interface{})
}

type nrotelhybridProcessor struct {
	app        *newrelic.Application
	mu         sync.Mutex
	txnMap     map[oteltrace.TraceID][]txnMapEntry // Trace ID -> stack of Transactions
	segmentMap map[oteltrace.SpanID]nrSegment      // SpanID -> Segment
	txnChecker func(txnMap map[oteltrace.TraceID][]txnMapEntry, traceID oteltrace.TraceID, spanID oteltrace.SpanID) bool
}

func NewHybridProcessor(app *newrelic.Application) *nrotelhybridProcessor {
	return &nrotelhybridProcessor{
		app:        app,
		txnMap:     map[oteltrace.TraceID][]txnMapEntry{},
		segmentMap: map[oteltrace.SpanID]nrSegment{},
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
		p.startTransaction(s, isWeb)
		return
	}
	// start the segment with the txn entry
	if entries := p.txnMap[s.SpanContext().TraceID()]; len(entries) > 0 {
		if entry := entries[len(entries)-1]; entry.txn != nil {
			p.startSegment(s, entry)
		}
	}
}

func (p *nrotelhybridProcessor) startTransaction(s trace.ReadWriteSpan, isWeb bool) {
	txn := p.app.StartTransaction(s.Name())
	if isWeb {
		attrs := s.Attributes()
		var fullURL string
		for _, attr := range attrs {
			if attr.Key == attribute.Key(AttrURLFull) {
				fullURL = attr.Value.AsString()
			}
		}
		req := newrelic.WebRequest{}
		nrURL, err := url.Parse(fullURL)
		if err == nil {
			req.URL = nrURL
		}
		txn.SetWebRequest(req)
	}
	traceID := s.SpanContext().TraceID()
	p.txnMap[traceID] = append(p.txnMap[traceID], txnMapEntry{txn, s.SpanContext().SpanID()})
}

func (p *nrotelhybridProcessor) startSegment(s trace.ReadWriteSpan, entry txnMapEntry) {
	seg := entry.txn.StartSegment(s.Name())
	p.segmentMap[s.SpanContext().SpanID()] = seg
}

func (p *nrotelhybridProcessor) OnEnd(s trace.ReadOnlySpan) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// use the trace id from trace.ReadOnlySpan to end the transaction
	traceID := s.SpanContext().TraceID()
	spanID := s.SpanContext().SpanID()

	if isTxn, _ := p.isTransaction(s.SpanKind(), s.SpanContext(), s.Parent()); isTxn {
		entries := p.txnMap[traceID]
		for i := len(entries) - 1; i >= 0; i-- {
			if entries[i].spanID == spanID {
				if entries[i].txn != nil {
					entries[i].txn.End()
				}
				entries = append(entries[:i], entries[i+1:]...)
				break
			}
		}
		if len(entries) == 0 {
			delete(p.txnMap, traceID)
		} else {
			p.txnMap[traceID] = entries
		}
		return
	}
	// otherwise end segment
	p.switchSegmentType(s)
	if seg, ok := p.segmentMap[spanID]; ok && seg != nil {
		for _, attr := range s.Attributes() {
			//  attr.Value.Type() switch on type
			seg.AddAttribute(string(attr.Key), extractAttributeValue(attr.Value))
		}
		seg.End()
		delete(p.segmentMap, spanID)
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

func (p *nrotelhybridProcessor) switchSegmentType(s trace.ReadOnlySpan) {
	var fullURL = ""
	segInterface, ok := p.segmentMap[s.SpanContext().SpanID()]
	if !ok {
		return
	}
	basicSegment, ok := segInterface.(*newrelic.Segment)
	if !ok {
		return
	}
	switch s.SpanKind() {
	case oteltrace.SpanKindClient:
		for _, attr := range s.Attributes() {
			if attr.Key == attribute.Key(AttrURLFull) {
				fullURL = attr.Value.AsString()
			}
			if attr.Key == attribute.Key(AttrDBSystemName) || attr.Key == attribute.Key(AttrDBSystem) {
				seg := &newrelic.DatastoreSegment{
					StartTime: basicSegment.StartTime,
				}
				// map attribtues for db
				p.segmentMap[s.SpanContext().SpanID()] = seg
				return
			}
		}
		seg := &newrelic.ExternalSegment{
			StartTime: basicSegment.StartTime,
		}
		seg.URL = fullURL
		p.segmentMap[s.SpanContext().SpanID()] = seg
	default:
		return
	}
}

func isWithinTransaction(txnMap map[oteltrace.TraceID][]txnMapEntry, traceID oteltrace.TraceID, spanID oteltrace.SpanID) bool {
	// if the innermost active transaction exists and is not the same span id, it is within an existing transaction
	if entries := txnMap[traceID]; len(entries) > 0 {
		return entries[len(entries)-1].spanID != spanID
	}
	return false
}

func extractAttributeValue(val attribute.Value) any {

	switch val.Type() {
	case attribute.BOOL:
		return val.AsBool()
	case attribute.INT64:
		return val.AsInt64()
	case attribute.FLOAT64:
		return val.AsFloat64()
	case attribute.STRING:
		return val.AsString()
	case attribute.BOOLSLICE:
		return val.AsBoolSlice()
	case attribute.INT64SLICE:
		return val.AsInt64Slice()
	case attribute.FLOAT64SLICE:
		return val.AsFloat64Slice()
	case attribute.STRINGSLICE:
		return val.AsStringSlice()
	case attribute.BYTESLICE:
		return val.AsByteSlice()
	case attribute.SLICE:
		return val.AsSlice()
	default:
		return nil // EMPTY OR INVALID
	}

}
