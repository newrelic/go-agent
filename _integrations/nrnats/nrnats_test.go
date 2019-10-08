package nrnats

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats-server/test"
	"github.com/nats-io/nats.go"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/internal"
)

func TestMain(m *testing.M) {
	s := test.RunDefaultServer()
	defer s.Shutdown()
	os.Exit(m.Run())
}

func testApp(t *testing.T) newrelic.Application {
	cfg := newrelic.NewConfig("appname", "0123456789012345678901234567890123456789")
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
	app, err := newrelic.NewApplication(cfg)
	if nil != err {
		t.Fatal(err)
	}
	replyfn := func(reply *internal.ConnectReply) {
		reply.AdaptiveSampler = internal.SampleEverything{}
	}
	internal.HarvestTesting(app, replyfn)
	return app
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
	app := testApp(t)
	txn := app.StartTransaction("testing", nil, nil)
	StartPublishSegment(txn, nil, "mysubject").End()
	txn.End()

	app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransaction/Go/testing", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/testing", Scope: "", Forced: false, Data: nil},
	})
}

func TestStartPublishSegmentBasic(t *testing.T) {
	app := testApp(t)
	txn := app.StartTransaction("testing", nil, nil)
	nc, err := nats.Connect(nats.DefaultURL)
	if nil != err {
		t.Fatal(err)
	}
	defer nc.Close()

	StartPublishSegment(txn, nc, "mysubject").End()
	txn.End()

	app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "MessageBroker/NATS/Topic/Produce/Named/mysubject", Scope: "", Forced: false, Data: nil},
		{Name: "MessageBroker/NATS/Topic/Produce/Named/mysubject", Scope: "OtherTransaction/Go/testing", Forced: false, Data: nil},
		{Name: "OtherTransaction/Go/testing", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/testing", Scope: "", Forced: false, Data: nil},
	})
	app.(internal.Expect).ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"category":      "generic",
				"name":          "OtherTransaction/Go/testing",
				"nr.entryPoint": true,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"category": "generic",
				"name":     "MessageBroker/NATS/Topic/Produce/Named/mysubject",
				"parentId": internal.MatchAnything,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
	app.(internal.Expect).ExpectTxnTraces(t, []internal.WantTxnTrace{{
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
	app := testApp(t)
	nc.QueueSubscribe("subject2", "queue1", WgWrapper(&wg, SubWrapper(app, func(msg *nats.Msg) {})))
	wg.Add(1)
	nc.Request("subject2", []byte("data"), time.Second)
	wg.Wait()

	app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransaction/Go/Message/NATS/Topic/Named/subject2", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/Message/NATS/Topic/Named/subject2", Scope: "", Forced: false, Data: nil},
	})
	app.(internal.Expect).ExpectTxnEvents(t, []internal.WantEvent{
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
		app := testApp(t)
		txn := app.StartTransaction("testing", nil, nil)
		StartPublishSegment(txn, nc, tc.subject).End()
		txn.End()

		app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
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
