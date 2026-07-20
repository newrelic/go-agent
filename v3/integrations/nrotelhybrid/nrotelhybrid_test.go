package nrotelhybrid

import (
	"testing"

	oteltrace "go.opentelemetry.io/otel/trace"
)

func Test_shouldStartTransaction(t *testing.T) {
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
			got := isTransaction(tt.parent)
			if got != tt.want {
				t.Errorf("isTransaction() = %v, want %v", got, tt.want)
			}
		})
	}
}
