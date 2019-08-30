package nrnats

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
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
