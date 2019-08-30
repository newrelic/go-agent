package nrnats

import (
	"os"
	"testing"

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
		{Name: "External/all", Scope: "", Forced: true, Data: nil},
		{Name: "External/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "External/127.0.0.1:4222/all", Scope: "", Forced: false, Data: nil},
		{Name: "External/127.0.0.1:4222/NATS/Publish/mysubject", Scope: "OtherTransaction/Go/testing", Forced: false, Data: nil},
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
				"category":  "http",
				"component": "NATS",
				"name":      "External/127.0.0.1:4222/NATS/Publish/mysubject",
				"parentId":  internal.MatchAnything,
				"span.kind": "client",
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
						SegmentName: "External/127.0.0.1:4222/NATS/Publish/mysubject",
						Attributes:  map[string]interface{}{},
					},
				},
			}},
		},
	}})
}

func TestStartPublishSegmentProcedure(t *testing.T) {
	testCases := []struct {
		subject   string
		procedure string
	}{
		{subject: "", procedure: "Publish"},
		{subject: "mysubject", procedure: "Publish/mysubject"},
		{subject: "_INBOX.asldfkjsldfjskd.ldskfjls", procedure: "Publish/_INBOX"},
	}

	app := testApp(t)
	txn := app.StartTransaction("testing", nil, nil)
	nc, err := nats.Connect(nats.DefaultURL)
	if nil != err {
		t.Fatal(err)
	}
	defer nc.Close()

	for _, tc := range testCases {
		seg := StartPublishSegment(txn, nc, tc.subject)
		if seg.Procedure != tc.procedure {
			t.Errorf("incorrect Procedure:\nactual=%s\nexpected=%s", seg.Procedure, tc.procedure)
		}
	}
}
