package newrelic

import (
	"io"

	"github.com/newrelic/go-agent/internal/logger"
)

// Logger is interface that groups the logging methods used in the go-agent.
// Loggers must be safe for use in multiple goroutines.
//
// For an example implementation, see: _integrations/nrlogrus/nrlogrus.go
type Logger interface {
	Error(msg string, context map[string]interface{})
	Warn(msg string, context map[string]interface{})
	Info(msg string, context map[string]interface{})
	Debug(msg string, context map[string]interface{})
	DebugEnabled() bool
}

// NewLogger creates a basic Logger at info level.
func NewLogger(w io.Writer) Logger {
	return logger.New(w, false)
}

// NewDebugLogger creates a basic Logger at debug level.
func NewDebugLogger(w io.Writer) Logger {
	return logger.New(w, true)
}
