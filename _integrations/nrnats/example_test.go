// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrnats

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
	newrelic "github.com/newrelic/go-agent"
)

func currentTransaction() newrelic.Transaction { return nil }

func ExampleStartPublishSegment() {
	nc, _ := nats.Connect(nats.DefaultURL)
	txn := currentTransaction()
	subject := "testing.subject"

	// Start the Publish segment
	seg := StartPublishSegment(txn, nc, subject)
	err := nc.Publish(subject, []byte("Hello World"))
	if nil != err {
		panic(err)
	}
	// Manually end the segment
	seg.End()
}

func ExampleStartPublishSegment_defer() {
	nc, _ := nats.Connect(nats.DefaultURL)
	txn := currentTransaction()
	subject := "testing.subject"

	// Start the Publish segment and defer End till the func returns
	defer StartPublishSegment(txn, nc, subject).End()
	m, err := nc.Request(subject, []byte("request"), time.Second)
	if nil != err {
		panic(err)
	}
	fmt.Println("Received reply message:", string(m.Data))
}

var clusterID, clientID string

// StartPublishSegment can be used with a NATS Streamming Connection as well
// (https://github.com/nats-io/stan.go).  Use the `NatsConn()` method on the
// `stan.Conn` interface (https://godoc.org/github.com/nats-io/stan#Conn) to
// access the `nats.Conn` object.
func ExampleStartPublishSegment_stan() {
	sc, _ := stan.Connect(clusterID, clientID)
	txn := currentTransaction()
	subject := "testing.subject"

	defer StartPublishSegment(txn, sc.NatsConn(), subject).End()
	sc.Publish(subject, []byte("Hello World"))
}
