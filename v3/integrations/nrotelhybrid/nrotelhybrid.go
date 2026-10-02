package nrotelhybrid

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/newrelic/go-agent/v3/newrelic"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
	"go.opentelemetry.io/otel/trace/noop"
)

// w3cVersion is the version of the W3C trace context traceparent header format
// that this package emits.
const w3cVersion = "00"

type txnMapEntry struct {
	txn    *newrelic.Transaction
	spanID oteltrace.SpanID
}

type segmentEntry struct {
	seg nrSegment
	txn *newrelic.Transaction
}

type nrSegment interface {
	End()
	AddAttribute(key string, val interface{})
}

type nrotelhybridProcessor struct {
	app        *newrelic.Application
	mu         sync.Mutex
	txnMap     map[oteltrace.TraceID][]txnMapEntry // Trace ID -> stack of Transactions
	segmentMap map[oteltrace.SpanID]segmentEntry   // SpanID -> Segment
	txnChecker func(txnMap map[oteltrace.TraceID][]txnMapEntry, traceID oteltrace.TraceID, spanID oteltrace.SpanID) bool
}

type hybridTracer struct {
	embedded.Tracer
	tracer oteltrace.Tracer
}

func (h *hybridTracer) Start(ctx context.Context, spanName string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	// decide if start
	// if does not meet start criteria{
	//			return ctx, RETURN NO-OP SPAN
	//}
	if !shouldStartOTelSpan(ctx, opts) {
		return ctx, noop.Span{}
	}
	return h.tracer.Start(ctx, spanName, opts...)
}

func Tracer(name string, opts ...oteltrace.TracerOption) oteltrace.Tracer {
	return &hybridTracer{
		tracer: otel.Tracer(name, opts...),
	}
}

func shouldStartOTelSpan(ctx context.Context, opts []oteltrace.SpanStartOption) bool {
	if newrelic.FromContext(ctx) != nil {
		return true // within an existing transaction
	}
	spanConfig := oteltrace.NewSpanStartConfig(opts...)
	parent := oteltrace.SpanContextFromContext(ctx)
	if !spanConfig.NewRoot() && parent.IsValid() {
		return true // has parent
	}
	switch spanConfig.SpanKind() {
	case oteltrace.SpanKindConsumer, oteltrace.SpanKindServer: // is a consumer or server span
		return true
	}
	return false
}

func NewHybridProcessor(app *newrelic.Application) *nrotelhybridProcessor {
	return &nrotelhybridProcessor{
		app:        app,
		txnMap:     map[oteltrace.TraceID][]txnMapEntry{},
		segmentMap: map[oteltrace.SpanID]segmentEntry{},
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

	// if no entry start it with the txn in context
	if txn := newrelic.FromContext(ctx); txn != nil {
		p.startSegment(s, txnMapEntry{txn: txn, spanID: s.SpanContext().SpanID()})
	}
}

func (p *nrotelhybridProcessor) startTransaction(s trace.ReadWriteSpan, isWeb bool) {
	txn := p.app.StartTransaction(s.Name())
	// A remote parent means an upstream service sent us trace context, so the
	// transaction must adopt the remote trace id and parent span id.
	if parent := s.Parent(); parent.IsValid() && parent.IsRemote() {
		transport := newrelic.TransportOther
		if isWeb {
			transport = newrelic.TransportHTTP
		}
		hdrs := http.Header{}
		hdrs.Set("traceparent", fmt.Sprintf("%s-%s-%s-%s", w3cVersion, parent.TraceID(), parent.SpanID(), parent.TraceFlags()))
		if ts := parent.TraceState().String(); ts != "" {
			hdrs.Set("tracestate", ts)
		}
		txn.AcceptDistributedTraceHeaders(transport, hdrs)
	}
	if isWeb {
		var fullURL string
		for _, attr := range s.Attributes() {
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
	p.segmentMap[s.SpanContext().SpanID()] = segmentEntry{
		seg: seg,
		txn: entry.txn,
	}
}

func (p *nrotelhybridProcessor) OnEnd(s trace.ReadOnlySpan) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// use the trace id from trace.ReadOnlySpan to end the transaction
	traceID := s.SpanContext().TraceID()
	spanID := s.SpanContext().SpanID()

	nrErr, hasErr := exceptionFromSpan(s)

	if isTxn, _ := p.isTransaction(s.SpanKind(), s.SpanContext(), s.Parent()); isTxn {
		entries := p.txnMap[traceID]
		for i := len(entries) - 1; i >= 0; i-- {
			if entries[i].spanID == spanID {
				if entries[i].txn != nil {
					if hasErr {
						entries[i].txn.NoticeError(nrErr)
					}
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
	// otherwise end segment if it exists in the map
	p.switchSegmentType(spanID, s.Attributes(), s.SpanKind())

	if segEntry, ok := p.segmentMap[spanID]; ok && segEntry.seg != nil {
		if hasErr && segEntry.txn != nil {
			segEntry.txn.NoticeError(nrErr)
		}
		// find type of segment to switch segment type and add attributes
		segEntry.seg.End()
		delete(p.segmentMap, spanID)
	}

}

func exceptionFromSpan(s trace.ReadOnlySpan) (newrelic.Error, bool) {
	for _, event := range s.Events() {
		if event.Name != AttrEventException {
			continue
		}
		var nrErr newrelic.Error
		for _, attr := range event.Attributes {
			switch string(attr.Key) {
			case AttrExceptionMessage:
				nrErr.Message = attr.Value.AsString()
			case AttrExceptionType:
				nrErr.Class = attr.Value.AsString()
			default:
				if nrErr.Attributes == nil {
					nrErr.Attributes = map[string]interface{}{}
				}
				nrErr.Attributes[string(attr.Key)] = attr.Value.AsString()
			}
		}
		return nrErr, true
	}
	return newrelic.Error{}, false
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
	case oteltrace.SpanKindProducer:
		return false, false
	case oteltrace.SpanKindConsumer:
		return !p.txnChecker(p.txnMap, current.TraceID(), current.SpanID()), false
	default:
		return false, false
	}
}

func (p *nrotelhybridProcessor) switchSegmentType(spanID oteltrace.SpanID, attributes []attribute.KeyValue, spanKind oteltrace.SpanKind) {
	segInterface, ok := p.segmentMap[spanID]
	if !ok {
		return
	}
	basicSegment, ok := segInterface.seg.(*newrelic.Segment)
	if !ok {
		return
	}

	switch spanKind {
	case oteltrace.SpanKindClient:
		for _, attr := range attributes {
			if attr.Key == attribute.Key(AttrDBSystemName) || attr.Key == attribute.Key(AttrDBSystem) {
				seg := &newrelic.DatastoreSegment{
					StartTime: basicSegment.StartTime,
				}
				// map attributes for db
				p.addSegmentAttributes(seg, attributes, OTELToNRDBAttributeMap)
				p.segmentMap[spanID] = segmentEntry{seg: seg, txn: segInterface.txn}
				return
			}
		}
		seg := &newrelic.ExternalSegment{
			StartTime: basicSegment.StartTime,
		}
		p.addSegmentAttributes(seg, attributes, OTELToNRHTTPAttributeMap)
		p.segmentMap[spanID] = segmentEntry{seg: seg, txn: segInterface.txn}
	case oteltrace.SpanKindProducer:
		seg := &newrelic.MessageProducerSegment{
			StartTime: basicSegment.StartTime,
		}
		p.addSegmentAttributes(seg, attributes, OTELToNRMessagingProducerAttributeMap)
		p.segmentMap[spanID] = segmentEntry{seg: seg, txn: segInterface.txn}
	case oteltrace.SpanKindConsumer:
		p.addSegmentAttributes(basicSegment, attributes, OTELToNRMessagingConsumerAttributeMap)
	default:
		p.addSegmentAttributes(basicSegment, attributes, nil)
	}
}

func (p *nrotelhybridProcessor) addSegmentAttributes(seg nrSegment, attributes []attribute.KeyValue, attrMap map[string]string) {
	switch s := seg.(type) {
	case *newrelic.DatastoreSegment:
		for _, attribute := range attributes {
			switch string(attribute.Key) {
			case AttrDBCollectionName, AttrDBSQLTable:
				s.Collection = attribute.Value.AsString()
			case AttrDBOperationName, AttrDBOperation:
				s.Operation = attribute.Value.AsString()
			case AttrDBStatement:
				s.ParameterizedQuery = attribute.Value.AsString()
			case AttrDBSystem, AttrDBSystemName:
				s.Product = newrelic.DatastoreProduct(attribute.Value.AsString())
			default:
				if nrAttribute, ok := checkMap(attribute.Key, attrMap); ok {
					s.AddAttribute(string(nrAttribute), extractAttributeValue(attribute.Value))
					continue
				}
				s.AddAttribute(string(attribute.Key), extractAttributeValue(attribute.Value))
			}
		}
	case *newrelic.ExternalSegment:
		for _, attribute := range attributes {
			switch string(attribute.Key) {
			case AttrURLFull, AttrHTTPURL:
				s.URL = attribute.Value.AsString()
			default:
				if nrAttribute, ok := checkMap(attribute.Key, attrMap); ok {
					s.AddAttribute(string(nrAttribute), extractAttributeValue(attribute.Value))
					continue
				}
				s.AddAttribute(string(attribute.Key), extractAttributeValue(attribute.Value))
			}
		}
	case *newrelic.MessageProducerSegment:
		for _, attribute := range attributes {
			switch string(attribute.Key) {
			case AttrMessagingSystem:
				s.Library = attribute.Value.AsString()
			case AttrMessagingDestinationName:
				s.DestinationName = attribute.Value.AsString()
			case AttrMessagingDestinationKind, AttrMessagingOperation, AttrMessagingOperationType:
				// messaging.desingation_kind this is deprecated on the OTEL side but leaving it here for spec compatibility
				s.DestinationType = newrelic.MessageDestinationType(attribute.Value.AsString())
			default:
				s.AddAttribute(string(attribute.Key), extractAttributeValue(attribute.Value))
			}

		}
	default:
		for _, attribute := range attributes {
			if nrAttribute, ok := checkMap(attribute.Key, attrMap); ok {
				seg.AddAttribute(string(nrAttribute), extractAttributeValue(attribute.Value))
				continue
			}
			seg.AddAttribute(string(attribute.Key), extractAttributeValue(attribute.Value))
		}
	}
}

func checkMap(key attribute.Key, attrMap map[string]string) (string, bool) {
	if attrMap == nil {
		return "", false
	}
	nrAttribute, ok := attrMap[string(key)]
	return nrAttribute, ok

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
