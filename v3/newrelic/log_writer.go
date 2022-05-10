package newrelic

import (
	"errors"
	"io"
)

type logWriter struct {
	app *Application
	out io.Writer
}

func NewLogWriter(app *Application, out io.Writer) (logWriter, error) {
	if app == nil || app.app == nil {
		return logWriter{}, errors.New("app must not be nil")
	}

	return logWriter{app, out}, nil
}

func (writer logWriter) Write(p []byte) (n int, err error) {
	internalApp := writer.app.app
	if internalApp.config.ApplicationLogging.Enabled && !internalApp.config.Config.HighSecurity {
		logEvent, err := CreateLogEvent(p)
		if err != nil {
			return 0, err
		}
		run, _ := internalApp.getState()

		// Run reply is unable to exlpicitly disable logging features, so we do not check it.
		// If the user wants to disable logging on the server side, they can only set the
		// log event limit to zero, which will set the harvest limit for log events to zero.

		internalApp.Consume(run.Reply.RunID, &logEvent)
	}

	if writer.out != nil {
		return writer.out.Write(p)
	} else {
		return len(p), nil
	}
}
