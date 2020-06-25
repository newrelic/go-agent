// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrlogrus sends go-agent log messages to
// https://github.com/sirupsen/logrus.
//
// Use this package if you are using logrus in your application and would like
// the go-agent log messages to end up in the same place.  If you are using
// the logrus standard logger, assign the newrelic.Config.Logger field to
// nrlogrus.StandardLogger():
//
//	cfg := newrelic.NewConfig("Your Application Name", "__YOUR_NEW_RELIC_LICENSE_KEY__")
//	cfg.Logger = nrlogrus.StandardLogger()
//
// If you are using a particular logrus Logger instance, assign the
// newrelic.Config.Logger field to the the output of nrlogrus.Transform:
//
//	l := logrus.New()
//	l.SetLevel(logrus.DebugLevel)
//	cfg := newrelic.NewConfig("Your Application Name", "__YOUR_NEW_RELIC_LICENSE_KEY__")
//	cfg.Logger = nrlogrus.Transform(l)
//
// This package requires logrus version v1.1.0 and above.
package nrlogrus

import (
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/internal"
	"github.com/sirupsen/logrus"
)

func init() { internal.TrackUsage("integration", "logging", "logrus") }

type shim struct {
	e *logrus.Entry
	l *logrus.Logger
}

func (s *shim) Error(msg string, c map[string]interface{}) {
	s.e.WithFields(c).Error(msg)
}
func (s *shim) Warn(msg string, c map[string]interface{}) {
	s.e.WithFields(c).Warn(msg)
}
func (s *shim) Info(msg string, c map[string]interface{}) {
	s.e.WithFields(c).Info(msg)
}
func (s *shim) Debug(msg string, c map[string]interface{}) {
	s.e.WithFields(c).Debug(msg)
}
func (s *shim) DebugEnabled() bool {
	lvl := s.l.GetLevel()
	return lvl >= logrus.DebugLevel
}

// StandardLogger returns a newrelic.Logger which forwards agent log messages to
// the logrus package-level exported logger.
func StandardLogger() newrelic.Logger {
	return Transform(logrus.StandardLogger())
}

// Transform turns a *logrus.Logger into a newrelic.Logger.
func Transform(l *logrus.Logger) newrelic.Logger {
	return &shim{
		l: l,
		e: l.WithFields(logrus.Fields{
			"component": "newrelic",
		}),
	}
}
