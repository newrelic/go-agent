package logger

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestShimLogger(t *testing.T) {
	logger := ShimLogger{IsDebugEnabled: true}

	//do nothing
	m := map[string]interface{}{"key1": "val1", "key2": "val2"}
	logger.Error("Do nothing", m)
	logger.Warn("Do nothing", m)
	logger.Info("Do nothing", m)
	logger.Debug("Do nothing", m)
	enabled := logger.DebugEnabled()

	if !enabled {
		t.Error("Debug logging is not enabled")
	}
}

func TestBasicLogger(t *testing.T) {
	b := &bytes.Buffer{}
	logger := New(b, true)

	m := map[string]interface{}{"key1": "val1", "key2": "val2"}
	logger.Error("error message", m)
	logger.Warn("warn message", m)
	logger.Info("info message", m)
	logger.Debug("debug message", m)

	enabled := logger.DebugEnabled()
	if !enabled {
		t.Error("Debug logging is not enabled")
	}

	var expected []string
	expected = append(expected,
		"{\"level\":\"error\",\"msg\":\"error message\",\"context\":{\"key1\":\"val1\",\"key2\":\"val2\"}}",
		"{\"level\":\"warn\",\"msg\":\"warn message\",\"context\":{\"key1\":\"val1\",\"key2\":\"val2\"}}",
		"{\"level\":\"info\",\"msg\":\"info message\",\"context\":{\"key1\":\"val1\",\"key2\":\"val2\"}}",
		"{\"level\":\"debug\",\"msg\":\"debug message\",\"context\":{\"key1\":\"val1\",\"key2\":\"val2\"}}")

	var jsonMap map[string]interface{}
	s := strings.Split(b.String(), "\n")
	s = s[:len(s)-1]
	for i, v := range s {
		jsonStr := v[strings.Index(v, "{"):]
		err := json.Unmarshal([]byte(jsonStr), &jsonMap)
		if err != nil {
			t.Errorf("Error %v unmarshaling JSON:", err)
		}
		if jsonStr != expected[i] {
			t.Errorf("JSON string does not match expected:\n\tExpected: %v\n\tActual: %v", expected[i], jsonStr)
		}
	}
}
