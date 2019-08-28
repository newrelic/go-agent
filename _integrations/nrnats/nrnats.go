package nrnats

import (
	"fmt"
	"strings"

	"github.com/nats-io/nats.go"
	newrelic "github.com/newrelic/go-agent"
)

// StartPublishSegment creates and starts a `newrelic.ExternalSegment`
// (https://godoc.org/github.com/newrelic/go-agent#ExternalSegment) for NATS
// publishers.  Call this function before calling any method that publishes or
// responds to a NATS message.  Call `End()`
// (https://godoc.org/github.com/newrelic/go-agent#ExternalSegment.End) on the
// returned newrelic.ExternalSegment when the publish is complete.  The
// `newrelic.Transaction` and `nats.Conn` parameters are required.  The subject
// parameter is the subject of the publish call and is used in metric and span
// names.
func StartPublishSegment(txn newrelic.Transaction, nc *nats.Conn, subject string) *newrelic.ExternalSegment {
	if nil == txn || nil == nc {
		return &newrelic.ExternalSegment{}
	}

	var proc string
	if strings.HasPrefix(subject, "_INBOX") {
		proc = "Publish/_INBOX"
	} else if subject != "" {
		proc = "Publish/" + subject
	} else {
		proc = "Publish"
	}

	return &newrelic.ExternalSegment{
		StartTime: newrelic.StartSegmentNow(txn),
		URL:       nc.ConnectedUrl(),
		Procedure: proc,
		Library:   "NATS",
	}
}

// TODO: more documentation
// Can be used to wrap the function for nats.Subscribe (https://godoc.org/github.com/nats-io/go-nats#Conn.Subscribe or
// https://godoc.org/github.com/nats-io/go-nats#EncodedConn.Subscribe)
// and nats.QueueSubscribe (https://godoc.org/github.com/nats-io/go-nats#Conn.QueueSubscribe or
// https://godoc.org/github.com/nats-io/go-nats#EncodedConn.QueueSubscribe)
func NrSubWrapper(app newrelic.Application, f func(msg *nats.Msg)) func(msg *nats.Msg) {
	if app == nil {
		return f
	}
	return func(msg *nats.Msg) {
		txn := app.StartTransaction(subTxnName(msg.Subject), nil, nil)
		defer txn.End()
		f(msg)
	}
}

func subTxnName(subject string) string {
	return fmt.Sprintf("Message/NATS/Topic/%s:subscriber", subject)

}
