package nrslog

import (
	"context"
	"log/slog"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func init() { internal.TrackUsage("integration", "logging", "slog") }

func transformAttributes(c map[string]interface{}) []any {
	attrs := make([]any, 0, len(c))
	for k, v := range c {
		attrs = append(attrs, slog.Any(k, v))
	}
	return attrs
}

type shim struct{ logger *slog.Logger }

func (s *shim) Error(msg string, c map[string]interface{}) {
	s.logger.Error(msg, transformAttributes(c)...)
}

func (s *shim) Warn(msg string, c map[string]interface{}) {
	s.logger.Warn(msg, transformAttributes(c)...)
}

func (s *shim) Info(msg string, c map[string]interface{}) {
	s.logger.Info(msg, transformAttributes(c)...)
}

func (s *shim) Debug(msg string, c map[string]interface{}) {
	s.logger.Debug(msg, transformAttributes(c)...)
}

func (s *shim) DebugEnabled() bool {
	return s.logger.Enabled(context.Background(), slog.LevelDebug)
}

// Transform turns a *slog.Logger into a newrelic.Logger.
func Transform(l *slog.Logger) newrelic.Logger { return &shim{logger: l} }

// ConfigLogger configures the newrelic.Application to send log messages to the
// provided slog.
func ConfigLogger(l *slog.Logger) newrelic.ConfigOption {
	return newrelic.ConfigLogger(Transform(l))
}
