package nrotelhybrid

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/crossagent"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/newrelic/go-agent/v3/newrelic/integrationsupport"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

type OtelTracingTestCase struct {
	TestDescription string                  `json:"testDescription"`
	Operations      []OtelTestCaseOperation `json:"operations"`
	AgentOutput     OtelTestCaseAgentOutput `json:"agentOutput"` // optional
}

type OtelTestCaseOperation struct {
	Command         string                  `json:"command"`
	Parameters      OtelTestCaseParameters  `json:"parameters"`
	ChildOperations []OtelTestCaseOperation `json:"childOperations"`
	Assertions      []OtelTestCaseAssertion `json:"assertions"` // run before Operation completes but after ChildOperations
}

type OtelTestCaseParameters struct {
	SpanName            string `json:"spanName"`
	SpanKind            string `json:"spanKind"`
	TransactionName     string `json:"transactionName"`
	SegmentName         string `json:"segmentName"`
	Name                string `json:"name"`
	Value               any    `json:"value"`
	ErrorMessage        string `json:"errorMessage"`
	URL                 string `json:"url"`
	TraceIdInHeader     string `json:"traceIdInHeader"`
	SpanIdInHeader      string `json:"spanIdInHeader"`
	SampledFlagInHeader string `json:"sampledFlagInHeader"`
}

type OtelTestCaseAssertion struct {
	Description string           `json:"description"`
	Rule        OtelTestCaseRule `json:"rule"`
}

type OtelTestCaseRule struct {
	Operator   string                     `json:"operator"`
	Parameters OtelTestCaseRuleParameters `json:"parameters"`
}

type OtelTestCaseRuleParameters struct {
	Object string `json:"object"`
	Left   string `json:"left"`
	Right  string `json:"right"`
	Value  string `json:"value"`
}

type OtelTestCaseAgentOutput struct {
	Transactions []OtelTestCaseTransaction `json:"transactions"`
	Spans        []OtelTestCaseSpan        `json:"spans"`
}

type OtelTestCaseTransaction struct {
	Name string `json:"name"`
}

type OtelTestCaseSpan struct {
	Name       string         `json:"name"`
	Category   string         `json:"category"`
	ParentName string         `json:"parentName"`
	EntryPoint bool           `json:"entryPoint"`
	Attributes map[string]any `json:"attributes"`
}

const (
	// Commands
	CommandDoWorkInSpan string = "DoWorkInSpan"

	// Operators
	OperatorNotValid string = "NotValid"

	// Objects
	NotValidObjectCurrentOtelSpan    string = "currentOTelSpan"
	NotValidObjectCurrentTransaction string = "currentTransaction"
)

func TestOtelTracing(t *testing.T) {
	var tcs []OtelTracingTestCase
	app := integrationsupport.NewTestApp(
		integrationsupport.SampleEverythingReplyFn,
		integrationsupport.ConfigFullTraces,
	)
	data, err := crossagent.ReadFile("otelhybrid/TestCaseDefinitions.json")
	if err != nil {
		t.Fatal(err)
	}

	if err := json.Unmarshal(data, &tcs); err != nil {
		t.Fatal(err)
	}

	for i, tc := range tcs {
		if i > 0 {
			// only doing 1st test case so far
			break
		}
		t.Run(tc.TestDescription, func(t *testing.T) {
			ctx := context.Background()
			processor := NewHybridProcessor(app.Application)
			tp := trace.NewTracerProvider(trace.WithSpanProcessor(processor))
			shutdown := func(ctx context.Context) error {
				err := tp.Shutdown(ctx)
				return err
			}
			defer shutdown(ctx)
			otel.SetTracerProvider(tp)

			for _, op := range tc.Operations {
				switch op.Command {
				case CommandDoWorkInSpan:
					// use spanKind and spanName to create span
					tracer := Tracer("test")
					spanCtx, span := tracer.Start(ctx, op.Parameters.SpanName, oteltrace.WithSpanKind(oteltrace.SpanKind(getSpanKind(op.Parameters.SpanKind))))
					// run child span
					// run assertions
					for _, assertion := range op.Assertions {
						rule := assertion.Rule
						switch rule.Operator {
						case OperatorNotValid:
							switch rule.Parameters.Object {
							case NotValidObjectCurrentOtelSpan:
								// check if current otel span is no-op
								if reflect.TypeOf(span) != reflect.TypeFor[noop.Span]() {
									t.Errorf("Expected Noop span, got a started span")
								}
								if oteltrace.SpanFromContext(spanCtx).SpanContext().IsValid() {
									t.Errorf("%s: current OTel span is valid", assertion.Description)
								}
							case NotValidObjectCurrentTransaction:
								if newrelic.FromContext(spanCtx) != nil {
									t.Errorf("Expected no transaction, got a started transaction")
								}
							}
						default:
							continue
						}
					}
					span.End() // should work even with a no-op span
					// end
				default:
					continue
				}
			}
			// agentOutput
			app.ExpectTxnEvents(t, []internal.WantEvent{})
			app.ExpectSpanEvents(t, []internal.WantEvent{})
		})
	}
}

func getSpanKind(spanKindStr string) int {
	switch spanKindStr {
	case "Internal":
		return 1
	default:

	}
	return 1
}
