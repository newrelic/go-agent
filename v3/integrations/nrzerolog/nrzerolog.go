// Copyright 2021 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrzerolog supports https://github.com/rs/zerolog
//
// Wrap your zerolog Logger using nrzerolog.Transform to send agent log messages to zerolog.
package nrzerolog

import (
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/rs/zerolog"
)

func init() { internal.TrackUsage("integration", "logging", "zerolog") }

type shim struct{ logger *zerolog.Logger }

func (s *shim) Error(msg string, c map[string]interface{}) {
	s.logger.Error().Fields(c).Msg(msg)
}
func (s *shim) Warn(msg string, c map[string]interface{}) {
	s.logger.Warn().Fields(c).Msg(msg)
}
func (s *shim) Info(msg string, c map[string]interface{}) {
	s.logger.Info().Fields(c).Msg(msg)
}
func (s *shim) Debug(msg string, c map[string]interface{}) {
	s.logger.Debug().Fields(c).Msg(msg)
}
func (s *shim) DebugEnabled() bool {
	return s.logger.GetLevel() == zerolog.DebugLevel
}

// Transform turns a *zerolog.Logger into a newrelic.Logger.
func Transform(l *zerolog.Logger) newrelic.Logger { return &shim{logger: l} }

// ConfigLogger configures the newrelic.Application to send log messsages to the
// provided zerolog logger.
func ConfigLogger(l *zerolog.Logger) newrelic.ConfigOption {
	return newrelic.ConfigLogger(Transform(l))
}
