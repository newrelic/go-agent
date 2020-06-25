// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrnats

import (
	"strings"

	nats "github.com/nats-io/nats.go"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/internal"
	"github.com/newrelic/go-agent/internal/integrationsupport"
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
func StartPublishSegment(txn newrelic.Transaction, nc *nats.Conn, subject string) *newrelic.MessageProducerSegment {
	if nil == txn {
		return nil
	}
	if nil == nc {
		return nil
	}
	return &newrelic.MessageProducerSegment{
		StartTime:            newrelic.StartSegmentNow(txn),
		Library:              "NATS",
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
func SubWrapper(app newrelic.Application, f func(msg *nats.Msg)) func(msg *nats.Msg) {
	if app == nil {
		return f
	}
	return func(msg *nats.Msg) {
		namer := internal.MessageMetricKey{
			Library:         "NATS",
			DestinationType: string(newrelic.MessageTopic),
			DestinationName: msg.Subject,
			Consumer:        true,
		}
		txn := app.StartTransaction(namer.Name(), nil, nil)
		defer txn.End()

		integrationsupport.AddAgentAttribute(txn, internal.AttributeMessageRoutingKey, msg.Sub.Subject, nil)
		integrationsupport.AddAgentAttribute(txn, internal.AttributeMessageQueueName, msg.Sub.Queue, nil)
		integrationsupport.AddAgentAttribute(txn, internal.AttributeMessageReplyTo, msg.Reply, nil)

		f(msg)
	}
}
