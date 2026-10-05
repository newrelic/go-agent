package nrotelhybrid

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/crossagent"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/newrelic/go-agent/v3/newrelic/integrationsupport"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
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
	Value               int    `json:"value"` // KEEPING AS INT FOR NOW SINCE THAT IS THE ONLY CASE
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
	CommandDoWorkInSpan          string = "DoWorkInSpan"
	CommandDoWorkInTransaction   string = "DoWorkInTransaction"
	CommandDoWorkInSegment       string = "DoWorkInSegment"
	CommandAddOTelAttribute      string = "AddOTelAttribute"
	CommandRecordExceptionOnSpan string = "RecordExceptionOnSpan"

	// Operators
	OperatorNotValid string = "NotValid"
	OperatorEquals   string = "Equals"

	// Objects
	NotValidObjectCurrentOtelSpan    string = "currentOTelSpan"
	NotValidObjectCurrentTransaction string = "currentTransaction"

	// Equals operands
	OperandCurrentOtelSpanTraceID    string = "currentOTelSpan.traceId"
	OperandCurrentOtelSpanSpanID     string = "currentOTelSpan.spanId"
	OperandCurrentTransactionTraceID string = "currentTransaction.traceId"
	OperandCurrentSegmentSpanID      string = "currentSegment.spanId"
)

func TestOtelTracing(t *testing.T) {
	var tcs []OtelTracingTestCase
	data, err := crossagent.ReadFile("otelhybrid/TestCaseDefinitions.json")
	if err != nil {
		t.Fatal(err)
	}

	if err := json.Unmarshal(data, &tcs); err != nil {
		t.Fatal(err)
	}

	for i, tc := range tcs {
		if i != 1 && i != 4 {
			continue
		}
		expectedTxnEvents, expectedSpanEvents := createExpectedEvents(tc.AgentOutput)
		t.Run(tc.TestDescription, func(t *testing.T) {
			// Each test case gets its own app, so events from an earlier case do
			// not carry into this case's agent output expectations.
			app := integrationsupport.NewTestApp(
				integrationsupport.SampleEverythingReplyFn,
				integrationsupport.ConfigFullTraces,
			)
			ctx := context.Background()
			processor := NewHybridProcessor(app.Application)
			tp := trace.NewTracerProvider(trace.WithSpanProcessor(processor))
			shutdown := func(ctx context.Context) error {
				err := tp.Shutdown(ctx)
				return err
			}
			defer shutdown(ctx)
			otel.SetTracerProvider(tp)
			// Run Operation

			RunOperation(t, ctx, tc.Operations, &app)
			// agentOutput
			app.ExpectTxnEventsPartial(t, expectedTxnEvents)
			app.ExpectSpanEventsPartial(t, expectedSpanEvents)
		})
	}
}

func RunOperation(t *testing.T, ctx context.Context, operations []OtelTestCaseOperation, app *integrationsupport.ExpectApp) {
	for _, op := range operations {
		switch op.Command {
		case CommandDoWorkInSpan:
			// use spanKind and spanName to create span
			tracer := Tracer("test")
			ctx, span := tracer.Start(ctx, op.Parameters.SpanName, oteltrace.WithSpanKind(oteltrace.SpanKind(getSpanKind(op.Parameters.SpanKind))))
			// run child operations
			RunOperation(t, ctx, op.ChildOperations, app)
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
						if oteltrace.SpanFromContext(ctx).SpanContext().IsValid() {
							t.Errorf("%s: current OTel span is valid", assertion.Description)
						}
					case NotValidObjectCurrentTransaction:
						if newrelic.FromContext(ctx) != nil {
							t.Errorf("Expected no transaction, got a started transaction")
						}
					}
				case OperatorEquals:
					left := resolveOperand(t, ctx, rule.Parameters.Left)
					right := resolveOperand(t, ctx, rule.Parameters.Right)
					if left != right {
						t.Errorf("%s: %s = %q but %s = %q", assertion.Description,
							rule.Parameters.Left, left, rule.Parameters.Right, right)
					}
				default:
					continue
				}
			}
			span.End() // should work even with a no-op span
			// end
		case CommandDoWorkInTransaction:
			// do work in transaction
			// begin a NR Transaction
			txn := app.StartTransaction(op.Parameters.TransactionName)
			txnCtx := newrelic.NewContext(ctx, txn)
			// run child operations
			RunOperation(t, txnCtx, op.ChildOperations, app)
			txn.End()
		case CommandDoWorkInSegment:
			// do work in segment
			// begin a NR Segment
			txn := newrelic.FromContext(ctx)
			seg := txn.StartSegment(op.Parameters.SegmentName)
			RunOperation(t, ctx, op.ChildOperations, app)
			seg.End()
		case CommandAddOTelAttribute:
			// add OTEL attribute
			// Use OTel API to add an attribute to the CURRENT span
			span := oteltrace.SpanFromContext(ctx)
			kv := attribute.KeyValue{
				Key:   attribute.Key(op.Parameters.Name),
				Value: attribute.IntValue(op.Parameters.Value), // SETTING AS INT SINCE THOSE ARE ONLY CASES NOW
			}
			span.SetAttributes(kv)
		case CommandRecordExceptionOnSpan:
			// Record Error on Span pulled from context
			// Use OTel API to add error to the CURRENT span
			span := oteltrace.SpanFromContext(ctx)
			span.RecordError(errors.New(op.Parameters.ErrorMessage))
		default:
			continue
		}
	}
}

// resolveOperand reads one side of an Equals rule.
func resolveOperand(t *testing.T, ctx context.Context, operand string) string {
	t.Helper()
	switch operand {
	case OperandCurrentOtelSpanTraceID:
		return oteltrace.SpanFromContext(ctx).SpanContext().TraceID().String()
	case OperandCurrentOtelSpanSpanID:
		return oteltrace.SpanFromContext(ctx).SpanContext().SpanID().String()
	case OperandCurrentTransactionTraceID:
		return newrelic.FromContext(ctx).GetTraceMetadata().TraceID
	case OperandCurrentSegmentSpanID:
		// GetTraceMetadata reports the span ID of the segment on top of the
		// transaction's stack.
		return newrelic.FromContext(ctx).GetTraceMetadata().SpanID
	default:
		t.Errorf("Equals operand %q is not implemented", operand)
		return ""
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

func createExpectedEvents(agentOutput OtelTestCaseAgentOutput) ([]internal.WantEvent, []internal.WantEvent) {
	transactionsAgentOutput := agentOutput.Transactions
	spansAgentOutput := agentOutput.Spans

	var transactionWantEvents []internal.WantEvent
	for _, txn := range transactionsAgentOutput {
		transactionWantEvents = append(transactionWantEvents, internal.WantEvent{
			Intrinsics: map[string]interface{}{
				"name": txn.Name,
			},
		})
	}

	var spanWantEvents []internal.WantEvent
	for _, span := range spansAgentOutput {
		if span.EntryPoint {
			// ExpectSpanEventsPartial drops the transaction's root span event,
			// so the entry point span is not expected either.
			continue
		}
		intrinsics := map[string]interface{}{
			"name": span.Name,
		}
		agentAttributes := map[string]interface{}{}

		if val, ok := span.Attributes[NRErrorMessage]; ok {
			switch reflect.TypeOf(val).Kind() {
			case reflect.String:
				agentAttributes[NRErrorMessage] = string(val.(string))
			}
		}

		spanWantEvents = append(spanWantEvents, internal.WantEvent{
			Intrinsics:      intrinsics,
			AgentAttributes: agentAttributes,
		})
	}
	return transactionWantEvents, spanWantEvents
}
