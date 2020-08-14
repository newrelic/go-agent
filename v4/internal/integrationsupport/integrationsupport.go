// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package integrationsupport exists to expose functionality to integration
// packages without adding noise to the public API.
package integrationsupport

import (
	"github.com/newrelic/go-agent/v4/internal"
	"github.com/newrelic/go-agent/v4/newrelic"
	"go.opentelemetry.io/otel/api/trace/testtrace"
)

// SampleAppName is a sample application name.
const SampleAppName = "my app"

// ExpectApp combines Application and Expect, for use in validating data in test apps
type ExpectApp struct {
	internal.Expect
	*newrelic.Application
}

// ConfigFullTraces enables distributed tracing and sets transaction
// trace and transaction trace segment thresholds to zero for full traces.
func ConfigFullTraces(cfg *newrelic.Config) {
	cfg.DistributedTracer.Enabled = true
	cfg.TransactionTracer.Segments.Threshold = 0
	cfg.TransactionTracer.Threshold.IsApdexFailing = false
	cfg.TransactionTracer.Threshold.Duration = 0
}

// NewTestApp creates an ExpectApp with the given ConnectReply function and Config function
func NewTestApp(cfgFn ...newrelic.ConfigOption) ExpectApp {
	sr := new(testtrace.StandardSpanRecorder)
	cfgFn = append(cfgFn, func(cfg *newrelic.Config) {
		tr := testtrace.NewProvider(testtrace.WithSpanRecorder(sr)).Tracer("go-agent-test")
		cfg.OpenTelemetry.Tracer = tr
	})

	app, err := newrelic.NewApplication(cfgFn...)
	if nil != err {
		panic(err)
	}

	return ExpectApp{
		Expect: &internal.OpenTelemetryExpect{
			Spans: sr,
		},
		Application: app,
	}
}

// NewBasicTestApp creates an ExpectApp with the standard testing connect reply function and config
func NewBasicTestApp() ExpectApp {
	return NewTestApp(BasicConfigFn)
}

// BasicConfigFn is a default config function to be used when no special settings are needed for a test app
var BasicConfigFn = func(cfg *newrelic.Config) {
	cfg.Enabled = false
}

// DTEnabledCfgFn is a reusable Config function that sets Distributed Tracing to enabled
var DTEnabledCfgFn = func(cfg *newrelic.Config) {
	cfg.Enabled = false
	cfg.DistributedTracer.Enabled = true
}
