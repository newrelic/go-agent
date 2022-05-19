// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import "time"

const (
	// app behavior

	// fixedHarvestPeriod is the period that fixed period data (metrics,
	// traces, and span events) is sent to New Relic.
	fixedHarvestPeriod = 60 * time.Second
	// collectorTimeout is the timeout used in the client for communication
	// with New Relic's servers.
	collectorTimeout = 20 * time.Second
	// appDataChanSize is the size of the channel that contains data sent
	// the app processor.
	appDataChanSize           = 200
	failedMetricAttemptsLimit = 5
	failedEventsAttemptsLimit = 10

	// transaction behavior
	maxStackTraceFrames = 100
	// maxTxnErrors is the maximum number of errors captured per
	// transaction.
	maxTxnErrors      = 5
	maxTxnSlowQueries = 10

	startingTxnTraceNodes = 16
	maxTxnTraceNodes      = 256

	// harvest data
	maxMetrics          = 2 * 1000
	maxRegularTraces    = 1
	maxSyntheticsTraces = 20
	maxHarvestErrors    = 20
	maxHarvestSlowSQLs  = 10
	// maxSpanEvents is the maximum number of Span Events that can be captured
	// per 60-second harvest cycle
	// DEPRECATED: replaced with DistributedTracer.ReservoirLimit configuration value
	// This constant is the default we start that value as, but it can be changed at runtime.
	// always find the dynamic value, e.g. run.MaxSpanEvents(), instead of this value.
	defaultMaxSpanEvents = 2000

	// attributes
	attributeKeyLengthLimit   = 255
	attributeValueLengthLimit = 255
	attributeUserLimit        = 64
	// attributeErrorLimit limits the number of extra attributes that can be
	// provided when noticing an error.
	attributeErrorLimit       = 32
	customEventAttributeLimit = 64

	// Limits affecting Config validation are found in the config package.

	// runtimeSamplerPeriod is the period of the runtime sampler.  Runtime
	// metrics should not depend on the sampler period, but the period must
	// be the same across instances.  For that reason, this value should not
	// be changed without notifying customers that they must update all
	// instance simultaneously for valid runtime metrics.
	runtimeSamplerPeriod = 60 * time.Second
)
