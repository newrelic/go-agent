package nrotelhybrid

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"testing"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/crossagent"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/newrelic/go-agent/v3/newrelic/integrationsupport"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
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

type ExternalCall struct {
	url     string
	headers http.Header
}

const (
	// Commands
	CommandDoWorkInSpan          string = "DoWorkInSpan"
	CommandDoWorkInTransaction   string = "DoWorkInTransaction"
	CommandDoWorkInSegment       string = "DoWorkInSegment"
	CommandAddOTelAttribute      string = "AddOTelAttribute"
	CommandRecordExceptionOnSpan string = "RecordExceptionOnSpan"
	CommandSimulateExternalCall  string = "SimulateExternalCall"
	CommandOTelInjectHeaders     string = "OTelInjectHeaders"

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

	ParameterCurrentTransactionSampled string = "currentTransaction.sampled"
	ParameterInjectedTraceId           string = "injected.traceId"
	ParameterInjectedSpanId            string = "injected.spanId"
	ParameterInjectedSampled           string = "injected.sampled"
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
		if i != 5 {
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
			otel.SetTextMapPropagator(propagation.TraceContext{})
			// Run Operation

			RunOperation(t, ctx, tc.Operations, &app, nil)
			// agentOutput
			app.ExpectTxnEventsPartial(t, expectedTxnEvents)
			app.ExpectSpanEventsPartial(t, expectedSpanEvents)
		})
	}
}

func RunOperation(t *testing.T, ctx context.Context, operations []OtelTestCaseOperation, app *integrationsupport.ExpectApp, externalCall *ExternalCall) {
	for _, op := range operations {
		switch op.Command {
		case CommandDoWorkInSpan:
			// use spanKind and spanName to create span
			tracer := Tracer("test")
			ctx, span := tracer.Start(ctx, op.Parameters.SpanName, oteltrace.WithSpanKind(oteltrace.SpanKind(getSpanKind(op.Parameters.SpanKind))))
			// run child operations
			RunOperation(t, ctx, op.ChildOperations, app, externalCall)
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
			RunOperation(t, txnCtx, op.ChildOperations, app, externalCall)
			txn.End()
		case CommandDoWorkInSegment:
			// do work in segment
			// begin a NR Segment
			txn := newrelic.FromContext(ctx)
			seg := txn.StartSegment(op.Parameters.SegmentName)
			RunOperation(t, ctx, op.ChildOperations, app, externalCall)
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
			RunOperation(t, ctx, op.ChildOperations, app, externalCall)
		case CommandRecordExceptionOnSpan:
			// Record Error on Span pulled from context
			// Use OTel API to add error to the CURRENT span
			span := oteltrace.SpanFromContext(ctx)
			span.RecordError(errors.New(op.Parameters.ErrorMessage))
			RunOperation(t, ctx, op.ChildOperations, app, externalCall)
		case CommandSimulateExternalCall:
			// Use url to simulate external call
			// Must create a request header collection in order to inject DT headers for other commands
			req, err := http.NewRequestWithContext(ctx, "GET", op.Parameters.URL, nil)
			if err != nil {
				t.Errorf("Could not build request")
			}
			RunOperation(t, ctx, op.ChildOperations, app, &ExternalCall{
				url:     op.Parameters.URL,
				headers: req.Header,
			})
		case CommandOTelInjectHeaders:
			// use simulated external call
			// no parameters needed
			otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(externalCall.headers))
			extractedCtx := propagation.TraceContext{}.Extract(ctx, propagation.HeaderCarrier(externalCall.headers))

			RunOperation(t, extractedCtx, op.ChildOperations, app, externalCall)
			for _, assertion := range op.Assertions {
				rule := assertion.Rule
				switch rule.Operator {
				case OperatorEquals:
					left := populateEqualsOperator(extractedCtx, rule.Parameters.Left, externalCall)
					if left == "" {
						t.Errorf("Could not populate left equals for %v", rule.Parameters.Left)
					}
					right := populateEqualsOperator(extractedCtx, rule.Parameters.Right, externalCall)
					if right == "" {
						t.Errorf("Could not populate left equals for %v", rule.Parameters.Right)
					}
					if left != right {
						t.Errorf("%v: %v does not equal %v: %v", rule.Parameters.Left, left, rule.Parameters.Right, right)
					}
				default:
					continue
				}
			}

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

func populateEqualsOperator(ctx context.Context, parameter string, externalCall *ExternalCall) string {
	switch parameter {
	case ParameterCurrentOTelSpanTraceId:
		return oteltrace.SpanContextFromContext(ctx).TraceID().String()
	case ParameterCurrentOTelSpanSpanId:
		return oteltrace.SpanContextFromContext(ctx).SpanID().String()
	case ParameterCurrentTransactionTraceId:
		return newrelic.FromContext(ctx).GetTraceMetadata().TraceID
	case ParameterCurrentSegmentSpanId:
		return newrelic.FromContext(ctx).GetTraceMetadata().SpanID
	case ParameterCurrentTransactionSampled:
		return strconv.FormatBool(newrelic.FromContext(ctx).IsSampled())
	case ParameterInjectedTraceId, ParameterInjectedSpanId, ParameterInjectedSampled:
		if externalCall == nil {
			return ""
		}
		// Parse the injected traceparent header independently of the injector.
		sc := oteltrace.SpanContextFromContext(
			propagation.TraceContext{}.Extract(ctx, propagation.HeaderCarrier(externalCall.headers)),
		)
		if !sc.IsValid() {
			return ""
		}
		switch parameter {
		case ParameterInjectedTraceId:
			return sc.TraceID().String()
		case ParameterInjectedSpanId:
			return sc.SpanID().String()
		default:
			return strconv.FormatBool(sc.IsSampled())
		}
	}
	return ""
}
