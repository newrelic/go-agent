package nrotelhybrid

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"

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
	app    *newrelic.Application
}

// wrapper for trace.TracerProvider
type hybridTracerProvider struct {
	embedded.TracerProvider
	provider *trace.TracerProvider
	app      *newrelic.Application
}

type hybridSpan struct {
	oteltrace.Span                       // inner SDK span
	spanContext    oteltrace.SpanContext // NR IDs
	txn            *newrelic.Transaction // set for a root span
	seg            *newrelic.Segment     // set for a child span
	endOnce        sync.Once
}

// TracerProvider is an oteltrace.TracerProvider that also exposes the lifecycle
// methods of the SDK provider it wraps.
type TracerProvider interface {
	oteltrace.TracerProvider
	ForceFlush(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

// NewTracerProvider wraps a new SDK trace.TracerProvider so that the tracers it hands out
// bridge OTel spans to New Relic transactions and segments on the given application.
func NewTracerProvider(app *newrelic.Application, opts ...trace.TracerProviderOption) TracerProvider {
	return &hybridTracerProvider{
		app:      app,
		provider: trace.NewTracerProvider(opts...),
	}
}

// Returns this package's implementation of oteltrace.Tracer.  The inner tracer comes from the
// wrapped SDK provider, so setting this provider as the global one does not recurse.
func (h *hybridTracerProvider) Tracer(name string, opts ...oteltrace.TracerOption) oteltrace.Tracer {
	return &hybridTracer{
		tracer: h.provider.Tracer(name, opts...),
		app:    h.app,
	}
}

func (h *hybridTracerProvider) Shutdown(ctx context.Context) error {
	return h.provider.Shutdown(ctx)
}

func (h *hybridTracerProvider) ForceFlush(ctx context.Context) error {
	return h.provider.ForceFlush(ctx)
}

// hybridTracer defines start logic as returning a noop.Span{} when it is decided that an OTel span should
// not be started.  Otherwise, call the original tracer Start.
func (h *hybridTracer) Start(ctx context.Context, spanName string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	if !shouldStartOTelSpan(ctx, opts) {
		return ctx, noop.Span{}
	}

	ctx, span := h.tracer.Start(ctx, spanName, opts...)

	readWriteSpan, ok := span.(trace.ReadWriteSpan)
	if !ok || !span.IsRecording() {
		return ctx, span
	}
	config := oteltrace.NewSpanStartConfig(opts...)
	hybridNRSpan := &hybridSpan{
		Span:        span,
		spanContext: span.SpanContext(),
	}
	// check if span is considered a transaction
	isTxn, isWeb := isTransaction(readWriteSpan.SpanKind(), readWriteSpan.Parent(), config.NewRoot(), ctx)
	switch {
	case isTxn:
		hybridNRSpan.startTransaction(ctx, readWriteSpan, isWeb, h.app)
		ctx = newrelic.NewContext(ctx, hybridNRSpan.txn)
	case newrelic.FromContext(ctx) != nil:
		hybridNRSpan.startSegment(ctx, readWriteSpan, newrelic.FromContext(ctx))
	default:
		return ctx, span
	}

	return oteltrace.ContextWithSpan(ctx, hybridNRSpan), hybridNRSpan
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

func (p *hybridSpan) startTransaction(ctx context.Context, s trace.ReadWriteSpan, isWeb bool, app *newrelic.Application) {
	if p == nil {
		return
	}
	txn := app.StartTransaction(s.Name())
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
	p.txn = txn

	traceId := s.SpanContext().TraceID()
	spanId := s.SpanContext().SpanID()
	traceId, err := oteltrace.TraceIDFromHex(txn.GetLinkingMetadata().TraceID)
	if err != nil {
		return
	}
	spanId, err = oteltrace.SpanIDFromHex(txn.GetLinkingMetadata().SpanID)
	if err != nil {
		return
	}
	p.SetSpanContext(s.SpanContext().WithTraceID(traceId).WithSpanID(spanId))
}

func (p *hybridSpan) startSegment(ctx context.Context, s trace.ReadWriteSpan, parentTxn *newrelic.Transaction) {
	if p == nil {
		return
	}
	seg := parentTxn.StartSegment(s.Name())
	otelSpanID := s.SpanContext().SpanID()
	seg.AddOtelSpanID(otelSpanID.String())
	p.seg = seg
}

func (p *hybridSpan) SetSpanContext(sc oteltrace.SpanContext) {
	p.spanContext = sc
}

// Defines what to do with New Relic events when an OTel span is ended.  The span is first
// checked for exceptions, which will produce an error on the transaction that it occurred.
// If the span appears in the txnMap, it will be ended.  Otherwise, it will be checked for in the
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
	links := s.Links()
	events := s.Events()

	if key, ok := p.txnMapKey[spanID]; ok { // spanID is the SDK span ID
		traceID = key
		delete(p.txnMapKey, spanID) // cleaned up when the root ends, so it stays bounded
	}
	entries := p.txnMap[traceID]

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
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].spanID != spanID {
			continue
		}
		if entries[i].txn != nil {
			if hasErr {
				entries[i].txn.NoticeError(nrErr)
			}
			if len(links) > 0 {
				for _, link := range links {
					entries[i].txn.AddLink(link.SpanContext.SpanID().String(), link.SpanContext.TraceID().String(), s.StartTime())
				}
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
		if len(links) > 0 {
			for _, link := range links {
				segEntry.seg.AddLink(link.SpanContext.SpanID().String(), link.SpanContext.TraceID().String(), s.StartTime())
			}
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

// isTransaction reports whether the span should start a transaction (isTxn),
// and whether that transaction is a web transaction (isWeb).  A span inside an existing
// transaction is a segment of it, unless it is a server or consumer span started as a new
// root (oteltrace.WithNewRoot), which begins its own transaction.
func isTransaction(kind oteltrace.SpanKind, parent oteltrace.SpanContext, newRoot bool, ctx context.Context) (isTxn, isWeb bool) {
	inTxn := newrelic.FromContext(ctx) != nil
	startsOwnTxn := newRoot && (kind == oteltrace.SpanKindServer || kind == oteltrace.SpanKindConsumer)
	if inTxn && !startsOwnTxn {
		return false, false // segment of the existing transaction
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
		return true, true
	case oteltrace.SpanKindConsumer:
		return true, false
	default: // client, producer, internal
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

// This adds attributes based on the type of segment that is found.  These attributes exist within a
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
				// messaging.destination_kind is deprecated on the OTEL side but leaving it here for spec compatibility
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
