// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrnats

import (
	"strings"

	nats "github.com/nats-io/nats.go"
	newrelic "github.com/newrelic/go-agent/v4/newrelic"
)

// StartPublishSegment creates and starts a `newrelic.MessageProducerSegment`
// (https://godoc.org/github.com/newrelic/go-agent#MessageProducerSegment) for NATS
// publishers.  Call this function before calling any method that publishes or
// responds to a NATS message.  Call `End()`
// (https://godoc.org/github.com/newrelic/go-agent#MessageProducerSegment.End) on the
// returned newrelic.MessageProducerSegment when the publish is complete.  The
// `newrelic.Transaction` and `nats.Conn` parameters are required.  The subject
// parameter is the subject of the publish call and is used in metric and span
// names.
func StartPublishSegment(txn *newrelic.Transaction, nc *nats.Conn, subject string) *newrelic.MessageProducerSegment {
	if nil == txn {
		return nil
	}
	if nil == nc {
		return nil
	}
	return &newrelic.MessageProducerSegment{
		StartTime:            txn.StartSegmentNow(),
		Library:              "nats",
		DestinationType:      newrelic.MessageTopic,
		DestinationName:      subject,
		DestinationTemporary: strings.HasPrefix(subject, "_INBOX"),
	}
}

// SubWrapper can be used to wrap the function for nats.Subscribe (https://godoc.org/github.com/nats-io/go-nats#Conn.Subscribe
// or https://godoc.org/github.com/nats-io/go-nats#EncodedConn.Subscribe)
// and nats.QueueSubscribe (https://godoc.org/github.com/nats-io/go-nats#Conn.QueueSubscribe or
// https://godoc.org/github.com/nats-io/go-nats#EncodedConn.QueueSubscribe)
// If the `newrelic.Application` parameter is non-nil, it will create a `newrelic.Transaction` and end the transaction
// when the passed function is complete.
func SubWrapper(app *newrelic.Application, f func(msg *nats.Msg)) func(msg *nats.Msg) {
	if app == nil {
		return f
	}
	return func(msg *nats.Msg) {
		name := msg.Subject + " receive"
		txn := app.StartTransaction(name)
		defer txn.End()

		txn.AddAttribute(newrelic.AttributeMessageRoutingKey, msg.Sub.Subject)
		txn.AddAttribute(newrelic.AttributeMessageQueueName, msg.Sub.Queue)
		txn.AddAttribute(newrelic.AttributeMessageReplyTo, msg.Reply)

		f(msg)
	}
}
