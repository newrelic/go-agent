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

	var jsonMap map[string]interface{}
	s := strings.Split(b.String(), "\n")
	s = s[:len(s)-1]
	for _, v := range s {
		jsonStr := v[strings.Index(v, "{"):]
		err := json.Unmarshal([]byte(jsonStr), &jsonMap)
		if err != nil {
			t.Errorf("Error %v unmarshaling JSON:", err)
		}
	}
}
