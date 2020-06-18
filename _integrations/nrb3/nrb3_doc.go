// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrb3 supports adding B3 headers to outgoing requests.
//
// When using the New Relic Go Agent, use this package if you want to add B3
// headers ("X-B3-TraceId", etc., see
// https://github.com/openzipkin/b3-propagation) to outgoing requests.
//
// Distributed tracing must be enabled
// (https://docs.newrelic.com/docs/understand-dependencies/distributed-tracing/enable-configure/enable-distributed-tracing)
// for B3 headers to be added properly.
package nrb3
