package nrstan

import (
	"testing"
	"time"

	"github.com/nats-io/nats-streaming-server/server"
	"github.com/nats-io/stan.go"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/internal"
)

const (
	clusterName = "my_test_cluster"
	clientName  = "me"
	subject     = "sample.subject"
)

func subFunc(_ *stan.Msg) {}

func createTestApp(t *testing.T) newrelic.Application {
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
		reply.AccountID = "123"
		reply.TrustedAccountKey = "123"
		reply.PrimaryAppID = "456"
	}
	internal.HarvestTesting(app, replyfn)
	return app
}

func TestNrSubWrapper(t *testing.T) {
	s, err := server.RunServer(clusterName)
	if err != nil {
		panic(err)
	}
	defer s.Shutdown()
	sc, err := stan.Connect(clusterName, clientName)
	if err != nil {
		t.Fatal("Couldn't connect to server", err)
	}
	defer sc.Close()

	app := createTestApp(t)
	sc.Subscribe(subject, StreamingSubWrapper(app, subFunc))
	sc.Publish(subject, []byte("data"))

	time.Sleep(100 * time.Millisecond)

	app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransaction/Go/Message/stan.go/Topic/sample.subject:subscriber", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/Message/stan.go/Topic/sample.subject:subscriber", Scope: "", Forced: false, Data: nil},
	})
	app.(internal.Expect).ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName: "OtherTransaction/Go/Message/stan.go/Topic/sample.subject:subscriber",
		Root: internal.WantTraceSegment{
			SegmentName: "ROOT",
			Attributes:  map[string]interface{}{},
			Children: []internal.WantTraceSegment{{
				SegmentName: "OtherTransaction/Go/Message/stan.go/Topic/sample.subject:subscriber",
				Attributes:  map[string]interface{}{"exclusive_duration_millis": internal.MatchAnything},
			}},
		},
	}})
}
