package internal

import (
	"go.opentelemetry.io/otel/api/trace/testtrace"
)

type TraceExpecter struct {
	*testtrace.Tracer
}

func (te *TraceExpecter) ExpectCustomEvents(t Validator, want []WantEvent)    {}
func (te *TraceExpecter) ExpectErrors(t Validator, want []WantError)          {}
func (te *TraceExpecter) ExpectErrorEvents(t Validator, want []WantEvent)     {}
func (te *TraceExpecter) ExpectMetrics(t Validator, want []WantMetric)        {}
func (te *TraceExpecter) ExpectMetricsPresent(t Validator, want []WantMetric) {}
func (te *TraceExpecter) ExpectTxnMetrics(t Validator, want WantTxn)          {}
func (te *TraceExpecter) ExpectTxnTraces(t Validator, want []WantTxnTrace)    {}
func (te *TraceExpecter) ExpectSlowQueries(t Validator, want []WantSlowQuery) {}

func (te *TraceExpecter) ExpectTxnEvents(t Validator, want []WantEvent)  {}
func (te *TraceExpecter) ExpectSpanEvents(t Validator, want []WantEvent) {}
