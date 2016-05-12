package log

import (
	"fmt"
	"time"
)

type Level int32

const (
	LevelError Level = iota
	LevelWarning
	LevelInfo
	LevelDebug
)

func (level Level) String() string {
	switch level {
	case LevelError:
		return "Error"
	case LevelWarning:
		return "Warning"
	case LevelInfo:
		return "Info"
	case LevelDebug:
		return "Debug"
	default:
		return fmt.Sprintf("Unknown(%d)", level)
	}
}

type Context map[string]interface{}

type Entry struct {
	Level     Level
	Timestamp time.Time
	Event     string
	Context   Context
}

type Hook interface {
	Fire(*Entry)
}

var (
	// Logger is fed log entries as they occur.  This value should be set
	// during initialization, and before other New Relic functions are
	// called: changing it during application execution is a race condition.
	// If changing the log file or log level dynamically is desirable,
	// Logger should be set to a Hook whose behavior is guarded by a
	// synchronization primitive.
	Logger Hook
)

func Error(event string, cs ...Context) { fire(LevelError, event, cs...) }
func Warn(event string, cs ...Context)  { fire(LevelWarning, event, cs...) }
func Info(event string, cs ...Context)  { fire(LevelInfo, event, cs...) }
func Debug(event string, cs ...Context) { fire(LevelDebug, event, cs...) }

func mergeContexts(cs ...Context) Context {
	switch len(cs) {
	case 0:
		return Context{}
	case 1:
		// A single context is not copied to improve performance.  This
		// assumes that Fire does not modify the Entry's Context.
		return cs[0]
	default:
		c := Context{}
		for _, cx := range cs {
			for key, val := range cx {
				c[key] = val
			}
		}
		return c
	}
}

func fire(level Level, event string, cs ...Context) {
	if nil != Logger {
		entry := Entry{
			Level:     level,
			Timestamp: time.Now(),
			Event:     event,
			Context:   mergeContexts(cs...),
		}
		Logger.Fire(&entry)
	}
}
