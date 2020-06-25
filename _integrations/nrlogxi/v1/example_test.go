// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrlogxi_test

import (
	log "github.com/mgutz/logxi/v1"
	newrelic "github.com/newrelic/go-agent"
	nrlogxi "github.com/newrelic/go-agent/_integrations/nrlogxi/v1"
)

func Example() {
	cfg := newrelic.NewConfig("Example App", "__YOUR_NEWRELIC_LICENSE_KEY__")

	// Create a new logxi logger:
	l := log.New("newrelic")
	l.SetLevel(log.LevelInfo)

	// Use nrlogxi to register the logger with the agent:
	cfg.Logger = nrlogxi.New(l)

	newrelic.NewApplication(cfg)
}
