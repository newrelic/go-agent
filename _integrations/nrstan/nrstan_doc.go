// Package nrstan instruments https://github.com/nats-io/stan.go.
//
// This package can be used to simplify instrumenting STAN subscribers.
//
//
// STAN subscribers
//
// `nrstan.StreamingSubWrapper` can be used to wrap the function for STREAMING `stan.Subscribe` and
// `stan.QueueSubscribe`. If the `newrelic.Application` parameter is non-nil, it will create a `newrelic.Transaction`
// and end the transaction when the passed function is complete. Example:
//
// sc, err := stan.Connect(clusterName, clientName)
// if err != nil {
// 	t.Fatal("Couldn't connect to server", err)
// }
// defer sc.Close()
// app := createTestApp(t)  // newrelic.Application
// sc.Subscribe(subject, StreamingSubWrapper(app, myMessageHandler)
//
//
// STAN publishers
//
// You can use `nrnats.StartPublishSegment` from the `nrnats` package to start an external segment when doing a streaming
// publish, which must be ended after publishing is complete.  Example:
//
// sc, err := stan.Connect(clusterName, clientName)
// if err != nil {
// 	t.Fatal("Couldn't connect to server", err)
// }
// txn := currentTransaction()  // current newrelic.Transaction
// seg := nrnats.StartPublishSegment(txn, sc.NatsConn(), subj)
// _ = sc.Publish(subj, []byte("Hello World"))
// seg.End()
//
// Full Publisher/Subscriber example:
// https://github.com/newrelic/go-agent/go-agent/blob/master/_integrations/nrstan/examples/main.go
package nrstan

import "github.com/newrelic/go-agent/internal"

func init() { internal.TrackUsage("integration", "framework", "stan") }
