package nrawsbedrock

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	"github.com/newrelic/go-agent/v3/newrelic"
)

// Mock Modeler implementation
type mockModeler struct {
	invokeModelCalled                   bool
	invokeModelWithResponseStreamCalled bool
	output                              *bedrockruntime.InvokeModelOutput
	streamOutput                        *bedrockruntime.InvokeModelWithResponseStreamOutput
	err                                 error
}

func (m *mockModeler) InvokeModel(ctx context.Context, input *bedrockruntime.InvokeModelInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error) {
	m.invokeModelCalled = true
	return m.output, m.err
}

func (m *mockModeler) InvokeModelWithResponseStream(ctx context.Context, input *bedrockruntime.InvokeModelWithResponseStreamInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelWithResponseStreamOutput, error) {
	m.invokeModelWithResponseStreamCalled = true
	return m.streamOutput, m.err
}

func testApp() integrationsupport.ExpectApp {
	return integrationsupport.NewTestApp(
		integrationsupport.SampleEverythingReplyFn,
		integrationsupport.DTEnabledCfgFn,
		newrelic.ConfigCodeLevelMetricsEnabled(false),
		newrelic.ConfigAIMonitoringEnabled(true),
		newrelic.ConfigAIMonitoringRecordContentEnabled(true),
	)
}

func TestIsEnabled(t *testing.T) {
	app := testApp()
	enabled, recordContent := isEnabled(app.Application, false)
	if !enabled || !recordContent {
		t.Error("Expected AI monitoring and record content to be enabled")
	}
}

func TestInvokeModel_Success(t *testing.T) {
	app := testApp()
	modeler := &mockModeler{
		output: &bedrockruntime.InvokeModelOutput{
			Body: []byte(`{"completion":"test output"}`),
		},
	}
	params := &bedrockruntime.InvokeModelInput{
		ModelId: strPtr("test-model"),
		Body:    []byte(`{"prompt":"test"}`),
	}
	output, err := InvokeModel(app.Application, modeler, context.Background(), params)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if output == nil || string(output.Body) != `{"completion":"test output"}` {
		t.Error("Unexpected output from InvokeModel")
	}
	if !modeler.invokeModelCalled {
		t.Error("Expected InvokeModel to be called on modeler")
	}
}

func TestInvokeModelWithResponseStream_Success(t *testing.T) {
	app := testApp()
	modeler := &mockModeler{
		streamOutput: &bedrockruntime.InvokeModelWithResponseStreamOutput{},
	}
	params := &bedrockruntime.InvokeModelWithResponseStreamInput{
		ModelId: strPtr("test-model"),
		Body:    []byte(`{"prompt":"test"}`),
	}
	resp, err := InvokeModelWithResponseStream(app.Application, modeler, context.Background(), params)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if resp.Response == nil {
		t.Error("Expected non-nil Response in ResponseStream")
	}
	if !modeler.invokeModelWithResponseStreamCalled {
		t.Error("Expected InvokeModelWithResponseStream to be called on modeler")
	}
}

func TestRecordEvent_NilSafe(t *testing.T) {
	var s *ResponseStream
	if err := s.RecordEvent([]byte("data")); err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
}

func TestClose_NilSafe(t *testing.T) {
	var s *ResponseStream
	if err := s.Close(); err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
}

func TestParseModelData_HandlesInputOutput(t *testing.T) {
	app := testApp()
	modelID := "test-model"
	meta := map[string]any{}
	input := []byte(`{"prompt":"hello"}`)
	output := []byte(`{"completion":"world"}`)
	inputs, outputs, sys := parseModelData(app.Application, modelID, meta, input, output, nil, false)
	if len(inputs) == 0 || len(outputs) == 0 {
		t.Error("Expected non-empty inputs and outputs")
	}
	if sys != "" {
		t.Error("Expected empty system message")
	}
}

func strPtr(s string) *string { return &s }
