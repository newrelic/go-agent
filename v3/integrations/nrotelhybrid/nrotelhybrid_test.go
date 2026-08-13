package nrotelhybrid

import (
	"context"
	"maps"
	"reflect"
	"testing"

	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/newrelic/go-agent/v3/newrelic/integrationsupport"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func Test_isTransaction(t *testing.T) {
	validTraceID := [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}
	validSpanID := [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		kind     oteltrace.SpanKind
		parent   oteltrace.SpanContext
		txnCheck bool
		wantTxn  bool
		wantWeb  bool
	}{
		{
			name: "Remote parent exists and is valid. Kind is unspecified.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				TraceID: validTraceID,
				SpanID:  validSpanID,
				Remote:  true,
			}),
			wantTxn: true,
			wantWeb: false,
		},
		{
			name: "Remote parent exists but is not valid. Kind is unspecified.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: true,
			}),
			wantTxn: true,
			wantWeb: false,
		},
		{
			name: "Remote parent exists. Kind is server.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				TraceID: validTraceID,
				SpanID:  validSpanID,
				Remote:  true,
			}),
			kind:    oteltrace.SpanKindServer,
			wantTxn: true,
			wantWeb: true,
		},
		{
			name: "Remote parent exists. Kind is client.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				TraceID: validTraceID,
				SpanID:  validSpanID,
				Remote:  true,
			}),
			kind:    oteltrace.SpanKindClient,
			wantTxn: true,
			wantWeb: true,
		},
		{
			name: "Remote parent exists. Kind is consumer.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				TraceID: validTraceID,
				SpanID:  validSpanID,
				Remote:  true,
			}),
			kind:    oteltrace.SpanKindConsumer,
			wantTxn: true,
			wantWeb: false,
		},
		{
			name: "Remote parent exists. Kind is producer.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				TraceID: validTraceID,
				SpanID:  validSpanID,
				Remote:  true,
			}),
			kind:    oteltrace.SpanKindProducer,
			wantTxn: true,
			wantWeb: false,
		},
		{
			name:    "No parent. Kind is unspecified.",
			kind:    oteltrace.SpanKindUnspecified,
			wantTxn: false,
			wantWeb: false,
		},
		{
			name: "Parent is not remote. Kind is unspecified.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: false,
			}),
			kind:    oteltrace.SpanKindUnspecified,
			wantTxn: false,
			wantWeb: false,
		},
		{
			name:    "No parent. Kind is internal.",
			kind:    oteltrace.SpanKindInternal,
			wantTxn: false,
			wantWeb: false,
		},
		{
			name: "Parent is not remote. Kind is internal.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: false,
			}),
			kind:    oteltrace.SpanKindInternal,
			wantTxn: false,
			wantWeb: false,
		},
		{
			name:     "No parent. Kind is server. txnChecker returns true.",
			kind:     oteltrace.SpanKindServer,
			txnCheck: true,
			wantTxn:  false,
			wantWeb:  true,
		},
		{
			name:     "No parent. Kind is server. txnChecker returns false.",
			kind:     oteltrace.SpanKindServer,
			txnCheck: false,
			wantTxn:  true,
			wantWeb:  true,
		},
		{
			name: "Parent is not remote. Kind is server. txnChecker returns true.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: false,
			}),
			kind:     oteltrace.SpanKindServer,
			txnCheck: true,
			wantTxn:  false,
			wantWeb:  true,
		},
		{
			name: "Parent is not remote. Kind is server. txnChecker returns false.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: false,
			}),
			kind:     oteltrace.SpanKindServer,
			txnCheck: false,
			wantTxn:  true,
			wantWeb:  true,
		},
		{
			name:    "No parent. Kind is client.",
			kind:    oteltrace.SpanKindClient,
			wantTxn: false,
			wantWeb: true,
		},
		{
			name: "Parent is not remote. Kind is client.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: false,
			}),
			kind:    oteltrace.SpanKindClient,
			wantTxn: false,
			wantWeb: true,
		},
		{
			name:    "No parent. Kind is producer.",
			kind:    oteltrace.SpanKindProducer,
			wantTxn: false,
			wantWeb: false,
		},
		{
			name: "Parent is not remote. Kind is producer.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: false,
			}),
			kind:    oteltrace.SpanKindProducer,
			wantTxn: false,
			wantWeb: false,
		},
		{
			name:     "No parent. Kind is consumer. txnCheck returns true.",
			kind:     oteltrace.SpanKindConsumer,
			txnCheck: true,
			wantTxn:  false,
			wantWeb:  false,
		},
		{
			name:     "No parent. Kind is consumer. txnCheck returns false.",
			kind:     oteltrace.SpanKindConsumer,
			txnCheck: false,
			wantTxn:  true,
			wantWeb:  false,
		},
		{
			name: "Parent is not remote. Kind is consumer. txnCheck returns true.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: false,
			}),
			kind:     oteltrace.SpanKindConsumer,
			txnCheck: true,
			wantTxn:  false,
			wantWeb:  false,
		},
		{
			name: "Parent is not remote. Kind is consumer. txnCheck returns false",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: false,
			}),
			kind:     oteltrace.SpanKindConsumer,
			txnCheck: false,
			wantTxn:  true,
			wantWeb:  false,
		},
		{
			name:    "No parent. Kind is not listed.",
			kind:    7,
			wantTxn: false,
			wantWeb: false,
		},
		{
			name: "Parent is not remote. Kind is not listed.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: false,
			}),
			kind:    8,
			wantTxn: false,
			wantWeb: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := nrotelhybridProcessor{
				txnChecker: func(txnMap map[oteltrace.TraceID][]txnMapEntry, traceID oteltrace.TraceID, spanID oteltrace.SpanID) bool {
					return tt.txnCheck
				},
			}
			gotTxn, gotWeb := p.isTransaction(tt.kind, oteltrace.SpanContext{}, tt.parent)
			if gotTxn != tt.wantTxn || gotWeb != tt.wantWeb {
				t.Errorf("isTransaction() = (%v, %v), want (%v, %v)", gotTxn, gotWeb, tt.wantTxn, tt.wantWeb)
			}
		})
	}
}

func Test_nrotelhybridProcessor_isWithinTransaction(t *testing.T) {
	validTraceID := [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}
	validSpanID := [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	otherTraceID := [16]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20}
	otherSpanID := [8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18}
	missingTraceID := [16]byte{0x12, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20}

	noEntriesMap := map[oteltrace.TraceID][]txnMapEntry{}

	singleEntryMap := map[oteltrace.TraceID][]txnMapEntry{
		validTraceID: {{txn: nil, spanID: validSpanID}},
	}
	singleEntryMultipleTxnMapEntriesMap := map[oteltrace.TraceID][]txnMapEntry{
		validTraceID: {
			{txn: nil, spanID: validSpanID},
			{txn: nil, spanID: otherSpanID},
		},
	}

	multiEntryMap := map[oteltrace.TraceID][]txnMapEntry{
		validTraceID: {{txn: nil, spanID: validSpanID}},
		otherTraceID: {{txn: nil, spanID: otherSpanID}},
	}

	multiEntryMultipleTxnMapEntriesMap := map[oteltrace.TraceID][]txnMapEntry{
		validTraceID: {
			{txn: nil, spanID: validSpanID},
			{txn: nil, spanID: otherSpanID},
		},
		otherTraceID: {
			{txn: nil, spanID: otherSpanID},
			{txn: nil, spanID: validSpanID},
		},
	}

	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		traceID oteltrace.TraceID
		spanID  oteltrace.SpanID
		txnMap  map[oteltrace.TraceID][]txnMapEntry
		want    bool
	}{
		{
			name:    "TraceID is not within empty transaction map.",
			traceID: validTraceID,
			spanID:  validSpanID,
			txnMap:  noEntriesMap,
			want:    false,
		},
		{
			name:    "TraceID is not within transaction map.",
			traceID: otherTraceID,
			spanID:  otherSpanID,
			txnMap:  singleEntryMap,
			want:    false,
		},
		{
			name:    "TraceID is within transaction map and span ID is the same.", // should return false since it is the same transaction as in the map
			traceID: validTraceID,
			spanID:  validSpanID,
			txnMap:  singleEntryMap,
			want:    false,
		},
		{
			name:    "TraceID is within transaction map and span ID not the same.",
			traceID: validTraceID,
			spanID:  otherSpanID,
			txnMap:  singleEntryMap,
			want:    true,
		},
		{
			name:    "TraceID is within transaction map with multiple entries and span ID is the same.",
			traceID: otherTraceID,
			spanID:  otherSpanID,
			txnMap:  multiEntryMap,
			want:    false,
		},
		{
			name:    "TraceID is within transaction map with multiple entries and span ID not the same.",
			traceID: otherTraceID,
			spanID:  validSpanID,
			txnMap:  multiEntryMap,
			want:    true,
		},
		{
			name:    "TraceID is not within with multiple entries transaction map.",
			traceID: missingTraceID,
			spanID:  validSpanID,
			txnMap:  multiEntryMap,
			want:    false,
		},
		{
			name:    "TraceID is within transaction map and is within an existing transaction.",
			traceID: validTraceID,
			spanID:  validSpanID,
			txnMap:  singleEntryMultipleTxnMapEntriesMap,
			want:    true,
		},
		{
			name:    "TraceID is within transaction map and is not within an existing transaction.",
			traceID: validTraceID,
			spanID:  otherSpanID,
			txnMap:  singleEntryMultipleTxnMapEntriesMap,
			want:    false,
		},
		{
			name:    "TraceID is within mutli entry transaction map is within an existing transaction.",
			traceID: validTraceID,
			spanID:  validSpanID,
			txnMap:  multiEntryMultipleTxnMapEntriesMap,
			want:    true,
		},
		{
			name:    "TraceID is within mutli entry transaction map is within an existing transaction.",
			traceID: validTraceID,
			spanID:  otherSpanID,
			txnMap:  multiEntryMultipleTxnMapEntriesMap,
			want:    false,
		},
		{
			name:    "TraceID is within mutli entry transaction map (different entry) is within an existing transaction.",
			traceID: otherTraceID,
			spanID:  otherSpanID,
			txnMap:  multiEntryMultipleTxnMapEntriesMap,
			want:    true,
		},
		{
			name:    "TraceID is within mutli entry transaction map (different entry) is within an existing transaction.",
			traceID: otherTraceID,
			spanID:  validSpanID,
			txnMap:  multiEntryMultipleTxnMapEntriesMap,
			want:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got := isWithinTransaction(tt.txnMap, tt.traceID, tt.spanID)
			if got != tt.want {
				t.Errorf("isWithinTransaction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_nrotelhybridProcessor(t *testing.T) {
	app := integrationsupport.NewTestApp(
		integrationsupport.SampleEverythingReplyFn,
		integrationsupport.ConfigFullTraces,
	)

	tests := []struct {
		name                string
		spanName            string
		fn                  func(ctx context.Context, spanName string, tp *trace.TracerProvider) oteltrace.Span
		expectTxnMapLen     int
		expectSegmentMapLen int
	}{
		{
			name:     "No txn or segment. Defaults to SpanKind.INTERNAL",
			spanName: "transaction-a",
			fn: func(ctx context.Context, spanName string, tp *trace.TracerProvider) oteltrace.Span {
				tracer := otel.Tracer("test")
				_, span := tracer.Start(ctx, spanName)
				return span
			},
			expectTxnMapLen:     0,
			expectSegmentMapLen: 0,
		},
		{
			name:     "No txn or segment. Set to SpanKind.INTERNAL",
			spanName: "transaction-a",
			fn: func(ctx context.Context, spanName string, tp *trace.TracerProvider) oteltrace.Span {
				tracer := otel.Tracer("test")
				_, span := tracer.Start(ctx, spanName, oteltrace.WithSpanKind(oteltrace.SpanKindInternal))
				return span
			},
			expectTxnMapLen:     0,
			expectSegmentMapLen: 0,
		},
		{
			name:     "Basic txn. Set to SpanKind.SERVER",
			spanName: "transaction-a",
			fn: func(ctx context.Context, spanName string, tp *trace.TracerProvider) oteltrace.Span {
				tracer := otel.Tracer("test")
				_, span := tracer.Start(ctx, spanName, oteltrace.WithSpanKind(oteltrace.SpanKindServer))
				return span
			},
			expectTxnMapLen:     1,
			expectSegmentMapLen: 0,
		},
		{
			name:     "No txn or segment. Set to SpanKind.CLIENT",
			spanName: "transaction-a",
			fn: func(ctx context.Context, spanName string, tp *trace.TracerProvider) oteltrace.Span {
				tracer := otel.Tracer("test")
				_, span := tracer.Start(ctx, spanName, oteltrace.WithSpanKind(oteltrace.SpanKindClient))
				return span
			},
			expectTxnMapLen:     0,
			expectSegmentMapLen: 0,
		},
		{
			name:     "Basic txn. Set to SpanKind.CONSUMER",
			spanName: "transaction-a",
			fn: func(ctx context.Context, spanName string, tp *trace.TracerProvider) oteltrace.Span {
				tracer := otel.Tracer("test")
				_, span := tracer.Start(ctx, spanName, oteltrace.WithSpanKind(oteltrace.SpanKindConsumer))
				return span
			},
			expectTxnMapLen:     1,
			expectSegmentMapLen: 0,
		},
		{
			name:     "No txn or segment. Set to SpanKind.UNSPECIFIED",
			spanName: "transaction-a",
			fn: func(ctx context.Context, spanName string, tp *trace.TracerProvider) oteltrace.Span {
				tracer := otel.Tracer("test")
				_, span := tracer.Start(ctx, spanName, oteltrace.WithSpanKind(oteltrace.SpanKindUnspecified))
				return span
			},
			expectTxnMapLen:     0,
			expectSegmentMapLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewHybridProcessor(app.Application)
			tp := trace.NewTracerProvider(trace.WithSpanProcessor(processor))
			shutdown := func(ctx context.Context) error {
				err := tp.Shutdown(ctx)
				return err
			}
			defer shutdown(context.Background())
			otel.SetTracerProvider(tp)

			span := tt.fn(context.Background(), tt.spanName, tp)

			if len(processor.txnMap) != tt.expectTxnMapLen {
				t.Errorf("Unexpected length of txnMap. Expected len =  %d, got len = %d", tt.expectTxnMapLen, len(processor.txnMap))
			}
			if len(processor.segmentMap) != tt.expectSegmentMapLen {
				t.Errorf("Unexpected length of segmentMap. Expected len =  %d, got len = %d", tt.expectSegmentMapLen, len(processor.segmentMap))
			}

			span.End()

		})
	}
}

func Test_extractAttributeValue(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		val  attribute.Value
		want any
	}{
		{
			name: "Bool",
			val:  attribute.BoolValue(true),
			want: true,
		},
		{
			name: "Int64",
			val:  attribute.Int64Value(42),
			want: int64(42),
		},
		{
			name: "Float64",
			val:  attribute.Float64Value(3.14),
			want: 3.14,
		},
		{
			name: "String",
			val:  attribute.StringValue("hello"),
			want: "hello",
		},
		{
			name: "BoolSlice",
			val:  attribute.BoolSliceValue([]bool{true, false}),
			want: []bool{true, false},
		},
		{
			name: "Int64Slice",
			val:  attribute.Int64SliceValue([]int64{1, 2, 3}),
			want: []int64{1, 2, 3},
		},
		{
			name: "Float64Slice",
			val:  attribute.Float64SliceValue([]float64{1.1, 2.2}),
			want: []float64{1.1, 2.2},
		},
		{
			name: "StringSlice",
			val:  attribute.StringSliceValue([]string{"a", "b"}),
			want: []string{"a", "b"},
		},
		{
			name: "ByteSlice",
			val:  attribute.ByteSliceValue([]byte{0x01, 0x02}),
			want: []byte{0x01, 0x02},
		},
		{
			name: "Slice",
			val:  attribute.SliceValue(attribute.StringValue("a"), attribute.IntValue(1)),
			want: []attribute.Value{attribute.StringValue("a"), attribute.IntValue(1)},
		},
		{
			name: "Empty/invalid value",
			val:  attribute.Value{},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractAttributeValue(tt.val)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractAttributeValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_nrotelhybridProcessor_switchSegmentType(t *testing.T) {
	validSpanID := [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	otherSpanID := [8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18}

	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		initialSegmentMap map[oteltrace.SpanID]nrSegment
		spanID            oteltrace.SpanID
		attributes        []attribute.KeyValue
		spanKind          oteltrace.SpanKind
		want              nrSegment
	}{
		{
			name:              "Segment does not exist in empty segment map.",
			initialSegmentMap: map[oteltrace.SpanID]nrSegment{},
			spanID:            validSpanID,
			spanKind:          oteltrace.SpanKindClient,
			want:              nil,
		},
		{
			name: "Segment does not exist in segment map",
			initialSegmentMap: map[oteltrace.SpanID]nrSegment{
				otherSpanID: &newrelic.Segment{
					StartTime: newrelic.SegmentStartTime{},
					Name:      "Basic Segment",
				},
			},
			spanID:   validSpanID,
			spanKind: oteltrace.SpanKindConsumer,
			want:     nil,
		},
		{
			name:   "Segment is not a basic segment upon type cast. Should stay the same type.",
			spanID: validSpanID,
			initialSegmentMap: map[oteltrace.SpanID]nrSegment{
				validSpanID: &newrelic.DatastoreSegment{
					StartTime: newrelic.SegmentStartTime{},
				},
			},
			spanKind: oteltrace.SpanKindInternal,
			want:     &newrelic.DatastoreSegment{},
		},
		{
			name:   "Segment is not a basic segment (different type) upon type cast. Should stay the same type.",
			spanID: otherSpanID,
			initialSegmentMap: map[oteltrace.SpanID]nrSegment{
				otherSpanID: &newrelic.ExternalSegment{
					StartTime: newrelic.SegmentStartTime{},
				},
			},
			spanKind: oteltrace.SpanKindInternal,
			want:     &newrelic.ExternalSegment{},
		},
		{
			name:   "Segment is a basic segment upon type cast. SpanKind is INTERNAL. Should stay the same type.",
			spanID: otherSpanID,
			initialSegmentMap: map[oteltrace.SpanID]nrSegment{
				otherSpanID: &newrelic.Segment{
					StartTime: newrelic.SegmentStartTime{},
					Name:      "Basic Segment",
				},
			},
			spanKind: oteltrace.SpanKindInternal,
			want:     &newrelic.Segment{},
		},
		{
			name:   "Segment is a basic segment upon type cast. SpanKind is UNSPECIFIED. Should stay the same type.",
			spanID: validSpanID,
			initialSegmentMap: map[oteltrace.SpanID]nrSegment{
				validSpanID: &newrelic.Segment{
					StartTime: newrelic.SegmentStartTime{},
					Name:      "Basic Segment",
				},
			},
			spanKind: oteltrace.SpanKindUnspecified,
			want:     &newrelic.Segment{},
		},
		{
			name:   "Segment is a basic segment upon type cast. SpanKind is CLIENT. Should convert to External Segment.",
			spanID: validSpanID,
			initialSegmentMap: map[oteltrace.SpanID]nrSegment{
				validSpanID: &newrelic.Segment{
					StartTime: newrelic.SegmentStartTime{},
					Name:      "Basic Segment",
				},
			},
			attributes: []attribute.KeyValue{},
			spanKind:   oteltrace.SpanKindClient,
			want:       &newrelic.ExternalSegment{},
		},
		{
			name:   "Segment is a basic segment upon type cast. SpanKind is CLIENT. Should convert to External Segment. Attributes are present.",
			spanID: otherSpanID,
			initialSegmentMap: map[oteltrace.SpanID]nrSegment{
				otherSpanID: &newrelic.Segment{
					StartTime: newrelic.SegmentStartTime{},
					Name:      "Basic Segment",
				},
			},
			attributes: []attribute.KeyValue{
				{Key: attribute.Key(AttrURLFull), Value: attribute.StringValue("testURL")},
			},
			spanKind: oteltrace.SpanKindClient,
			want:     &newrelic.ExternalSegment{},
		},
		{
			name:   "Segment is a basic segment upon type cast. SpanKind is CLIENT. Should convert to Datastore Segment. Uses AttrDBSystem constant.",
			spanID: validSpanID,
			initialSegmentMap: map[oteltrace.SpanID]nrSegment{
				validSpanID: &newrelic.Segment{
					StartTime: newrelic.SegmentStartTime{},
					Name:      "Basic Segment",
				},
			},
			attributes: []attribute.KeyValue{
				{Key: attribute.Key(AttrDBSystem), Value: attribute.StringValue("test-system")},
			},
			spanKind: oteltrace.SpanKindClient,
			want:     &newrelic.DatastoreSegment{},
		},
		{
			name:   "Segment is a basic segment upon type cast. SpanKind is CLIENT.Should convert to Datastore Segment. Uses AttrDBSystemName constant.",
			spanID: otherSpanID,
			initialSegmentMap: map[oteltrace.SpanID]nrSegment{
				otherSpanID: &newrelic.Segment{
					StartTime: newrelic.SegmentStartTime{},
					Name:      "Basic Segment",
				},
			},
			attributes: []attribute.KeyValue{
				{Key: attribute.Key(AttrDBSystemName), Value: attribute.StringValue("test-system")},
			},
			spanKind: oteltrace.SpanKindClient,
			want:     &newrelic.DatastoreSegment{},
		},
		{
			name:   "Segment is a basic segment upon type cast. SpanKind is CLIENT.Should convert to Datastore Segment. Contains both DB constants.",
			spanID: otherSpanID,
			initialSegmentMap: map[oteltrace.SpanID]nrSegment{
				otherSpanID: &newrelic.Segment{
					StartTime: newrelic.SegmentStartTime{},
					Name:      "Basic Segment",
				},
			},
			attributes: []attribute.KeyValue{
				{Key: attribute.Key(AttrDBSystemName), Value: attribute.StringValue("test-system")},
				{Key: attribute.Key(AttrDBSystem), Value: attribute.StringValue("test-system")},
			},
			spanKind: oteltrace.SpanKindClient,
			want:     &newrelic.DatastoreSegment{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewHybridProcessor(&newrelic.Application{})
			maps.Copy(p.segmentMap, tt.initialSegmentMap)
			p.switchSegmentType(tt.spanID, tt.attributes, tt.spanKind)

			seg, ok := p.segmentMap[tt.spanID]

			if tt.want == nil {
				if ok {
					t.Errorf("Expected no segment, got segment at spanID = %v", tt.spanID)
				}
				return
			}

			if !ok {
				t.Errorf("Expected segment at spanID = %v, got nothing", tt.spanID)
				return
			}

			if reflect.TypeOf(seg) != reflect.TypeOf(tt.want) {
				t.Errorf("Expected segment type of %v, got %v", reflect.TypeOf(seg), reflect.TypeOf(tt.want))
			}
		})
	}
}

func Test_checkMap(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		key     attribute.Key
		attrMap map[string]string
		want    string
		want2   bool
	}{
		{
			name:    "Nil map",
			key:     attribute.Key("some.key"),
			attrMap: nil,
			want:    "",
			want2:   false,
		},
		{
			name:    "Key present in map",
			key:     attribute.Key("some.key"),
			attrMap: map[string]string{"some.key": "nr.attribute"},
			want:    "nr.attribute",
			want2:   true,
		},
		{
			name:    "Key not present in map",
			key:     attribute.Key("missing.key"),
			attrMap: map[string]string{"some.key": "nr.attribute"},
			want:    "",
			want2:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got2 := checkMap(tt.key, tt.attrMap)
			if got != tt.want {
				t.Errorf("checkMap() = %v, want %v", got, tt.want)
			}
			if got2 != tt.want2 {
				t.Errorf("checkMap() = %v, want %v", got2, tt.want2)
			}
		})
	}
}

type fakeSegment struct {
	attrs map[string]interface{}
}

func (f *fakeSegment) End() {}

func (f *fakeSegment) AddAttribute(key string, val interface{}) {
	if f.attrs == nil {
		f.attrs = map[string]interface{}{}
	}
	f.attrs[key] = val
}

func Test_nrotelhybridProcessor_addSegmentAttributes(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		attributes []attribute.KeyValue
		attrMap    map[string]string
		want       map[string]interface{}
	}{
		{
			name:       "No attributes.",
			attributes: []attribute.KeyValue{},
			attrMap:    map[string]string{},
			want:       map[string]interface{}{},
		},
		{
			name: "Nil attrMap. Keys pass through unmapped.",
			attributes: []attribute.KeyValue{
				attribute.String("otel.key", "value"),
			},
			attrMap: nil,
			want: map[string]interface{}{
				"otel.key": "value",
			},
		},
		{
			name: "Key present in attrMap is renamed.",
			attributes: []attribute.KeyValue{
				attribute.String("otel.key", "value"),
			},
			attrMap: map[string]string{
				"otel.key": "nr.key",
			},
			want: map[string]interface{}{
				"nr.key": "value",
			},
		},
		{
			name: "Mix of mapped and unmapped keys, various value types.",
			attributes: []attribute.KeyValue{
				attribute.String("otel.mapped", "value"),
				attribute.Int64("otel.unmapped", 42),
				attribute.Bool("otel.bool", true),
			},
			attrMap: map[string]string{
				"otel.mapped": "nr.mapped",
			},
			want: map[string]interface{}{
				"nr.mapped":     "value",
				"otel.unmapped": int64(42),
				"otel.bool":     true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewHybridProcessor(&newrelic.Application{})
			seg := &fakeSegment{}
			p.addSegmentAttributes(seg, tt.attributes, tt.attrMap)

			if seg.attrs == nil {
				seg.attrs = map[string]interface{}{}
			}
			if !reflect.DeepEqual(seg.attrs, tt.want) {
				t.Errorf("addSegmentAttributes() attrs = %v, want %v", seg.attrs, tt.want)
			}
		})
	}
}

func Test_addSegmentAttributes_DatastoreSegment(t *testing.T) {
	tests := []struct {
		name           string
		attributes     []attribute.KeyValue
		wantCollection string
		wantOperation  string
		wantQuery      string
	}{
		{
			name: "db.collection.name and db.operation.name set typed fields.",
			attributes: []attribute.KeyValue{
				attribute.String(AttrDBCollectionName, "users"),
				attribute.String(AttrDBOperationName, "SELECT"),
			},
			wantCollection: "users",
			wantOperation:  "SELECT",
		},
		{
			name: "Legacy db.sql.table and db.operation set typed fields.",
			attributes: []attribute.KeyValue{
				attribute.String(AttrDBSQLTable, "orders"),
				attribute.String(AttrDBOperation, "INSERT"),
			},
			wantCollection: "orders",
			wantOperation:  "INSERT",
		},
		{
			name: "db.statement sets ParameterizedQuery.",
			attributes: []attribute.KeyValue{
				attribute.String(AttrDBStatement, "SELECT count(*) FROM pg_catalog.pg_tables"),
			},
			wantQuery: "SELECT count(*) FROM pg_catalog.pg_tables",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewHybridProcessor(&newrelic.Application{})
			seg := &newrelic.DatastoreSegment{}
			p.addSegmentAttributes(seg, tt.attributes, OTELToNRDBAttributeMap)

			if seg.Collection != tt.wantCollection {
				t.Errorf("Collection = %v, want %v", seg.Collection, tt.wantCollection)
			}
			if seg.Operation != tt.wantOperation {
				t.Errorf("Operation = %v, want %v", seg.Operation, tt.wantOperation)
			}
			if seg.ParameterizedQuery != tt.wantQuery {
				t.Errorf("ParameterizedQuery = %v, want %v", seg.ParameterizedQuery, tt.wantQuery)
			}
		})
	}
}

func Test_addSegmentAttributes_ExternalSegment(t *testing.T) {
	tests := []struct {
		name       string
		attributes []attribute.KeyValue
		wantURL    string
	}{
		{
			name: "url.full sets URL.",
			attributes: []attribute.KeyValue{
				attribute.String(AttrURLFull, "http://example.com/path"),
			},
			wantURL: "http://example.com/path",
		},
		{
			name: "Legacy http.url sets URL.",
			attributes: []attribute.KeyValue{
				attribute.String(AttrHTTPURL, "http://legacy.example.com/path"),
			},
			wantURL: "http://legacy.example.com/path",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewHybridProcessor(&newrelic.Application{})
			seg := &newrelic.ExternalSegment{}
			p.addSegmentAttributes(seg, tt.attributes, OTELToNRHTTPAttributeMap)

			if seg.URL != tt.wantURL {
				t.Errorf("URL = %v, want %v", seg.URL, tt.wantURL)
			}
		})
	}
}
