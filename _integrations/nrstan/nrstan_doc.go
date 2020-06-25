// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrstan instruments https://github.com/nats-io/stan.go.
//
// This package can be used to simplify instrumenting NATS Streaming subscribers. Currently due to the nature of
// the NATS Streaming framework we are limited to two integration points: `StartPublishSegment` for publishers, and
// `SubWrapper` for subscribers.
//
//
// NATS Streaming subscribers
//
// `nrstan.StreamingSubWrapper` can be used to wrap the function for STREAMING stan.Subscribe and stan.QueueSubscribe
// (https://godoc.org/github.com/nats-io/stan.go#Conn) If the `newrelic.Application` parameter is non-nil, it will
// create a `newrelic.Transaction` and end the transaction when the passed function is complete. Example:
//
//	sc, err := stan.Connect(clusterName, clientName)
//	if err != nil {
// 		t.Fatal("Couldn't connect to server", err)
//	}
//	defer sc.Close()
//	app := createTestApp(t)  // newrelic.Application
//	sc.Subscribe(subject, StreamingSubWrapper(app, myMessageHandler)
//
//
// NATS Streaming publishers
//
// You can use `nrnats.StartPublishSegment` from the `nrnats` package
// (https://godoc.org/github.com/newrelic/go-agent/_integrations/nrnats/#StartPublishSegment)
// to start an external segment when doing a streaming publish, which must be ended after publishing is complete.
// Example:
//
//	sc, err := stan.Connect(clusterName, clientName)
//	if err != nil {
//		t.Fatal("Couldn't connect to server", err)
//	}
//	txn := currentTransaction()  // current newrelic.Transaction
//	seg := nrnats.StartPublishSegment(txn, sc.NatsConn(), subj)
//	sc.Publish(subj, []byte("Hello World"))
//	seg.End()
//
// Full Publisher/Subscriber example:
// https://github.com/newrelic/go-agent/blob/master/_integrations/nrstan/examples/main.go
package nrstan

import "github.com/newrelic/go-agent/internal"

func init() { internal.TrackUsage("integration", "framework", "stan") }
