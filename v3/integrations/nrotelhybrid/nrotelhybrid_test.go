package nrotelhybrid

import (
	"maps"
	"testing"

	oteltrace "go.opentelemetry.io/otel/trace"
)

func Test_isTransaction(t *testing.T) {
	validTraceID := [16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10}
	validSpanID := [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		parent oteltrace.SpanContext
		want   bool
	}{
		{
			name: "Remote parent exists and is valid",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				TraceID: validTraceID,
				SpanID:  validSpanID,
				Remote:  true,
			}),
			want: true,
		},
		{
			name: "Local parent exists and is valid",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				TraceID: validTraceID,
				SpanID:  validSpanID,
				Remote:  false,
			}),
			want: false,
		},
		{
			name:   "No parent (root span)",
			parent: oteltrace.SpanContext{},
			want:   true,
		},
		{
			name: "Invalid parent marked remote",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				Remote: true,
			}),
			want: true,
		},
		{
			name: "Partially invalid parent (zero SpanID), not remote",
			parent: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
				TraceID: validTraceID,
				Remote:  false,
			}),
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//p := nrotelhybridProcessor{}
			//got := p.isTransaction(tt.parent)
			// if true {
			// 	t.Errorf("isTransaction() = %v, want %v", got, tt.want)
			// }
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
			var p nrotelhybridProcessor
			p.txnMap = make(map[oteltrace.TraceID]txnMapEntry)
			maps.Copy(p.txnMap, tt.txnMap)

			got := p.isWithinTransaction(tt.traceID, tt.spanID)
			if got != tt.want {
				t.Errorf("isWithinTransaction() = %v, want %v", got, tt.want)
			}
		})
	}
}
