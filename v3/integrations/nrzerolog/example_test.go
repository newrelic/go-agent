// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrzerolog_test

import (
	"os"

	"github.com/newrelic/go-agent/v3/integrations/nrzerolog"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/rs/zerolog"
)

func Example() {
	// Create a new zerolog logger:
	zl := zerolog.New(os.Stderr)

	newrelic.NewApplication(
		newrelic.ConfigAppName("Example App"),
		newrelic.ConfigLicense("__YOUR_NEWRELIC_LICENSE_KEY__"),
		// Use nrzerolog to register the logger with the agent:
		nrzerolog.ConfigLogger(&zl),
	)
}
