// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrstan

import (
	stan "github.com/nats-io/stan.go"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)

// StreamingSubWrapper can be used to wrap the function for STREAMING stan.Subscribe and stan.QueueSubscribe
// (https://godoc.org/github.com/nats-io/stan.go#Conn)
// If the `newrelic.Application` parameter is non-nil, it will create a `newrelic.Transaction` and end the transaction
// when the passed function is complete.
func StreamingSubWrapper(app *newrelic.Application, f func(msg *stan.Msg)) func(msg *stan.Msg) {
	if app == nil {
		return f
	}
	return func(msg *stan.Msg) {
		namer := internal.MessageMetricKey{
			Library:         "STAN",
			DestinationType: string(newrelic.MessageTopic),
			DestinationName: msg.MsgProto.Subject,
			Consumer:        true,
		}
		txn := app.StartTransaction(namer.Name())
		defer txn.End()

		integrationsupport.AddAgentAttribute(txn, newrelic.AttributeMessageRoutingKey, msg.MsgProto.Subject, nil)
		integrationsupport.AddAgentAttribute(txn, newrelic.AttributeMessageReplyTo, msg.MsgProto.Reply, nil)

		f(msg)
	}
}
