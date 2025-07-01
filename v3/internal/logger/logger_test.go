package logger

import (
	"os"
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
	logger := New(os.Stdout, true)

	m := map[string]interface{}{"key1": "val1", "key2": "val2"}
	logger.Error("error message", m)
	logger.Warn("warn message", m)
	logger.Info("info message", m)
	logger.Debug("info message", m)

	//capture stdout and cmp

	enabled := logger.DebugEnabled()
	if !enabled {
		t.Error("Debug logging is not enabled")
	}
}
