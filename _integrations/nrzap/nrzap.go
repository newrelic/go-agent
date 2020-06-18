// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrzap supports https://github.com/uber-go/zap
//
// Wrap your zap Logger using nrzap.Transform to send agent log messages to zap.
package nrzap

import (
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/internal"
	"go.uber.org/zap"
)

func init() { internal.TrackUsage("integration", "logging", "zap") }

type shim struct{ logger *zap.Logger }

func transformAttributes(atts map[string]interface{}) []zap.Field {
	fs := make([]zap.Field, 0, len(atts))
	for key, val := range atts {
		fs = append(fs, zap.Any(key, val))
	}
	return fs
}

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
	ce := s.logger.Check(zap.DebugLevel, "debugging")
	return ce != nil
}

// Transform turns a *zap.Logger into a newrelic.Logger.
func Transform(l *zap.Logger) newrelic.Logger { return &shim{logger: l} }
