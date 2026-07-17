// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrotelhybrid provides a hybrid mode of operation for the New Relic Go Agent, allowing
// it to also provide access to the OpenTelemetry API functions.
package nrotel

func init() {
	internal.TrackUsage("integration", "nrotelhybrid")
}
