package nrstan

import (
	"fmt"

	"github.com/nats-io/stan.go"
	newrelic "github.com/newrelic/go-agent"
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
		defer app.StartTransaction(subTxnName(msg.Subject), nil, nil).End()
		f(msg)
	}
}

func subTxnName(subject string) string {
	return fmt.Sprintf("Message/stan.go/Topic/%s:subscriber", subject)
}
