// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// +build !go1.9
// This build tag is necessary because Infinite Tracing is only supported for Go version 1.9 and up

package newrelic

import "testing"

func TestSupported8TVersion(t *testing.T) {
	_, err := NewApplication(
		ConfigLicense("1234567890123456789012345678901234567890"),
		ConfigAppName("name"),
		func(c *Config) {
			c.InfiniteTracing.TraceObserver.Host = "localhost"
		},
	)
	if nil == err {
		t.Error("expected unsupported version error for 8T but got none")
	}
}
