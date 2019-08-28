package nrstan

import (
	"fmt"
	"github.com/nats-io/stan.go"
	newrelic "github.com/newrelic/go-agent"
)

// TODO: more documentation
// Can be used to wrap the function for STREAMING stan.Subscribe  and stan.QueueSubscribe
// (https://godoc.org/github.com/nats-io/stan.go#Conn)
func NrStreamingSubWrapper(app newrelic.Application, f func(msg *stan.Msg)) func(msg *stan.Msg) {
	if app == nil {
		return f
	}
	return func(msg *stan.Msg) {
		txn := app.StartTransaction(subTxnName(msg.Subject), nil, nil)
		defer txn.End()
		f(msg)
	}
}

func subTxnName(subject string) string {
	return fmt.Sprintf("Message/stan.go/Topic/%s:subscriber", subject)
}
