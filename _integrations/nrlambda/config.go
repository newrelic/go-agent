// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrlambda

import (
	"os"
	"time"

	newrelic "github.com/newrelic/go-agent"
)

// NewConfig populates a newrelic.Config with correct default settings for a
// Lambda serverless environment.  NewConfig will populate fields based on
// environment variables common to all New Relic agents that support Lambda.
// Environment variables NEW_RELIC_ACCOUNT_ID, NEW_RELIC_TRUSTED_ACCOUNT_KEY,
// and NEW_RELIC_PRIMARY_APPLICATION_ID configure fields required for
// distributed tracing.  Environment variable NEW_RELIC_APDEX_T may be used to
// set a custom apdex threshold.
func NewConfig() newrelic.Config {
	return newConfigInternal(os.Getenv)
}

func newConfigInternal(getenv func(string) string) newrelic.Config {
	cfg := newrelic.NewConfig("", "")

	cfg.ServerlessMode.Enabled = true

	cfg.ServerlessMode.AccountID = getenv("NEW_RELIC_ACCOUNT_ID")
	cfg.ServerlessMode.TrustedAccountKey = getenv("NEW_RELIC_TRUSTED_ACCOUNT_KEY")
	cfg.ServerlessMode.PrimaryAppID = getenv("NEW_RELIC_PRIMARY_APPLICATION_ID")

	cfg.DistributedTracer.Enabled = true

	if s := getenv("NEW_RELIC_APDEX_T"); "" != s {
		if apdex, err := time.ParseDuration(s + "s"); nil == err {
			cfg.ServerlessMode.ApdexThreshold = apdex
		}
	}

	return cfg
}
