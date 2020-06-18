// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrstan

import (
	"os"
	"sync"
	"testing"

	"github.com/nats-io/nats-streaming-server/server"
	"github.com/nats-io/stan.go"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/internal"
	"github.com/newrelic/go-agent/internal/integrationsupport"
)

const (
	clusterName = "my_test_cluster"
	clientName  = "me"
)

func TestMain(m *testing.M) {
	s, err := server.RunServer(clusterName)
	if err != nil {
		panic(err)
	}
	defer s.Shutdown()
	os.Exit(m.Run())
}

func createTestApp() integrationsupport.ExpectApp {
	return integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn, cfgFn)
}

var cfgFn = func(cfg *newrelic.Config) {
	cfg.Enabled = false
	cfg.DistributedTracer.Enabled = true
	cfg.TransactionTracer.SegmentThreshold = 0
	cfg.TransactionTracer.Threshold.IsApdexFailing = false
	cfg.TransactionTracer.Threshold.Duration = 0
	cfg.Attributes.Include = append(cfg.Attributes.Include,
		newrelic.AttributeMessageRoutingKey,
		newrelic.AttributeMessageQueueName,
		newrelic.AttributeMessageExchangeType,
		newrelic.AttributeMessageReplyTo,
		newrelic.AttributeMessageCorrelationID,
	)
}

func TestSubWrapperWithNilApp(t *testing.T) {
	subject := "sample.subject1"
	sc, err := stan.Connect(clusterName, clientName)
	if err != nil {
		t.Fatal("Couldn't connect to server", err)
	}
	defer sc.Close()

	wg := sync.WaitGroup{}
	sc.Subscribe(subject, StreamingSubWrapper(nil, func(msg *stan.Msg) {
		defer wg.Done()
	}))
	wg.Add(1)
	sc.Publish(subject, []byte("data"))
	wg.Wait()
}

func TestSubWrapper(t *testing.T) {
	subject := "sample.subject2"
	sc, err := stan.Connect(clusterName, clientName)
	if err != nil {
		t.Fatal("Couldn't connect to server", err)
	}
	defer sc.Close()

	wg := sync.WaitGroup{}
	app := createTestApp()
	sc.Subscribe(subject, WgWrapper(&wg, StreamingSubWrapper(app, func(msg *stan.Msg) {})))

	wg.Add(1)
	sc.Publish(subject, []byte("data"))
	wg.Wait()

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransaction/Go/Message/STAN/Topic/Named/sample.subject2", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/Message/STAN/Topic/Named/sample.subject2", Scope: "", Forced: false, Data: nil},
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":     "OtherTransaction/Go/Message/STAN/Topic/Named/sample.subject2",
				"guid":     internal.MatchAnything,
				"priority": internal.MatchAnything,
				"sampled":  internal.MatchAnything,
				"traceId":  internal.MatchAnything,
			},
			AgentAttributes: map[string]interface{}{
				"message.routingKey": "sample.subject2",
			},
			UserAttributes: map[string]interface{}{},
		},
	})
}

// Wrapper function to ensure that the NR wrapper is done recording transaction data before wg.Done() is called
func WgWrapper(wg *sync.WaitGroup, nrWrap func(msg *stan.Msg)) func(msg *stan.Msg) {
	return func(msg *stan.Msg) {
		nrWrap(msg)
		wg.Done()
	}
}
