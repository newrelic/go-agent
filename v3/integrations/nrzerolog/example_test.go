// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrzerolog_test

import (
	"os"

	newrelic "github.com/newrelic/go-agent"
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
