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

type hybridSegment struct {
	seg       nrSegment
	parentTxn *newrelic.Transaction
}

type hybridSpan struct {
	trace.ReadWriteSpan                       // inner SDK span
	spanContext         oteltrace.SpanContext // NR IDs
	txn                 *newrelic.Transaction // set for a root span
	seg                 *hybridSegment        // set for a child span
	endOnce             sync.Once
	app                 *newrelic.Application
}

// segmentSpan is the OTel view of a New Relic span: identity only.  It is
// non-recording and never exported; it exists so that OTel API calls made while
// a segment is open (SpanFromContext, starting child spans) see the segment's
// ids rather than those of the enclosing OTel span.
type nrSpan struct {
	noop.Span
	sc oteltrace.SpanContext
}

func (s nrSpan) SpanContext() oteltrace.SpanContext { return s.sc }

// nrSpanContextHook is registered on the application by NewTracerProvider and
// is invoked by Transaction.StartSegmentCtx or Application.StartTransactionCtx.  If the ids can't be parsed, ctx is
// returned unchanged.
func nrSpanContextHook(ctx context.Context, traceID, spanID string, sampled bool) context.Context {
	tid, err := oteltrace.TraceIDFromHex(traceID)
	if err != nil {
		return ctx
	}
	sid, err := oteltrace.SpanIDFromHex(spanID)
	if err != nil {
		return ctx
	}
	var flags oteltrace.TraceFlags
	if sampled {
		flags = oteltrace.FlagsSampled
	}
	sc := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    tid,
		SpanID:     sid,
		TraceFlags: flags,
	})
	return oteltrace.ContextWithSpan(ctx, nrSpan{sc: sc})
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
	// Make segments started with Transaction.StartSegmentCtx visible to the
	// OTel API as the current span.
	app.SetSegmentContextHook(nrSpanContextHook)
	app.SetTransactionContextHook(nrSpanContextHook)
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
	// add a check if OTel is enabled

	ctx, span := h.tracer.Start(ctx, spanName, opts...)

	readWriteSpan, ok := span.(trace.ReadWriteSpan)
	if !ok || !span.IsRecording() {
		return ctx, span
	}
	config := oteltrace.NewSpanStartConfig(opts...)
	hybridNRSpan := &hybridSpan{
		ReadWriteSpan: readWriteSpan,
		spanContext:   span.SpanContext(),
		app:           h.app,
	}
	// check if span is considered a transaction
	isTxn, isWeb := isTransaction(readWriteSpan.SpanKind(), readWriteSpan.Parent(), config.NewRoot(), ctx)
	switch {
	case isTxn:
		hybridNRSpan.startTransaction(readWriteSpan, isWeb)
		if hybridNRSpan.txn == nil {
			return ctx, span
		}
		ctx = newrelic.NewContext(ctx, hybridNRSpan.txn)
	case newrelic.FromContext(ctx) != nil:
		hybridNRSpan.startSegment(readWriteSpan, newrelic.FromContext(ctx))
		if hybridNRSpan.seg == nil {
			return ctx, span
		}
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

func (p *hybridSpan) startTransaction(s trace.ReadWriteSpan, isWeb bool) {
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
	p.txn = txn

	var traceId oteltrace.TraceID
	var spanId oteltrace.SpanID
	traceId, err := oteltrace.TraceIDFromHex(txn.GetLinkingMetadata().TraceID)
	if err != nil {
		traceId = s.SpanContext().TraceID()
	}
	spanId, err = oteltrace.SpanIDFromHex(txn.GetLinkingMetadata().SpanID)
	if err != nil {
		spanId = s.SpanContext().SpanID()
	}
	p.SetSpanContext(s.SpanContext().WithTraceID(traceId).WithSpanID(spanId))
}

func (p *hybridSpan) startSegment(s trace.ReadWriteSpan, parentTxn *newrelic.Transaction) {
	if p == nil {
		return
	}
	seg := parentTxn.StartSegment(s.Name())
	otelSpanID := s.SpanContext().SpanID()
	seg.AddOtelSpanID(otelSpanID.String())
	p.seg = &hybridSegment{
		seg:       seg,
		parentTxn: parentTxn,
	}

	var traceId oteltrace.TraceID
	var spanId oteltrace.SpanID
	traceId, err := oteltrace.TraceIDFromHex(parentTxn.GetLinkingMetadata().TraceID)
	if err != nil {
		traceId = s.SpanContext().TraceID()
	}
	spanId, err = oteltrace.SpanIDFromHex(seg.SpanID())
	if err != nil {
		spanId = s.SpanContext().SpanID()
	}
	p.SetSpanContext(s.SpanContext().WithTraceID(traceId).WithSpanID(spanId))
}

func (p *hybridSpan) SetSpanContext(sc oteltrace.SpanContext) {
	p.spanContext = sc
}

func (p *hybridSpan) SpanContext() oteltrace.SpanContext {
	return p.spanContext
}

func (p *hybridSpan) End(options ...oteltrace.SpanEndOption) {
	if p == nil {
		return
	}
	// OTel treats every End after the first as a no-op, so the New Relic side must too.
	p.endOnce.Do(func() {
		links := p.Links()
		events := p.Events()
		nrErr, hasErr := exceptionFromSpan(p)
		p.ReadWriteSpan.End(options...) // end inner OTel span

		switch {
		case p.txn != nil:
			if hasErr {
				p.txn.NoticeError(nrErr)
			}
			for _, link := range links {
				p.txn.AddLink(link.SpanContext.SpanID().String(), link.SpanContext.TraceID().String(), p.StartTime())
			}
			for j, event := range events {
				if j > 99 {
					p.app.RecordCustomMetric(spanEventEventsDroppedMetricName, 1.0)
					continue
				}
				p.txn.AddSpanEventEvent(event.Name, event.Time, otelEventAttributes(event.Attributes))
			}
			p.txn.End()
		case p.seg != nil:
			p.seg.switchSegmentType(p.Attributes(), p.SpanKind())

			if hasErr && p.seg.parentTxn != nil {
				p.seg.parentTxn.NoticeError(nrErr)
			}
			for _, link := range links {
				p.seg.seg.AddLink(link.SpanContext.SpanID().String(), link.SpanContext.TraceID().String(), p.StartTime())
			}
			for i, event := range events {
				if i > 99 {
					p.app.RecordCustomMetric(spanEventEventsDroppedMetricName, 1.0)
					continue
				}
				p.seg.seg.AddSpanEventEvent(event.Name, event.Time, otelEventAttributes(event.Attributes))
			}
			p.seg.seg.End()
		}
	})
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
func (p *hybridSegment) switchSegmentType(attributes []attribute.KeyValue, spanKind oteltrace.SpanKind) {
	if p == nil {
		return
	}
	segInterface := p.seg

	basicSegment, ok := segInterface.(*newrelic.Segment)
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
				addSegmentAttributes(seg, attributes, OTELToNRDBAttributeMap)
				p.seg = seg
				return
			}
		}
		seg := &newrelic.ExternalSegment{
			StartTime: basicSegment.StartTime,
		}
		addSegmentAttributes(seg, attributes, OTELToNRHTTPAttributeMap)
		p.seg = seg
	case oteltrace.SpanKindProducer:
		seg := &newrelic.MessageProducerSegment{
			StartTime: basicSegment.StartTime,
		}
		addSegmentAttributes(seg, attributes, OTELToNRMessagingProducerAttributeMap)
		p.seg = seg
	case oteltrace.SpanKindConsumer:
		addSegmentAttributes(basicSegment, attributes, OTELToNRMessagingConsumerAttributeMap)
	default:
		addSegmentAttributes(basicSegment, attributes, nil)
	}
}

// This adds attributes based on the type of segment that is found.  These attributes exist within a
// constant map and will need to be updated when the spec outlines semconv changes.
func addSegmentAttributes(seg nrSegment, attributes []attribute.KeyValue, attrMap map[string]string) {

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
