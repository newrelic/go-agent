package log

import "time"

type Level int32

const (
	LevelError Level = iota
	LevelWarning
	LevelInfo
	LevelDebug
)

type Context map[string]interface{}

type Entry struct {
	Level     Level
	Timestamp time.Time
	Event     string
	Context   Context
}

type Hook interface {
	Fire(Entry)
}

var (
	// Logger is fed log entries as they occur.  This value should be set
	// during initialization before other New Relic functions are called:
	// changing it during application execution is a race condition. If you
	// want to change the file or level dynamically then Logger should be
	// set to a Hook guarded by a synchronization primitive.
	Logger Hook
)

func Error(event string, ctx Context) { fire(LevelError, event, ctx) }
func Warn(event string, ctx Context)  { fire(LevelWarning, event, ctx) }
func Info(event string, ctx Context)  { fire(LevelInfo, event, ctx) }
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
