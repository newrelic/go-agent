package log

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
)

type LogFile struct {
	level  Level
	logger *log.Logger
}

// SetLogFile is used to setup a log file and the selected level.  This function
// modifies the unprotected public Logger global, and therefore this function
// should only be used at startup.  The filename can be set to a file path,
// "stdout", or "stderr".
func SetLogFile(filename string, level Level) error {
	l, err := NewFile(filename, level)
	if nil != err {
		return err
	}
	Logger = l
	return nil
}

func NewFile(location string, level Level) (*LogFile, error) {
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

	return &LogFile{
		logger: log.New(w, logPid, logFlags),
		level:  level,
	}, nil
}

const logFlags = log.Ldate | log.Ltime | log.Lmicroseconds

var (
	logPid = fmt.Sprintf("(%d) ", os.Getpid())
)

func (f *LogFile) Fire(e *Entry) {
	if e.Level <= f.level {
		js, err := json.Marshal(struct {
			Level   string  `json:"level"`
			Event   string  `json:"event"`
			Context Context `json:"context"`
		}{
			e.Level.String(),
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
