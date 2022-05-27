package newrelic

import (
	"fmt"
	"testing"
	"time"
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
	data := LogData{
		Timestamp: 123456,
		Severity:  "INFO",
		Message:   "test message",
	}

	data.toLogEvent()
}

func recordLogBenchmarkHelper(b *testing.B, data *LogData, h *harvest) {
	event, _ := data.toLogEvent()
	event.MergeIntoHarvest(h)
}

func BenchmarkRecordLog(b *testing.B) {
	harvest := newHarvest(time.Now(), testHarvestCfgr)
	data := LogData{
		Timestamp: 123456,
		Severity:  "INFO",
		Message:   "test message",
	}

	b.ReportAllocs()
	b.ResetTimer()

	recordLogBenchmarkHelper(b, &data, harvest)
}

func BenchmarkRecordLog100(b *testing.B) {
	harvest := newHarvest(time.Now(), testHarvestCfgr)

	logs := make([]*LogData, 100)
	for i := 0; i < 100; i++ {
		logs[i] = &LogData{
			Timestamp: 123456,
			Severity:  "INFO",
			Message:   "test message " + fmt.Sprint(i),
		}
	}

	b.ReportAllocs()
	b.ResetTimer()

	for _, log := range logs {
		recordLogBenchmarkHelper(b, log, harvest)
	}
}
