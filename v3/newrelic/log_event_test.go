package newrelic

import (
	"testing"
	"time"
)

func TestCreateLogEvent(t *testing.T) {
	tests := []struct {
		Timestamp  int64
		Severity   string
		Message    string
		SpanID     string
		TraceID    string
		Attributes map[string]interface{}
	}{
		{
			Timestamp: 123456,
			Severity:  "debug",
			Message:   "test",
			SpanID:    "123Ifker1",
			TraceID:   "23000L343",
		},
		{
			Timestamp: 123456,
			Severity:  "debug",
			Message:   "test",
		},
		{
			Timestamp: 123456,
			Severity:  "debug",
			Message:   "test",
		},
		{
			Timestamp: 123456,
			Severity:  "debug",
			Message:   "test",
			SpanID:    "123Ifker1",
			TraceID:   "23000L343",
			Attributes: map[string]interface{}{
				"one": "attributeOne",
				"two": "attributeTwo",
			},
		},
	}

	for _, test := range tests {
		var l []byte
		if len(test.Attributes) > 0 {
			l = writeLogWithAttributes(test.Severity, test.Message, test.SpanID, test.TraceID, int64(test.Timestamp), test.Attributes)
		} else {
			l = writeLog(test.Severity, test.Message, test.SpanID, test.TraceID, int64(test.Timestamp))
		}

		logEvent, err := CreateLogEvent(l)
		if err != nil {
			t.Error(err)
		}

		if logEvent.traceID != test.TraceID {
			t.Errorf("invalid traceID: expect \"%s\", got \"%s\"", test.TraceID, logEvent.traceID)
		}
		if logEvent.severity != test.Severity {
			t.Errorf("invalid severity: expect \"%s\", got \"%s\"", test.Severity, logEvent.severity)
		}
	}
}

func TestLogTooLarge(t *testing.T) {
	l := make([]byte, maxLogBytes+1)
	_, err := CreateLogEvent(l)
	if err == nil {
		t.Error("Failed to catch log too large error")
	}
	if err != errLogTooLarge {
		t.Error(err)
	}
}

func TestLogTooSmall(t *testing.T) {
	l := []byte{}
	_, err := CreateLogEvent(l)
	if err == nil {
		t.Error("Failed to catch log too large error")
	}
	if err != errEmptyLog {
		t.Error(err)
	}
}

func BenchmarkCreateLogEvent(b *testing.B) {
	b.ReportAllocs()
	json := writeLog("debug", "test message", "", "", time.Now().UnixMilli())
	_, err := CreateLogEvent(json)
	if err != nil {
		b.Error(err)
	}
}

func BenchmarkCreateLogEvent100(b *testing.B) {
	json := writeLog("debug", "test message", "", "", time.Now().UnixMilli())
	for i := 0; i < 100; i++ {
		_, err := CreateLogEvent(json)
		if err != nil {
			b.Error(err)
		}
	}
}
