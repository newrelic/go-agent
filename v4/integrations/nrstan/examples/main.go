// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	stan "github.com/nats-io/stan.go"
	"github.com/newrelic/go-agent/v3/integrations/nrnats"
	"github.com/newrelic/go-agent/v3/integrations/nrstan"
	"github.com/newrelic/go-agent/v3/newrelic"
)

var app *newrelic.Application

func doAsync(sc stan.Conn, txn *newrelic.Transaction) {
	wg := sync.WaitGroup{}
	subj := "async"

	// Simple Async Subscriber
	// Use the nrstan.StreamingSubWrapper to wrap the stan.MsgHandler and
	// create a newrelic.Transaction with each processed stan.Msg
	_, err := sc.Subscribe(subj, nrstan.StreamingSubWrapper(app, func(m *stan.Msg) {
		defer wg.Done()
		fmt.Println("Received async message:", string(m.Data))
	}))
	if nil != err {
		panic(err)
	}

	// Simple Publisher
	wg.Add(1)
	// Use nrnats.StartPublishSegment to create a newrelic.ExternalSegment for
	// the call to sc.Publish
	seg := nrnats.StartPublishSegment(txn, sc.NatsConn(), subj)
	err = sc.Publish(subj, []byte("Hello World"))
	seg.End()
	if nil != err {
		panic(err)
	}

	wg.Wait()
}

func doQueue(sc stan.Conn, txn *newrelic.Transaction) {
	wg := sync.WaitGroup{}
	subj := "queue"

	// Queue Subscriber
	// Use the nrstan.StreamingSubWrapper to wrap the stan.MsgHandler and
	// create a newrelic.Transaction with each processed stan.Msg
	_, err := sc.QueueSubscribe(subj, "myqueue", nrstan.StreamingSubWrapper(app, func(m *stan.Msg) {
		defer wg.Done()
		fmt.Println("Received queue message:", string(m.Data))
	}))
	if nil != err {
		panic(err)
	}

	wg.Add(1)
	// Use nrnats.StartPublishSegment to create a newrelic.ExternalSegment for
	// the call to sc.Publish
	seg := nrnats.StartPublishSegment(txn, sc.NatsConn(), subj)
	err = sc.Publish(subj, []byte("Hello World"))
	seg.End()
	if nil != err {
		panic(err)
	}

	wg.Wait()
}

func main() {
	// Initialize agent
	var err error
	app, err = newrelic.NewApplication(
		newrelic.ConfigAppName("STAN App"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDebugLogger(os.Stdout),
	)
	if nil != err {
		panic(err)
	}
	defer app.Shutdown(10 * time.Second)
	err = app.WaitForConnection(5 * time.Second)
	if nil != err {
		panic(err)
	}
	txn := app.StartTransaction("main")
	defer txn.End()

	// Connect to a server
	sc, err := stan.Connect("test-cluster", "clientid")
	if nil != err {
		panic(err)
	}
	defer sc.Close()

	doAsync(sc, txn)
	doQueue(sc, txn)
}
