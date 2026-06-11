// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package integrationsupport

import "github.com/newrelic/go-agent/v3/internal"

// Re-export internal test types so external packages can use the test
// infrastructure without importing the restricted internal package.

// Validator is used for testing.
type Validator = internal.Validator

// WantMetric is a metric expectation.
type WantMetric = internal.WantMetric

// WantError is a traced error expectation.
type WantError = internal.WantError

// WantLog is a traced log event expectation.
type WantLog = internal.WantLog

// WantEvent is a transaction or error event expectation.
type WantEvent = internal.WantEvent

// WantTxn provides the expectation parameters to ExpectTxnMetrics.
type WantTxn = internal.WantTxn

// WantTxnTrace is a transaction trace expectation.
type WantTxnTrace = internal.WantTxnTrace

// WantTraceSegment is a transaction trace segment expectation.
type WantTraceSegment = internal.WantTraceSegment

// WantSlowQuery is a slow query expectation.
type WantSlowQuery = internal.WantSlowQuery

// ConnectReply holds settings returned by the New Relic collector.
type ConnectReply = internal.ConnectReply

var (
	// MatchAnything can be used in attribute maps to accept any value.
	MatchAnything = internal.MatchAnything

	// MatchAnyString accepts any string value in attribute comparisons.
	MatchAnyString = internal.MatchAnyString

	// MatchAnyUnixMilli accepts any unix millisecond timestamp in log comparisons.
	MatchAnyUnixMilli = internal.MatchAnyUnixMilli
)

// HarvestTesting sets up an in-memory harvest for use in tests.
var HarvestTesting = internal.HarvestTesting

// NewTraceIDGenerator creates a new deterministic trace ID generator for tests.
var NewTraceIDGenerator = internal.NewTraceIDGenerator
