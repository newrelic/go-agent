package internal

import (
	"go.opentelemetry.io/otel/api/trace/testtrace"
)

// TraceExpecter implements internal.Expect for use in testing.
type TraceExpecter struct {
	*testtrace.Tracer
}

// ExpectTxnEvents TODO
func (te *TraceExpecter) ExpectTxnEvents(t Validator, want []WantEvent) {}

// ExpectSpanEvents TODO
func (te *TraceExpecter) ExpectSpanEvents(t Validator, want []WantEvent) {}

// ExpectCustomEvents TODO
func (te *TraceExpecter) ExpectCustomEvents(t Validator, want []WantEvent) {}

// ExpectErrors TODO
func (te *TraceExpecter) ExpectErrors(t Validator, want []WantError) {}

// ExpectErrorEvents TODO
func (te *TraceExpecter) ExpectErrorEvents(t Validator, want []WantEvent) {}

// ExpectMetrics TODO
func (te *TraceExpecter) ExpectMetrics(t Validator, want []WantMetric) {}

// ExpectMetricsPresent TODO
func (te *TraceExpecter) ExpectMetricsPresent(t Validator, want []WantMetric) {}

// ExpectTxnMetrics TODO
func (te *TraceExpecter) ExpectTxnMetrics(t Validator, want WantTxn) {}

// ExpectTxnTraces TODO
func (te *TraceExpecter) ExpectTxnTraces(t Validator, want []WantTxnTrace) {}

// ExpectSlowQueries TODO
func (te *TraceExpecter) ExpectSlowQueries(t Validator, want []WantSlowQuery) {}
