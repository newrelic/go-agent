package newrelic

import (
	"testing"
	"time"
)

func BenchmarkWrite(b *testing.B) {
	app, err := NewApplication(
		ConfigAppLogForwardingEnabled(true),
	)
	if err != nil {
		b.Error(err)
	}

	json := writeLog("debug", "test message", "", "", time.Now().UnixMilli())
	writer, err := NewLogWriter(app, nil)
	if err != nil {
		b.Error(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	writer.Write(json)
}
