package internal

import "testing"

func TestTraceIDGenerator(t *testing.T) {
	tg := NewTraceIDGenerator(12345)
	traceID := tg.GenerateTraceID()
	if traceID != "1ae969564b34a33ecd1af05fe6923d6d" {
		t.Error(traceID)
	}
	spanID := tg.GenerateSpanID()
	if spanID != "e71870997d38ef60" {
		t.Error(spanID)
	}
}
