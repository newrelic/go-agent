package nrotelhybrid

import (
	"testing"

	"go.opentelemetry.io/otel/trace"
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
		want     bool
	}{
		{
			name: "Remote parent exists and is valid.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				TraceID: validTraceID,
				SpanID:  validSpanID,
				Remote:  true,
			}),
			want: true,
		},
		{
			name: "Remote parent exists but is not valid.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: true,
			}),
			want: true,
		},
		{
			name: "No parent. Kind is unspecified.",
			kind: oteltrace.SpanKindUnspecified,
			want: false,
		},
		{
			name: "Parent is not remote. Kind is unspecified.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: false,
			}),
			kind: oteltrace.SpanKindUnspecified,
			want: false,
		},
		{
			name: "No parent. Kind is internal.",
			kind: oteltrace.SpanKindInternal,
			want: false,
		},
		{
			name: "Parent is not remote. Kind is internal.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: false,
			}),
			kind: oteltrace.SpanKindInternal,
			want: false,
		},
		{
			name:     "No parent. Kind is server. txnChecker returns true.",
			kind:     oteltrace.SpanKindServer,
			txnCheck: true,
			want:     false,
		},
		{
			name:     "No parent. Kind is server. txnChecker returns false.",
			kind:     oteltrace.SpanKindServer,
			txnCheck: false,
			want:     true,
		},
		{
			name: "Parent is not remote. Kind is server. txnChecker returns true.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: false,
			}),
			kind:     oteltrace.SpanKindServer,
			txnCheck: true,
			want:     false,
		},
		{
			name: "Parent is not remote. Kind is server. txnChecker returns false.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: false,
			}),
			kind:     oteltrace.SpanKindServer,
			txnCheck: false,
			want:     true,
		},
		{
			name: "No parent. Kind is client.",
			kind: oteltrace.SpanKindClient,
			want: false,
		},
		{
			name: "Parent is not remote. Kind is client.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: false,
			}),
			kind: oteltrace.SpanKindClient,
			want: false,
		},
		{
			name: "No parent. Kind is producer.",
			kind: oteltrace.SpanKindProducer,
			want: false,
		},
		{
			name: "Parent is not remote. Kind is producer.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: false,
			}),
			kind: oteltrace.SpanKindProducer,
			want: false,
		},
		{
			name:     "No parent. Kind is consumer. txnCheck returns true.",
			kind:     oteltrace.SpanKindConsumer,
			txnCheck: true,
			want:     false,
		},
		{
			name:     "No parent. Kind is consumer. txnCheck returns false.",
			kind:     oteltrace.SpanKindConsumer,
			txnCheck: false,
			want:     true,
		},
		{
			name: "Parent is not remote. Kind is consumer. txnCheck returns true.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: false,
			}),
			kind:     oteltrace.SpanKindConsumer,
			txnCheck: true,
			want:     false,
		},
		{
			name: "Parent is not remote. Kind is consumer. txnCheck returns false",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: false,
			}),
			kind:     oteltrace.SpanKindConsumer,
			txnCheck: false,
			want:     true,
		},
		{
			name: "No parent. Kind is not listed.",
			kind: 7,
			want: false,
		},
		{
			name: "Parent is not remote. Kind is not listed.",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: false,
			}),
			kind: 8,
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := nrotelhybridProcessor{
				txnChecker: func(txnMap map[oteltrace.TraceID]txnMapEntry, traceID oteltrace.TraceID, spanID oteltrace.SpanID) bool {
					return tt.txnCheck
				},
			}
			got := p.isTransaction(tt.kind, trace.SpanContext{}, tt.parent)
			if got != tt.want {
				t.Errorf("isTransaction() = %v, want %v", got, tt.want)
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
