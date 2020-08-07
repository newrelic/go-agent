package internal

import (
	"go.opentelemetry.io/otel/api/trace/testtrace"
)

// OpenTelemetryExpect implements internal.Expect for use in testing.
type OpenTelemetryExpect struct {
	Spans *testtrace.StandardSpanRecorder
}

func expectSpan(t Validator, want WantSpan, span *testtrace.Span) {
	t.Helper()
	name := span.Name()
	if want.Name != "" {
		if name != want.Name {
			t.Errorf("Incorrect span name:\n\texpect=%s actual=%s",
				want.Name, name)
		}
	}
	spanCtx := span.SpanContext()
	if want.SpanID != "" {
		if id := spanCtx.SpanID.String(); id != want.SpanID {
			t.Errorf("Incorrect id for span '%s':\n\texpect=%s actual=%s",
				name, want.SpanID, id)
		}
	}
	if want.TraceID != "" {
		if id := spanCtx.TraceID.String(); id != want.TraceID {
			t.Errorf("Incorrect trace id for span '%s':\n\texpect=%s actual=%s",
				name, want.TraceID, id)
		}
	}
	if want.ParentID != "" {
		id := span.ParentSpanID().String()
		if want.ParentID == MatchAnyParent {
			if id == MatchNoParent {
				t.Errorf("Incorrect parent id for span '%s': expected a parent but found none",
					name)
			}
		} else if id != want.ParentID {
			t.Errorf("Incorrect parent id for span '%s':\n\texpect=%s actual=%s",
				name, want.ParentID, id)
		}
	}
}

func (e *OpenTelemetryExpect) spans() []*testtrace.Span {
	return e.Spans.Completed()
}

// ExpectSpanEvents TODO
func (e *OpenTelemetryExpect) ExpectSpanEvents(t Validator, want []WantSpan) {
	t.Helper()
	spans := e.spans()
	if len(want) != len(spans) {
		t.Errorf("Incorrect number of recorded spans: expect=%d actual=%d",
			len(want), len(spans))
		return
	}
	for i := 0; i < len(want); i++ {
		expectSpan(t, want[i], spans[i])
	}
}

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
