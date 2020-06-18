// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/_integrations/nrnats"
)

var app newrelic.Application

func doAsync(nc *nats.Conn, txn newrelic.Transaction) {
	wg := sync.WaitGroup{}
	subj := "async"

	// Simple Async Subscriber
	// Use the nrnats.SubWrapper to wrap the nats.MsgHandler and create a
	// newrelic.Transaction with each processed nats.Msg
	_, err := nc.Subscribe(subj, nrnats.SubWrapper(app, func(m *nats.Msg) {
		defer wg.Done()
		fmt.Println("Received async message:", string(m.Data))
	}))
	if nil != err {
		panic(err)
	}

	// Simple Publisher
	wg.Add(1)
	// Use nrnats.StartPublishSegment to create a
	// newrelic.MessageProducerSegment for the call to nc.Publish
	seg := nrnats.StartPublishSegment(txn, nc, subj)
	err = nc.Publish(subj, []byte("Hello World"))
	seg.End()
	if nil != err {
		panic(err)
	}

	wg.Wait()
}

func doQueue(nc *nats.Conn, txn newrelic.Transaction) {
	wg := sync.WaitGroup{}
	subj := "queue"

	// Queue Subscriber
	// Use the nrnats.SubWrapper to wrap the nats.MsgHandler and create a
	// newrelic.Transaction with each processed nats.Msg
	_, err := nc.QueueSubscribe(subj, "myQueueName", nrnats.SubWrapper(app, func(m *nats.Msg) {
		defer wg.Done()
		fmt.Println("Received queue message:", string(m.Data))
	}))
	if nil != err {
		panic(err)
	}

	wg.Add(1)
	// Use nrnats.StartPublishSegment to create a
	// newrelic.MessageProducerSegment for the call to nc.Publish
	seg := nrnats.StartPublishSegment(txn, nc, subj)
	err = nc.Publish(subj, []byte("Hello World"))
	seg.End()
	if nil != err {
		panic(err)
	}

	wg.Wait()
}

func doSync(nc *nats.Conn, txn newrelic.Transaction) {
	subj := "sync"

	// Simple Sync Subscriber
	sub, err := nc.SubscribeSync(subj)
	if nil != err {
		panic(err)
	}
	// Use nrnats.StartPublishSegment to create a
	// newrelic.MessageProducerSegment for the call to nc.Publish
	seg := nrnats.StartPublishSegment(txn, nc, subj)
	err = nc.Publish(subj, []byte("Hello World"))
	seg.End()
	if nil != err {
		panic(err)
	}
	m, err := sub.NextMsg(time.Second)
	if nil != err {
		panic(err)
	}
	fmt.Println("Received sync message:", string(m.Data))
}

func doChan(nc *nats.Conn, txn newrelic.Transaction) {
	subj := "chan"

	// Channel Subscriber
	ch := make(chan *nats.Msg)
	_, err := nc.ChanSubscribe(subj, ch)
	if nil != err {
		panic(err)
	}

	// Use nrnats.StartPublishSegment to create a
	// newrelic.MessageProducerSegment for the call to nc.Publish
	seg := nrnats.StartPublishSegment(txn, nc, subj)
	err = nc.Publish(subj, []byte("Hello World"))
	seg.End()
	if nil != err {
		panic(err)
	}

	m := <-ch
	fmt.Println("Received chan message:", string(m.Data))
}

func doReply(nc *nats.Conn, txn newrelic.Transaction) {
	subj := "reply"

	// Replies
	nc.Subscribe(subj, func(m *nats.Msg) {
		// Use nrnats.StartPublishSegment to create a
		// newrelic.MessageProducerSegment for the call to nc.Publish
		seg := nrnats.StartPublishSegment(txn, nc, m.Reply)
		nc.Publish(m.Reply, []byte("Hello World"))
		seg.End()
	})

	// Requests
	// Use nrnats.StartPublishSegment to create a
	// newrelic.MessageProducerSegment for the call to nc.Request
	seg := nrnats.StartPublishSegment(txn, nc, subj)
	m, err := nc.Request(subj, []byte("request"), time.Second)
	seg.End()
	if nil != err {
		panic(err)
	}
	fmt.Println("Received reply message:", string(m.Data))
}

func doRespond(nc *nats.Conn, txn newrelic.Transaction) {
	subj := "respond"
	// Respond
	nc.Subscribe(subj, func(m *nats.Msg) {
		// Use nrnats.StartPublishSegment to create a
		// newrelic.MessageProducerSegment for the call to m.Respond
		seg := nrnats.StartPublishSegment(txn, nc, m.Reply)
		m.Respond([]byte("Hello World"))
		seg.End()
	})

	// Requests
	// Use nrnats.StartPublishSegment to create a
	// newrelic.MessageProducerSegment for the call to nc.Request
	seg := nrnats.StartPublishSegment(txn, nc, subj)
	m, err := nc.Request(subj, []byte("request"), time.Second)
	seg.End()
	if nil != err {
		panic(err)
	}
	fmt.Println("Received respond message:", string(m.Data))
}

func mustGetEnv(key string) string {
	if val := os.Getenv(key); "" != val {
		return val
	}
	panic(fmt.Sprintf("environment variable %s unset", key))
}

func main() {
	// Initialize agent
	cfg := newrelic.NewConfig("NATS App", mustGetEnv("NEW_RELIC_LICENSE_KEY"))
	cfg.Logger = newrelic.NewDebugLogger(os.Stdout)
	var err error
	app, err = newrelic.NewApplication(cfg)
	if nil != err {
		panic(err)
	}
	defer app.Shutdown(10 * time.Second)
	err = app.WaitForConnection(5 * time.Second)
	if nil != err {
		panic(err)
	}
	txn := app.StartTransaction("main", nil, nil)
	defer txn.End()

	// Connect to a server
	nc, err := nats.Connect(nats.DefaultURL)
	if nil != err {
		panic(err)
	}
	defer nc.Drain()

	doAsync(nc, txn)
	doQueue(nc, txn)
	doSync(nc, txn)
	doChan(nc, txn)
	doReply(nc, txn)
	doRespond(nc, txn)
}
