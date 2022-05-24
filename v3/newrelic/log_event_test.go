package newrelic

import (
	"testing"
)

func TestWriteJSON(t *testing.T) {
	event := logEvent{
		severity:  "INFO",
		message:   "test message",
		timestamp: 123456,
	}
	actual, err := event.MarshalJSON()
	if err != nil {
		t.Error(err)
	}

	expect := `{"level":"INFO","message":"test message","timestamp":123456}`
	actualString := string(actual)
	if expect != actualString {
		t.Errorf("Log json did not build correctly: expecting %s, got %s", expect, actualString)
	}
}

func TestWriteJSONWithTrace(t *testing.T) {
	event := logEvent{
		severity:  "INFO",
		message:   "test message",
		timestamp: 123456,
		traceID:   "123Ad234",
		spanID:    "adf3441",
	}
	actual, err := event.MarshalJSON()
	if err != nil {
		t.Error(err)
	}

	expect := `{"level":"INFO","message":"test message","span.id":"adf3441","trace.id":"123Ad234","timestamp":123456}`
	actualString := string(actual)
	if expect != actualString {
		t.Errorf("Log json did not build correctly: expecting %s, got %s", expect, actualString)
	}
}

func BenchmarkToLogEvent(b *testing.B) {
	b.ReportAllocs()
	data := LogData{
		Severity:  "INFO",
		Message:   "test message",
		Timestamp: 123456,
		TraceID:   "123Ad234",
		SpanID:    "adf3441",
	}
	data.ToLogEvent()
}
