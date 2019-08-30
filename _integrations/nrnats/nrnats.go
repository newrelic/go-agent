package nrnats

import (
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
