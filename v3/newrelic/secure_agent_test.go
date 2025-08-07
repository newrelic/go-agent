package newrelic

import (
	"net/http"
	"testing"
	"time"

	"github.com/newrelic/go-agent/v3/internal"
)

func TestGetSecurityAgentInterface(t *testing.T) {
	// Test with no-op agent
	secureAgent = noOpSecurityAgent{}
	agent := GetSecurityAgentInterface()
	if agent == nil {
		t.Error("Expected non-nil security agent interface")
	}
	if _, ok := agent.(noOpSecurityAgent); !ok {
		t.Error("Expected noOpSecurityAgent type")
	}
}

func TestNoOpSecurityAgent(t *testing.T) {
	agent := noOpSecurityAgent{}

	// Test RefreshState
	result := agent.RefreshState(map[string]string{"test": "value"})
	if result != false {
		t.Error("Expected RefreshState to return false")
	}

	// Test DeactivateSecurity (should not panic)
	agent.DeactivateSecurity()

	// Test SendEvent
	event := agent.SendEvent("test", "data")
	if event != nil {
		t.Error("Expected SendEvent to return nil")
	}

	// Test IsSecurityActive
	active := agent.IsSecurityActive()
	if active != false {
		t.Error("Expected IsSecurityActive to return false")
	}

	// Test DistributedTraceHeaders (should not panic)
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	agent.DistributedTraceHeaders(req, nil)

	// Test SendExitEvent (should not panic)
	agent.SendExitEvent(nil, nil)

	// Test RequestBodyReadLimit
	limit := agent.RequestBodyReadLimit()
	expected := 300 * 1000
	if limit != expected {
		t.Errorf("Expected RequestBodyReadLimit to return %d, got %d", expected, limit)
	}
}

func TestIsSecurityAgentPresent(t *testing.T) {
	// Save original state
	originalAgent := secureAgent
	defer func() { secureAgent = originalAgent }()

	tests := []struct {
		name     string
		agent    securityAgent
		expected bool
	}{
		{
			name:     "no-op agent",
			agent:    noOpSecurityAgent{},
			expected: false,
		},
		{
			name:     "mock real agent",
			agent:    &mockSecurityAgent{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secureAgent = tt.agent
			result := IsSecurityAgentPresent()
			if result != tt.expected {
				t.Errorf("Expected IsSecurityAgentPresent to return %t with %s", tt.expected, tt.name)
			}
		})
	}
}

func TestApplicationRegisterSecurityAgentNilCases(t *testing.T) {
	// Save original state
	originalAgent := secureAgent
	defer func() { secureAgent = originalAgent }()

	cfgfn := func(cfg *Config) { cfg.Enabled = true }
	reply := &internal.ConnectReply{
		EntityGUID: "test-guid-123",
		RunID:      "123",
		AccountID:  "test-account-id",
	}

	tests := []struct {
		name      string
		app       *Application
		agent     securityAgent
		expectErr string
	}{
		{
			name:      "nil application",
			app:       nil,
			agent:     &mockSecurityAgent{},
			expectErr: "Expected secureAgent to remain unchanged with nil application",
		},
		{
			name:      "application with nil app field",
			app:       &Application{app: nil},
			agent:     &mockSecurityAgent{},
			expectErr: "Expected secureAgent to remain unchanged with nil app field",
		},
		{
			name:      "nil agent",
			app:       testApp(func(r *internal.ConnectReply) { *r = *reply }, cfgfn, t).Application,
			agent:     nil,
			expectErr: "Expected secureAgent to remain unchanged with nil agent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.app.RegisterSecurityAgent(tt.agent)
			if secureAgent != originalAgent {
				t.Error(tt.expectErr)
			}
		})
	}
}

func TestApplicationRegisterSecurityAgentConnected(t *testing.T) {
	// Save original state
	originalAgent := secureAgent
	defer func() { secureAgent = originalAgent }()

	cfgfn := func(cfg *Config) { cfg.Enabled = true }
	reply := &internal.ConnectReply{
		EntityGUID: "test-guid-123",
		RunID:      "123",
		AccountID:  "test-account-id",
	}
	app := testApp(func(r *internal.ConnectReply) { *r = *reply }, cfgfn, t)
	defer app.Shutdown(10 * time.Second)

	// Mock the app.run state to simulate a connected app
	if app.app != nil {
		app.app.run = &appRun{Reply: reply}
		app.app.run.Config.hostname = "test-hostname"
		app.app.run.firstAppName = "test-app"
	}

	mockAgent := &mockSecurityAgent{}
	app.RegisterSecurityAgent(mockAgent)

	// Verify the agent was registered
	if secureAgent != mockAgent {
		t.Error("Expected secureAgent to be set to mockAgent")
	}

	if !mockAgent.refreshStateCalled {
		t.Error("Expected RefreshState to be called during registration")
	}
}

func TestApplicationRegisterSecurityAgentNotConnected(t *testing.T) {
	// Save original state
	originalAgent := secureAgent
	defer func() { secureAgent = originalAgent }()

	// Test with app that's not connected to New Relic
	app := testApp(nil, ConfigEnabled(true), t)
	defer app.Shutdown(10 * time.Second)
	mockAgent := &mockSecurityAgent{}
	app.RegisterSecurityAgent(mockAgent)

	// Verify the agent was registered even if not connected
	if secureAgent != mockAgent {
		t.Error("Expected secureAgent to be set to mockAgent even when not connected")
	}

	if mockAgent.refreshStateCalled {
		t.Error("Expected RefreshState to NOT be called during registration")
	}
}

func TestApplicationUpdateSecurityConfigNilCases(t *testing.T) {
	// Test with nil application
	var app *Application
	app.UpdateSecurityConfig("test config")
	// Should not panic

	// Test with application that has nil app field
	app = &Application{}
	app.UpdateSecurityConfig("test config")
	// Should not panic
}

func TestApplicationUpdateSecurityConfig(t *testing.T) {
	app := testApp(nil, ConfigEnabled(true), t)
	defer app.Shutdown(10 * time.Second)

	app.UpdateSecurityConfig("test config")
	if app.app == nil || app.app.config.Config.Security != "test config" {
		t.Error("Expected security config to be updated")
	}
}

func TestGetLinkedMetaDataNilCases(t *testing.T) {
	// Test with nil app
	metadata := getLinkedMetaData(nil)
	if len(metadata) != 0 {
		t.Error("Expected empty metadata for nil app")
	}

	// Test with app that has nil run
	app := &app{}
	metadata = getLinkedMetaData(app)
	if len(metadata) != 0 {
		t.Error("Expected empty metadata for app with nil run")
	}
}

func TestGetLinkedMetaData(t *testing.T) {
	cfgfn := func(cfg *Config) { cfg.Enabled = true }
	reply := &internal.ConnectReply{
		EntityGUID: "test-guid-123",
		RunID:      "123",
		AccountID:  "test-account-id",
	}
	app := testApp(func(r *internal.ConnectReply) { *r = *reply }, cfgfn, t)
	// Mock the app.run state to simulate a connected app
	if app.app != nil {
		app.app.run = &appRun{Reply: reply}
		app.app.run.Config.hostname = "test-hostname"
		app.app.run.firstAppName = "test-app"
	}
	defer app.Shutdown(10 * time.Second)

	metadata := getLinkedMetaData(app.app)

	// Verify metadata is populated with expected fields and values
	expectedFields := map[string]string{
		"hostname":   "test-hostname",
		"entityName": "test-app",
		"entityGUID": "test-guid-123",
		"agentRunId": "123",
		"accountId":  "test-account-id",
	}

	if len(metadata) != len(expectedFields) {
		t.Errorf("Expected %d metadata fields, got %d", len(expectedFields), len(metadata))
	}

	for field, expectedValue := range expectedFields {
		if value, exists := metadata[field]; !exists {
			t.Errorf("Expected metadata field '%s' to exist", field)
		} else if value != expectedValue {
			t.Errorf("Expected metadata field '%s' to have value '%s', got '%s'", field, expectedValue, value)
		}
	}
}

func TestBodyBufferWrite(t *testing.T) {
	// Save original state
	originalAgent := secureAgent
	defer func() { secureAgent = originalAgent }()

	// Set up mock agent with known limit
	mockAgent := &mockSecurityAgent{limit: 10}
	secureAgent = mockAgent

	tests := []struct {
		name           string
		initialBuf     []byte
		writeData      []byte
		expectedN      int
		expectedBufLen int
		expectedBuf    string
		expectedTrunc  bool
	}{
		{
			name:           "write within limit",
			initialBuf:     nil,
			writeData:      []byte("hello"),
			expectedN:      5,
			expectedBufLen: 5,
			expectedBuf:    "hello",
			expectedTrunc:  false,
		},
		{
			name:           "write exactly to limit",
			initialBuf:     nil,
			writeData:      []byte("1234567890"),
			expectedN:      10,
			expectedBufLen: 10,
			expectedBuf:    "1234567890",
			expectedTrunc:  false,
		},
		{
			name:           "write partial when exceeding limit",
			initialBuf:     []byte("12345"),
			writeData:      []byte("abcdefgh"),
			expectedN:      5,
			expectedBufLen: 10,
			expectedBuf:    "12345abcde",
			expectedTrunc:  true,
		},
		{
			name:           "write when buffer at limit-1",
			initialBuf:     []byte("123456789"),
			writeData:      []byte("abc"),
			expectedN:      1,
			expectedBufLen: 10,
			expectedBuf:    "123456789a",
			expectedTrunc:  true,
		},
		{
			name:           "write when buffer already at limit",
			initialBuf:     []byte("1234567890"),
			writeData:      []byte("x"),
			expectedN:      0,
			expectedBufLen: 10,
			expectedBuf:    "1234567890",
			expectedTrunc:  true,
		},
		{
			name:           "write empty data",
			initialBuf:     nil,
			writeData:      []byte{},
			expectedN:      0,
			expectedBufLen: 0,
			expectedBuf:    "",
			expectedTrunc:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buffer := &BodyBuffer{buf: tt.initialBuf}
			n, err := buffer.Write(tt.writeData)

			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
			if n != tt.expectedN {
				t.Errorf("Expected %d bytes written, got %d", tt.expectedN, n)
			}
			if len(buffer.buf) != tt.expectedBufLen {
				t.Errorf("Expected buffer length %d, got %d", tt.expectedBufLen, len(buffer.buf))
			}
			if string(buffer.buf) != tt.expectedBuf {
				t.Errorf("Expected '%s', got '%s'", tt.expectedBuf, string(buffer.buf))
			}
			if buffer.isDataTruncated != tt.expectedTrunc {
				t.Errorf("Expected truncated=%t, got %t", tt.expectedTrunc, buffer.isDataTruncated)
			}
		})
	}
}

func TestBodyBufferLen(t *testing.T) {
	// Test with nil buffer
	var buffer *BodyBuffer
	if buffer.Len() != 0 {
		t.Error("Expected Len() to return 0 for nil buffer")
	}

	// Test with empty buffer
	buffer = &BodyBuffer{}
	if buffer.Len() != 0 {
		t.Error("Expected Len() to return 0 for empty buffer")
	}

	// Test with data
	buffer.buf = []byte("test")
	if buffer.Len() != 4 {
		t.Errorf("Expected Len() to return 4, got %d", buffer.Len())
	}
}

func TestBodyBufferRead(t *testing.T) {
	// Test with nil buffer
	var buffer *BodyBuffer
	data := buffer.read()
	if len(data) != 0 {
		t.Error("Expected empty slice for nil buffer")
	}

	// Test with buffer containing data
	buffer = &BodyBuffer{buf: []byte("test")}
	data = buffer.read()
	if string(data) != "test" {
		t.Errorf("Expected 'test', got '%s'", string(data))
	}
}

func TestBodyBufferIsBodyTruncated(t *testing.T) {
	// Test with nil buffer
	var buffer *BodyBuffer
	if buffer.isBodyTruncated() {
		t.Error("Expected false for nil buffer")
	}

	// Test with non-truncated buffer
	buffer = &BodyBuffer{}
	if buffer.isBodyTruncated() {
		t.Error("Expected false for non-truncated buffer")
	}

	// Test with truncated buffer
	buffer.isDataTruncated = true
	if !buffer.isBodyTruncated() {
		t.Error("Expected true for truncated buffer")
	}
}

func TestBodyBufferString(t *testing.T) {
	// Test with nil buffer
	var buffer *BodyBuffer
	str, truncated := buffer.String()
	if str != "" || truncated != false {
		t.Error("Expected empty string and false for nil buffer")
	}

	// Test with buffer containing data
	buffer = &BodyBuffer{
		buf:             []byte("test data"),
		isDataTruncated: true,
	}
	str, truncated = buffer.String()
	if str != "test data" {
		t.Errorf("Expected 'test data', got '%s'", str)
	}
	if !truncated {
		t.Error("Expected truncated to be true")
	}
}

// Mock security agent for testing
type mockSecurityAgent struct {
	limit              int
	refreshStateCalled bool
	refreshStateParams map[string]string
}

func (m *mockSecurityAgent) RefreshState(params map[string]string) bool {
	m.refreshStateCalled = true
	m.refreshStateParams = params
	return true
}

func (m *mockSecurityAgent) DeactivateSecurity()                        {}
func (m *mockSecurityAgent) SendEvent(string, ...any) any               { return nil }
func (m *mockSecurityAgent) IsSecurityActive() bool                     { return true }
func (m *mockSecurityAgent) DistributedTraceHeaders(*http.Request, any) {}
func (m *mockSecurityAgent) SendExitEvent(any, error)                   {}
func (m *mockSecurityAgent) RequestBodyReadLimit() int {
	if m.limit > 0 {
		return m.limit
	}
	return 300 * 1000
}
