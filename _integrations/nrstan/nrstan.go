// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrstan

import (
	stan "github.com/nats-io/stan.go"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/internal"
	"github.com/newrelic/go-agent/internal/integrationsupport"
)

// StreamingSubWrapper can be used to wrap the function for STREAMING stan.Subscribe and stan.QueueSubscribe
// (https://godoc.org/github.com/nats-io/stan.go#Conn)
// If the `newrelic.Application` parameter is non-nil, it will create a `newrelic.Transaction` and end the transaction
// when the passed function is complete.
func StreamingSubWrapper(app newrelic.Application, f func(msg *stan.Msg)) func(msg *stan.Msg) {
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
		txn := app.StartTransaction(namer.Name(), nil, nil)
		defer txn.End()

		integrationsupport.AddAgentAttribute(txn, internal.AttributeMessageRoutingKey, msg.MsgProto.Subject, nil)
		integrationsupport.AddAgentAttribute(txn, internal.AttributeMessageReplyTo, msg.MsgProto.Reply, nil)

		f(msg)
	}
}
