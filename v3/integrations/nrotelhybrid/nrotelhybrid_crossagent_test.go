package nrotelhybrid

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/newrelic/go-agent/v3/internal/crossagent"
	"github.com/newrelic/go-agent/v3/newrelic/integrationsupport"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
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

// Commands
var CommandDoWorkInSpan string = "DoWorkInSpan"

// Operators
var OperatorNotValid string = "NotValid"

// Objects
var NotValidObjectCurrentOtelSpan = "currentOTelSpan"
var NotValidObjectCurrentTransaction = "currentTransaction"

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

	for _, tc := range tcs {
		t.Run(tc.TestDescription, func(t *testing.T) {
			processor := NewHybridProcessor(app.Application)
			tp := trace.NewTracerProvider(trace.WithSpanProcessor(processor))
			shutdown := func(ctx context.Context) error {
				err := tp.Shutdown(ctx)
				return err
			}
			defer shutdown(context.Background())
			otel.SetTracerProvider(tp)

			for _, op := range tc.Operations {
				switch op.Command {
				case CommandDoWorkInSpan:
					// use spanKind and spanName to create span
					tracer := otel.Tracer("test")
					_, span := tracer.Start(context.Background(), op.Parameters.SpanName, oteltrace.WithSpanKind(oteltrace.SpanKind(getSpanKind(op.Parameters.SpanKind))))
					// run child span
					// run assertions
					for _, assertion := range op.Assertions {
						rule := assertion.Rule
						switch rule.Operator {
						case OperatorNotValid:
							if rule.Parameters.Object == NotValidObjectCurrentOtelSpan {
								// check if current otel span is nil
							} else if rule.Parameters.Object == NotValidObjectCurrentTransaction {

							}
						default:
							continue
						}
					}
					// end
				default:
					continue
				}
			}
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
