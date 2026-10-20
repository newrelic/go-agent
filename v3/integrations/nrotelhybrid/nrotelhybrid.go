package nrotelhybrid

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
	"go.opentelemetry.io/otel/trace/noop"
)

const spanEventEventsDroppedMetricName = "Supportability/Go/SpanEvent/Events/Dropped"

// w3cVersion is the version of the W3C trace context traceparent header format
// that this package emits.
const w3cVersion = "00"

// slice of entry in nrotelhybridProcessor txnMap
type txnMapEntry struct {
	txn    *newrelic.Transaction
	spanID oteltrace.SpanID
}

// entry in nrotelhybridProcessor segmentMap
type segmentEntry struct {
	seg nrSegment
	txn *newrelic.Transaction
}

// interface to describe different segments on the new relic side
// eventually, the type will be extracted to populate the correct
// attributes
type nrSegment interface {
	End()
	AddAttribute(key string, val any)
	AddLink(spanID, traceID string, start time.Time)
	AddOtelSpanID(spanID string)
	AddSpanEventEvent(name string, start time.Time, attrs newrelic.SpanEventAttributes)
}

// tracer that implements oteltrace.Tracer
type hybridTracer struct {
	embedded.Tracer
	tracer oteltrace.Tracer
}

// Returns this package's implementation of oteltrace.Tracer
func Tracer(name string, opts ...oteltrace.TracerOption) oteltrace.Tracer {
	return &hybridTracer{
		tracer: otel.Tracer(name, opts...),
	}
}

// hybridTracer defines start logic as returning a noop.Span{} when it is decided that an OTel span should
// not be started.  Otherwise, call the original tracer Start.
func (h *hybridTracer) Start(ctx context.Context, spanName string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	if !shouldStartOTelSpan(ctx, opts) {
		return ctx, noop.Span{}
	}
	return h.tracer.Start(ctx, spanName, opts...)
}

// shouldStartOTelSpan will decide whether or not an OTel span should be started given the current
// layout of New Relic transactions and OTel span context.  A span should be started if it exists
// within a NR Transaction, it has a parent span, or it is a consumer or server span.
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

// nrotelhybridProcessor implements the interface trace.SpanProcessor.  It keeps state to keep
// track of transactions and segments that share context with OTel spans.
type nrotelhybridProcessor struct {
	app        *newrelic.Application
	mu         sync.Mutex
	txnMap     map[oteltrace.TraceID][]txnMapEntry // Trace ID -> stack of Transactions
	segmentMap map[oteltrace.SpanID]segmentEntry   // SpanID -> Segment
	txnChecker func(txnMap map[oteltrace.TraceID][]txnMapEntry, traceID oteltrace.TraceID, spanID oteltrace.SpanID) bool
}

// Returns this package's implementation of a trace.SpanProcessor if OTel is enabled.
func NewHybridProcessor(app *newrelic.Application) *nrotelhybridProcessor {
	conf, _ := app.Config()
	if !conf.OpenTelemetry.Enabled {
		return nil
	}
	return &nrotelhybridProcessor{
		app:        app,
		txnMap:     map[oteltrace.TraceID][]txnMapEntry{},
		segmentMap: map[oteltrace.SpanID]segmentEntry{},
		txnChecker: isWithinTransaction,
	}
}

// Defines what to do and what kind of New Relic event is created when an OTel span is started.
// If a span is considered a transaction based on its context, a transaction will be started.
// Otherwise, a segment will be started.
func (p *nrotelhybridProcessor) OnStart(ctx context.Context, s trace.ReadWriteSpan) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	// check if span is considered a transaction
	if isTxn, isWeb := p.isTransaction(s.SpanKind(), s.SpanContext(), s.Parent(), ctx); isTxn {
		p.startTransaction(s, isWeb)
		return
	}

	// if not, start the segment with the txn entry
	if entries := p.txnMap[s.SpanContext().TraceID()]; len(entries) > 0 {
		if entry := entries[len(entries)-1]; entry.txn != nil {
			p.startSegment(s, entry)
			return
		}
	}

	// if no txn entry start it with the txn in context
	if txn := newrelic.FromContext(ctx); txn != nil {
		p.startSegment(s, txnMapEntry{txn: txn, spanID: s.SpanContext().SpanID()})
	}

}

// Call that begins a transaction within the nrotelhybridProcessor.  The New Relic
// transaction is started with the name of the OTel span.  It is then checked for trace
// context to be used in distributed tracing.  If it is considered a web transaction, it
// will begin a web request.  Finally, the transaction is added to the txnMap in the
// nrotelhybridProcessor state.
func (p *nrotelhybridProcessor) startTransaction(s trace.ReadWriteSpan, isWeb bool) {
	if p == nil {
		return
	}
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
	p.txnMap[traceID] = append(p.txnMap[traceID], txnMapEntry{txn: txn, spanID: s.SpanContext().SpanID()})
}

// Call that begins a segment within the nrotelhybridProcessor.  The New Relic
// segment is started with the OTel span name.  Then, it is added to the segmentMap
// in the nrotelhybridProcessor state.
func (p *nrotelhybridProcessor) startSegment(s trace.ReadWriteSpan, entry txnMapEntry) {
	if p == nil {
		return
	}
	seg := entry.txn.StartSegment(s.Name())
	otelSpanID := s.SpanContext().SpanID()
	seg.AddOtelSpanID(otelSpanID.String())
	p.segmentMap[otelSpanID] = segmentEntry{
		seg: seg,
		txn: entry.txn,
	}
}

// Defines what to do with New Relic events when an OTel span is ended.  The span is first
// checked for exceptions, which will produce an error on the transaction that it occurred.
// If the span appears in the txnMap, it will be ended.   Otherwise, it will checked for in the
// segmentMap and subsequently ended.
func (p *nrotelhybridProcessor) OnEnd(s trace.ReadOnlySpan) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	// use the trace id from trace.ReadOnlySpan to end the transaction
	traceID := s.SpanContext().TraceID()
	spanID := s.SpanContext().SpanID()
	events := s.Events()

	var transactionsToEnd []*newrelic.Transaction
	defer func() {
		for _, txn := range transactionsToEnd {
			if txn != nil {
				txn.End()
			}
		}
	}()

	nrErr, hasErr := exceptionFromSpan(s)

	// a span is a transaction only if OnStart recorded it in txnMap
	entries := p.txnMap[traceID]
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].spanID != spanID {
			continue
		}
		if entries[i].txn != nil {
			if hasErr {
				entries[i].txn.NoticeError(nrErr)
			}
			if len(events) > 0 {
				for j, event := range events {
					if j > 99 {
						p.app.RecordCustomMetric(spanEventEventsDroppedMetricName, 1.0)
						continue
					}
					entries[i].txn.AddSpanEventEvent(event.Name, event.Time, otelEventAttributes(event.Attributes))
				}
			}
			transactionsToEnd = append(transactionsToEnd, entries[i].txn)
		}
		entries = append(entries[:i], entries[i+1:]...)
		if len(entries) == 0 {
			delete(p.txnMap, traceID)
		} else {
			p.txnMap[traceID] = entries
		}
		break
	}
	// otherwise end segment if it exists in the map
	p.switchSegmentType(spanID, s.Attributes(), s.SpanKind())

	if segEntry, ok := p.segmentMap[spanID]; ok && segEntry.seg != nil {
		if hasErr && segEntry.txn != nil {
			segEntry.txn.NoticeError(nrErr)
		}
		if len(events) > 0 {
			for i, event := range events {
				if i > 99 {
					p.app.RecordCustomMetric(spanEventEventsDroppedMetricName, 1.0)
					continue
				}
				segEntry.seg.AddSpanEventEvent(event.Name, event.Time, otelEventAttributes(event.Attributes))
			}
		}
		segEntry.seg.End()
		delete(p.segmentMap, spanID)
	}
}

// checks events on an OTel span for exceptions.  If they exist, the New Relic Error
// type is populated with the necessary attributes.
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
func (p *nrotelhybridProcessor) isTransaction(kind oteltrace.SpanKind, current oteltrace.SpanContext, parent oteltrace.SpanContext, ctx context.Context) (isTxn, isWeb bool) {
	if p == nil {
		return false, false
	}
	if newrelic.FromContext(ctx) != nil {
		return false, true // within an existing transaction
	}
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

// switches the basic Segment started by the nrotelhybridProcessor into different
// New Relic segments based on span kind.  This also defines what kind of attributes
// will get added to each segment.  This runs within OnEnd in the nrotelhybridProcessor.
func (p *nrotelhybridProcessor) switchSegmentType(spanID oteltrace.SpanID, attributes []attribute.KeyValue, spanKind oteltrace.SpanKind) {
	if p == nil {
		return
	}
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

// This add attributes based on the type of segment that is found.  These attributes exist within a
// constant map and will need to be updated when the spec outlines semconv changes.
func (p *nrotelhybridProcessor) addSegmentAttributes(seg nrSegment, attributes []attribute.KeyValue, attrMap map[string]string) {
	if p == nil {
		return
	}
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

// a special function to check the constant attribute maps using the attribute.Key
func checkMap(key attribute.Key, attrMap map[string]string) (string, bool) {
	if attrMap == nil {
		return "", false
	}
	nrAttribute, ok := attrMap[string(key)]
	return nrAttribute, ok

}

// checks if an OTel spans exists within a transaction
func isWithinTransaction(txnMap map[oteltrace.TraceID][]txnMapEntry, traceID oteltrace.TraceID, spanID oteltrace.SpanID) bool {
	// if the innermost active transaction exists and is not the same span id, it is within an existing transaction
	if entries := txnMap[traceID]; len(entries) > 0 {
		return entries[len(entries)-1].spanID != spanID
	}
	return false
}

// otelEventAttributes adapts an OTEL event's attributes to
// newrelic.SpanEventAttributes, deferring conversion until the SpanEvent
// event is actually serialized.
type otelEventAttributes []attribute.KeyValue

func (a otelEventAttributes) WriteAttributes(write func(key string, val any)) {
	for _, attr := range a {
		write(string(attr.Key), extractAttributeValue(attr.Value))
	}
}

// get the specific value based on the type for attribute.Value
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
