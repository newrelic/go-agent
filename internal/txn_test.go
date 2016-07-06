package internal

import (
	"net/http"
	"testing"
	"time"

	"github.com/newrelic/go-agent/api"
)

func TestResponseCodeIsError(t *testing.T) {
	cfg := api.NewConfig("my app", "0123456789012345678901234567890123456789")

	if is := responseCodeIsError(&cfg, 200); is {
		t.Error(is)
	}
	if is := responseCodeIsError(&cfg, 400); !is {
		t.Error(is)
	}
	if is := responseCodeIsError(&cfg, 404); is {
		t.Error(is)
	}
	if is := responseCodeIsError(&cfg, 503); !is {
		t.Error(is)
	}
}

func TestHostFromRequestResponse(t *testing.T) {
	req, err := http.NewRequest("GET", "http://example.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	host := hostFromRequestResponse(req, &http.Response{Request: req})
	if host != "example.com" {
		t.Error("normal usage", host)
	}
	host = hostFromRequestResponse(nil, &http.Response{Request: req})
	if host != "example.com" {
		t.Error("missing request", host)
	}
	host = hostFromRequestResponse(req, nil)
	if host != "example.com" {
		t.Error("missing response", host)
	}
	host = hostFromRequestResponse(nil, nil)
	if host != "" {
		t.Error("missing request and response", host)
	}
	req.URL = nil
	host = hostFromRequestResponse(req, nil)
	if host != "" {
		t.Error("missing URL", host)
	}
}

func TestCreateTxnMetrics(t *testing.T) {
	webName := "WebTransaction/zip/zap"
	backgroundName := "OtherTransaction/zip/zap"
	args := createTxnMetricsArgs{
		duration:       123 * time.Second,
		exclusive:      109 * time.Second,
		apdexThreshold: 2 * time.Second,
	}

	args.name = webName
	args.isWeb = true
	args.errorsSeen = 1
	args.zone = apdexTolerating
	metrics := newMetricTable(100, time.Now())
	createTxnMetrics(args, metrics)
	expectMetrics(t, metrics, []WantMetric{
		{webName, "", true, []float64{1, 123, 109, 123, 123, 123 * 123}},
		{webRollup, "", true, []float64{1, 123, 109, 123, 123, 123 * 123}},
		{dispatcherMetric, "", true, []float64{1, 123, 0, 123, 123, 123 * 123}},
		{errorsAll, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{errorsWeb, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{"Errors/" + webName, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{apdexRollup, "", true, []float64{0, 1, 0, 2, 2, 0}},
		{"Apdex/zip/zap", "", false, []float64{0, 1, 0, 2, 2, 0}},
	})

	args.name = webName
	args.isWeb = true
	args.errorsSeen = 0
	args.zone = apdexTolerating
	metrics = newMetricTable(100, time.Now())
	createTxnMetrics(args, metrics)
	expectMetrics(t, metrics, []WantMetric{
		{webName, "", true, []float64{1, 123, 109, 123, 123, 123 * 123}},
		{webRollup, "", true, []float64{1, 123, 109, 123, 123, 123 * 123}},
		{dispatcherMetric, "", true, []float64{1, 123, 0, 123, 123, 123 * 123}},
		{apdexRollup, "", true, []float64{0, 1, 0, 2, 2, 0}},
		{"Apdex/zip/zap", "", false, []float64{0, 1, 0, 2, 2, 0}},
	})

	args.name = backgroundName
	args.isWeb = false
	args.errorsSeen = 1
	args.zone = apdexNone
	metrics = newMetricTable(100, time.Now())
	createTxnMetrics(args, metrics)
	expectMetrics(t, metrics, []WantMetric{
		{backgroundName, "", true, []float64{1, 123, 109, 123, 123, 123 * 123}},
		{backgroundRollup, "", true, []float64{1, 123, 109, 123, 123, 123 * 123}},
		{errorsAll, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{errorsBackground, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{"Errors/" + backgroundName, "", true, []float64{1, 0, 0, 0, 0, 0}},
	})

	args.name = backgroundName
	args.isWeb = false
	args.errorsSeen = 0
	args.zone = apdexNone
	metrics = newMetricTable(100, time.Now())
	createTxnMetrics(args, metrics)
	expectMetrics(t, metrics, []WantMetric{
		{backgroundName, "", true, []float64{1, 123, 109, 123, 123, 123 * 123}},
		{backgroundRollup, "", true, []float64{1, 123, 109, 123, 123, 123 * 123}},
	})

}
