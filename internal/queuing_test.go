package internal

import (
	"net/http"
	"testing"
	"time"
)

func TestParseQueueTime(t *testing.T) {
	badInput := []string{
		"",
		"nope",
		"t",
		"0",
		"0.0",
		"9999999999999999999999999999999999999999999999999",
		"-1368811467146000",
		"3000000000",
		"3000000000000",
		"900000000",
		"900000000000",
	}
	for _, s := range badInput {
		if qt := parseQueueTime(s); !qt.IsZero() {
			t.Error(s, qt)
		}
	}

	testcases := []struct {
		input  string
		expect int64
	}{
		// Microseconds
		{"1368811467146000", 1368811467},
		// Milliseconds
		{"1368811467146.000", 1368811467},
		{"1368811467146", 1368811467},
		// Seconds
		{"1368811467.146000", 1368811467},
		{"1368811467.146", 1368811467},
		{"1368811467", 1368811467},
	}
	for _, tc := range testcases {
		qt := parseQueueTime(tc.input)
		if qt.Unix() != tc.expect {
			t.Error(tc.input, tc.expect, qt, qt.UnixNano())
		}
	}
}

func TestNewProxiesXQueueStart(t *testing.T) {
	hdr := make(http.Header)
	hdr.Set("X-Queue-Start", "1465798814")
	pr := NewProxies(hdr, time.Unix(1465798816, 0))
	expectAttributes(t, pr.asAttributes(), map[string]interface{}{
		"queueDuration":                    2,
		"caller.transportDuration.Unknown": 2,
	})
	metrics := newMetricTable(100, time.Now())
	pr.createMetrics(metrics, sampleCaller, true)
	ExpectMetrics(t, metrics, []WantMetric{
		{"WebFrontend/QueueTime", "", true, []float64{1, 2, 2, 2, 2, 4}},
		{"IntermediaryTransportDuration/App/123/456/HTTP/Unknown/all", "", false, []float64{1, 2, 2, 2, 2, 4}},
		{"IntermediaryTransportDuration/App/123/456/HTTP/Unknown/allWeb", "", false, []float64{1, 2, 2, 2, 2, 4}},
	})
}

func TestNewProxiesXRequestStart(t *testing.T) {
	hdr := make(http.Header)
	hdr.Set("X-Request-Start", "1465798814")
	pr := NewProxies(hdr, time.Unix(1465798816, 0))
	expectAttributes(t, pr.asAttributes(), map[string]interface{}{
		"queueDuration":                    2,
		"caller.transportDuration.Unknown": 2,
	})
	metrics := newMetricTable(100, time.Now())
	pr.createMetrics(metrics, sampleCaller, true)
	ExpectMetrics(t, metrics, []WantMetric{
		{"WebFrontend/QueueTime", "", true, []float64{1, 2, 2, 2, 2, 4}},
		{"IntermediaryTransportDuration/App/123/456/HTTP/Unknown/all", "", false, []float64{1, 2, 2, 2, 2, 4}},
		{"IntermediaryTransportDuration/App/123/456/HTTP/Unknown/allWeb", "", false, []float64{1, 2, 2, 2, 2, 4}},
	})
}

func TestNewProxiesXQueueStartTEquals(t *testing.T) {
	hdr := make(http.Header)
	hdr.Set("X-Queue-Start", "t=1465798814")
	pr := NewProxies(hdr, time.Unix(1465798816, 0))
	expectAttributes(t, pr.asAttributes(), map[string]interface{}{
		"queueDuration":                    2,
		"caller.transportDuration.Unknown": 2,
	})
	metrics := newMetricTable(100, time.Now())
	pr.createMetrics(metrics, sampleCaller, true)
	ExpectMetrics(t, metrics, []WantMetric{
		{"WebFrontend/QueueTime", "", true, []float64{1, 2, 2, 2, 2, 4}},
		{"IntermediaryTransportDuration/App/123/456/HTTP/Unknown/all", "", false, []float64{1, 2, 2, 2, 2, 4}},
		{"IntermediaryTransportDuration/App/123/456/HTTP/Unknown/allWeb", "", false, []float64{1, 2, 2, 2, 2, 4}},
	})
}

func TestNewProxiesManyInbound(t *testing.T) {
	hdr := make(http.Header)
	hdr.Set("X-Queue-Start", "1465798814")
	hdr.Set("x-newrelic-timestamp-zap", "1465798813")
	pr := NewProxies(hdr, time.Unix(1465798816, 0))
	expectAttributes(t, pr.asAttributes(), map[string]interface{}{
		"queueDuration":                    3,
		"caller.transportDuration.Unknown": 2,
		"caller.transportDuration.Zap":     3,
	})
	metrics := newMetricTable(100, time.Now())
	pr.createMetrics(metrics, sampleCaller, true)
	ExpectMetrics(t, metrics, []WantMetric{
		{"WebFrontend/QueueTime", "", true, []float64{1, 3, 3, 3, 3, 9}},
		{"IntermediaryTransportDuration/App/123/456/HTTP/Unknown/all", "", false, []float64{1, 2, 2, 2, 2, 4}},
		{"IntermediaryTransportDuration/App/123/456/HTTP/Unknown/allWeb", "", false, []float64{1, 2, 2, 2, 2, 4}},
		{"IntermediaryTransportDuration/App/123/456/HTTP/Zap/all", "", false, []float64{1, 3, 3, 3, 3, 9}},
		{"IntermediaryTransportDuration/App/123/456/HTTP/Zap/allWeb", "", false, []float64{1, 3, 3, 3, 3, 9}},
	})
}

func TestNewProxiesNone(t *testing.T) {
	hdr := make(http.Header)
	pr := NewProxies(hdr, time.Unix(1465798816, 0))
	expectAttributes(t, pr.asAttributes(), map[string]interface{}{})
	metrics := newMetricTable(100, time.Now())
	pr.createMetrics(metrics, sampleCaller, true)
	ExpectMetrics(t, metrics, []WantMetric{})
}

func TestNewProxiesFutureTime(t *testing.T) {
	hdr := make(http.Header)
	hdr.Set("X-Queue-Start", "1465798817")
	pr := NewProxies(hdr, time.Unix(1465798816, 0))
	expectAttributes(t, pr.asAttributes(), map[string]interface{}{
		"queueDuration":                    0,
		"caller.transportDuration.Unknown": 0,
	})
	metrics := newMetricTable(100, time.Now())
	pr.createMetrics(metrics, sampleCaller, true)
	ExpectMetrics(t, metrics, []WantMetric{
		{"WebFrontend/QueueTime", "", true, []float64{1, 0, 0, 0, 0, 0}},
		{"IntermediaryTransportDuration/App/123/456/HTTP/Unknown/all", "", false, []float64{1, 0, 0, 0, 0, 0}},
		{"IntermediaryTransportDuration/App/123/456/HTTP/Unknown/allWeb", "", false, []float64{1, 0, 0, 0, 0, 0}},
	})
}

func TestNewProxiesInvalidTime(t *testing.T) {
	hdr := make(http.Header)
	hdr.Set("X-Queue-Start", "!!!!!!")
	pr := NewProxies(hdr, time.Unix(1465798816, 0))
	expectAttributes(t, pr.asAttributes(), map[string]interface{}{
		"queueDuration":                    0,
		"caller.transportDuration.Unknown": 0,
	})
	metrics := newMetricTable(100, time.Now())
	pr.createMetrics(metrics, sampleCaller, true)
	ExpectMetrics(t, metrics, []WantMetric{
		{"WebFrontend/QueueTime", "", true, []float64{1, 0, 0, 0, 0, 0}},
		{"IntermediaryTransportDuration/App/123/456/HTTP/Unknown/all", "", false, []float64{1, 0, 0, 0, 0, 0}},
		{"IntermediaryTransportDuration/App/123/456/HTTP/Unknown/allWeb", "", false, []float64{1, 0, 0, 0, 0, 0}},
	})
}
