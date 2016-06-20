// Package log contains the logging system used by the Go Agent.  It is designed
// to be simple and easy to integrate with your application's existing logging.
package log

import "time"

// Level represents the log message type.
type Level int32

const (
	// LevelError is used for critical issues.
	LevelError Level = iota
	// LevelWarning is used for non critical issues.
	LevelWarning
	// LevelInfo is used for information messages.
	LevelInfo
	// LevelDebug includes communication with New Relic's servers and an
	// entry for each transaction.  It will impact performance and is not
	// recommended for production environments.
	LevelDebug
)

// Context contains key value pairs for structured logging.
type Context map[string]interface{}

// Entry is a single log message event.
type Entry struct {
	Level     Level
	Timestamp time.Time
	Event     string
	Context   Context
}

// Hook processes log entries.
type Hook interface {
	Fire(Entry)
}

var (
	// Logger is fed log entries as they occur.  This value should be set
	// during initialization before other New Relic functions are called:
	// changing it during application execution is a race condition.  Logger
	// must be set to a Hook guarded by a synchronization primitive if you
	// want to change the file or level dynamically.
	Logger Hook
)

// Error generates a LevelError log entry.
func Error(event string, ctx Context) { fire(LevelError, event, ctx) }

// Warn generates a LevelWarn log entry.
func Warn(event string, ctx Context) { fire(LevelWarning, event, ctx) }

// Info generates a LevelInfo log entry.
func Info(event string, ctx Context) { fire(LevelInfo, event, ctx) }

// Debug generates a LevelDebug log entry.
func Debug(event string, ctx Context) { fire(LevelDebug, event, ctx) }

func fire(level Level, event string, ctx Context) {
	if nil != Logger {
		Logger.Fire(Entry{
			Level:     level,
			Timestamp: time.Now(),
			Event:     event,
			Context:   ctx,
		})
	}
}
