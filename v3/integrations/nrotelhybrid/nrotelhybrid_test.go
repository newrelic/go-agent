package nrotelhybrid

import (
	"context"
	"testing"

	"github.com/newrelic/go-agent/v3/newrelic/integrationsupport"
	"go.opentelemetry.io/otel"
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
				txnChecker: func(txnMap map[oteltrace.TraceID]txnMapEntry, traceID oteltrace.TraceID, spanID oteltrace.SpanID) bool {
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

	noEntriesMap := map[oteltrace.TraceID]txnMapEntry{}

	singleEntryMap := map[oteltrace.TraceID]txnMapEntry{
		validTraceID: {txn: nil, spanID: validSpanID},
	}

	multiEntryMap := map[oteltrace.TraceID]txnMapEntry{
		validTraceID: {txn: nil, spanID: validSpanID},
		otherTraceID: {txn: nil, spanID: otherSpanID},
	}

	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		traceID oteltrace.TraceID
		spanID  oteltrace.SpanID
		txnMap  map[oteltrace.TraceID]txnMapEntry
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
