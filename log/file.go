package log

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
)

type logFile struct {
	level  Level
	logger *log.Logger
}

// SetLogFile sets up a basic log file for the agent to use.  This function
// modifies the Logger global and should only be used at startup.  The filename
// can be set to a file path, "stdout", or "stderr".
func SetLogFile(filename string, level Level) error {
	l, err := newFile(filename, level)
	if nil != err {
		return err
	}
	Logger = l
	return nil
}

func newFile(location string, level Level) (*logFile, error) {
	var w io.Writer

	switch location {
	case "stdout":
		w = os.Stdout
	case "stderr":
		w = os.Stderr
	default:
		var err error
		w, err = os.OpenFile(location, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
		if nil != err {
			return nil, err
		}
	}

	return &logFile{
		logger: log.New(w, logPid, logFlags),
		level:  level,
	}, nil
}

const logFlags = log.Ldate | log.Ltime | log.Lmicroseconds

var (
	logPid = fmt.Sprintf("(%d) ", os.Getpid())
)

func levelString(l Level) string {
	switch l {
	case LevelError:
		return "Error"
	case LevelWarning:
		return "Warning"
	case LevelInfo:
		return "Info"
	case LevelDebug:
		return "Debug"
	default:
		return fmt.Sprintf("Unknown(%d)", l)
	}
}

func (f *logFile) Fire(e Entry) {
	if e.Level <= f.level {
		js, err := json.Marshal(struct {
			Level   string  `json:"level"`
			Event   string  `json:"event"`
			Context Context `json:"context"`
		}{
			levelString(e.Level),
			e.Event,
			e.Context,
		})
		if nil == err {
			f.logger.Printf(string(js))
		} else {
			f.logger.Printf("unable to marshal log entry: %v", err)
		}
	}
}
