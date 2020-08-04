package internal

import (
	"go.opentelemetry.io/otel/api/trace/testtrace"
)

// OpenTelemetryExpect implements internal.Expect for use in testing.
type OpenTelemetryExpect struct {
	*testtrace.Tracer
}

// ExpectTxnEvents TODO
func (e *OpenTelemetryExpect) ExpectTxnEvents(t Validator, want []WantEvent) {}

// ExpectSpanEvents TODO
func (e *OpenTelemetryExpect) ExpectSpanEvents(t Validator, want []WantEvent) {}

// ExpectCustomEvents TODO
func (e *OpenTelemetryExpect) ExpectCustomEvents(t Validator, want []WantEvent) {}

// ExpectErrors TODO
func (e *OpenTelemetryExpect) ExpectErrors(t Validator, want []WantError) {}

// ExpectErrorEvents TODO
func (e *OpenTelemetryExpect) ExpectErrorEvents(t Validator, want []WantEvent) {}

// ExpectMetrics TODO
func (e *OpenTelemetryExpect) ExpectMetrics(t Validator, want []WantMetric) {}

// ExpectMetricsPresent TODO
func (e *OpenTelemetryExpect) ExpectMetricsPresent(t Validator, want []WantMetric) {}

// ExpectTxnMetrics TODO
func (e *OpenTelemetryExpect) ExpectTxnMetrics(t Validator, want WantTxn) {}

// ExpectTxnTraces TODO
func (e *OpenTelemetryExpect) ExpectTxnTraces(t Validator, want []WantTxnTrace) {}

// ExpectSlowQueries TODO
func (e *OpenTelemetryExpect) ExpectSlowQueries(t Validator, want []WantSlowQuery) {}
