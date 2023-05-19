// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrnats

import (
	"github.com/nats-io/nats-server/test"
	nats "github.com/nats-io/nats.go"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
	"os"
	"sync"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	s := test.RunDefaultServer()
	defer s.Shutdown()
	os.Exit(m.Run())
}

func testApp() integrationsupport.ExpectApp {
	return integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn, integrationsupport.ConfigFullTraces, cfgFn)
}

var cfgFn = func(cfg *newrelic.Config) {
	cfg.Attributes.Include = append(cfg.Attributes.Include,
		newrelic.AttributeMessageRoutingKey,
		newrelic.AttributeMessageQueueName,
		newrelic.AttributeMessageExchangeType,
		newrelic.AttributeMessageReplyTo,
		newrelic.AttributeMessageCorrelationID,
	)
}

func TestStartPublishSegmentNilTxn(t *testing.T) {
	// Make sure that a nil transaction does not cause panics
	nc, err := nats.Connect(nats.DefaultURL)
	if nil != err {
		t.Fatal(err)
	}
	defer nc.Close()

	StartPublishSegment(nil, nc, "mysubject").End()
}

func TestStartPublishSegmentNilConn(t *testing.T) {
	// Make sure that a nil nats.Conn does not cause panics and does not record
	// metrics
	app := testApp()
	txn := app.StartTransaction("testing")
	StartPublishSegment(txn, nil, "mysubject").End()
	txn.End()

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransaction/Go/testing", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/testing", Scope: "", Forced: false, Data: nil},
	})
}

func TestStartPublishSegmentBasic(t *testing.T) {
	app := testApp()
	txn := app.StartTransaction("testing")
	nc, err := nats.Connect(nats.DefaultURL)
	if nil != err {
		t.Fatal(err)
	}
	defer nc.Close()

	StartPublishSegment(txn, nc, "mysubject").End()
	txn.End()

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "MessageBroker/NATS/Topic/Produce/Named/mysubject", Scope: "", Forced: false, Data: nil},
		{Name: "MessageBroker/NATS/Topic/Produce/Named/mysubject", Scope: "OtherTransaction/Go/testing", Forced: false, Data: nil},
		{Name: "OtherTransaction/Go/testing", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/testing", Scope: "", Forced: false, Data: nil},
	})
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"category": "generic",
				"name":     "MessageBroker/NATS/Topic/Produce/Named/mysubject",
				"parentId": internal.MatchAnything,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"category":         "generic",
				"name":             "OtherTransaction/Go/testing",
				"transaction.name": "OtherTransaction/Go/testing",
				"nr.entryPoint":    true,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName: "OtherTransaction/Go/testing",
		Root: internal.WantTraceSegment{
			SegmentName: "ROOT",
			Attributes:  map[string]interface{}{},
			Children: []internal.WantTraceSegment{{
				SegmentName: "OtherTransaction/Go/testing",
				Attributes:  map[string]interface{}{"exclusive_duration_millis": internal.MatchAnything},
				Children: []internal.WantTraceSegment{
					{
						SegmentName: "MessageBroker/NATS/Topic/Produce/Named/mysubject",
						Attributes:  map[string]interface{}{},
					},
				},
			}},
		},
	},
	})
}

func TestSubWrapperWithNilApp(t *testing.T) {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		t.Fatal("Error connecting to NATS server", err)
	}
	wg := sync.WaitGroup{}
	nc.Subscribe("subject1", SubWrapper(nil, func(msg *nats.Msg) {
		wg.Done()
	}))
	wg.Add(1)
	nc.Publish("subject1", []byte("data"))
	wg.Wait()
}

func TestSubWrapper(t *testing.T) {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		t.Fatal("Error connecting to NATS server", err)
	}
	wg := sync.WaitGroup{}
	app := testApp()
	nc.QueueSubscribe("subject2", "queue1", WgWrapper(&wg, SubWrapper(app.Application, func(msg *nats.Msg) {})))
	wg.Add(1)
	nc.Request("subject2", []byte("data"), time.Second)
	wg.Wait()

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransaction/Go/Message/NATS/Topic/Named/subject2", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/Message/NATS/Topic/Named/subject2", Scope: "", Forced: false, Data: nil},
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":     "OtherTransaction/Go/Message/NATS/Topic/Named/subject2",
				"guid":     internal.MatchAnything,
				"priority": internal.MatchAnything,
				"sampled":  internal.MatchAnything,
				"traceId":  internal.MatchAnything,
			},
			AgentAttributes: map[string]interface{}{
				"message.replyTo":    internal.MatchAnything, // starts with _INBOX
				"message.routingKey": "subject2",
				"message.queueName":  "queue1",
			},
			UserAttributes: map[string]interface{}{},
		},
	})
}

func TestStartPublishSegmentNaming(t *testing.T) {
	testCases := []struct {
		subject string
		metric  string
	}{
		{subject: "", metric: "MessageBroker/NATS/Topic/Produce/Named/Unknown"},
		{subject: "mysubject", metric: "MessageBroker/NATS/Topic/Produce/Named/mysubject"},
		{subject: "_INBOX.asldfkjsldfjskd.ldskfjls", metric: "MessageBroker/NATS/Topic/Produce/Temp"},
	}

	nc, err := nats.Connect(nats.DefaultURL)
	if nil != err {
		t.Fatal(err)
	}
	defer nc.Close()

	for _, tc := range testCases {
		app := testApp()
		txn := app.StartTransaction("testing")
		StartPublishSegment(txn, nc, tc.subject).End()
		txn.End()

		app.ExpectMetrics(t, []internal.WantMetric{
			{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
			{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
			{Name: "OtherTransaction/Go/testing", Scope: "", Forced: true, Data: nil},
			{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
			{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
			{Name: "OtherTransactionTotalTime/Go/testing", Scope: "", Forced: false, Data: nil},
			{Name: tc.metric, Scope: "", Forced: false, Data: nil},
			{Name: tc.metric, Scope: "OtherTransaction/Go/testing", Forced: false, Data: nil},
		})
	}
}

// Wrapper function to ensure that the NR wrapper is done recording transaction data before wg.Done() is called
func WgWrapper(wg *sync.WaitGroup, nrWrap func(msg *nats.Msg)) func(msg *nats.Msg) {
	return func(msg *nats.Msg) {
		nrWrap(msg)
		wg.Done()
	}
}
