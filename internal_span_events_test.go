package newrelic

import (
	"testing"

	"github.com/newrelic/go-agent/internal"
)

func TestSpanEventSuccess(t *testing.T) {
	// Test that a basic segment creates a span event, and that a
	// transaction has a root span event.
	replyfn := func(reply *internal.ConnectReply) {
		reply.AdaptiveSampler = internal.SampleEverything{}
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
		cfg.CrossApplicationTracer.Enabled = false
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello", nil, nil)
	segment := StartSegment(txn, "mySegment")
	segment.End()
	txn.End()
	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":          "OtherTransaction/Go/hello",
				"sampled":       true,
				"category":      "generic",
				"priority":      internal.MatchAnything,
				"guid":          internal.MatchAnything,
				"transactionId": internal.MatchAnything,
				"nr.entryPoint": true,
				"traceId":       internal.MatchAnything,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"name":          "Custom/mySegment",
				"sampled":       true,
				"category":      "generic",
				"priority":      internal.MatchAnything,
				"guid":          internal.MatchAnything,
				"transactionId": internal.MatchAnything,
				"traceId":       internal.MatchAnything,
				"parentId":      internal.MatchAnything,
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
}

func TestSpanEventsLocallyDisabled(t *testing.T) {
	// Test that span events do not get created if Config.SpanEvents.Enabled
	// is false.
	replyfn := func(reply *internal.ConnectReply) {
		reply.AdaptiveSampler = internal.SampleEverything{}
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
		cfg.CrossApplicationTracer.Enabled = false
		cfg.SpanEvents.Enabled = false
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello", nil, nil)
	segment := StartSegment(txn, "mySegment")
	segment.End()
	txn.End()
	app.ExpectSpanEvents(t, []internal.WantEvent{})
}

func TestSpanEventsRemotelyDisabled(t *testing.T) {
	// Test that span events do not get created if the connect reply
	// disables span events.
	replyfn := func(reply *internal.ConnectReply) {
		reply.AdaptiveSampler = internal.SampleEverything{}
		reply.CollectSpanEvents = false
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = true
		cfg.CrossApplicationTracer.Enabled = false
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello", nil, nil)
	segment := StartSegment(txn, "mySegment")
	segment.End()
	txn.End()
	app.ExpectSpanEvents(t, []internal.WantEvent{})
}

func TestSpanEventsDisabledWithoutDistributedTracing(t *testing.T) {
	// Test that span events do not get created distributed tracing is not
	// enabled.
	replyfn := func(reply *internal.ConnectReply) {
		reply.AdaptiveSampler = internal.SampleEverything{}
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = false
		cfg.CrossApplicationTracer.Enabled = false
	}
	app := testApp(replyfn, cfgfn, t)
	txn := app.StartTransaction("hello", nil, nil)
	segment := StartSegment(txn, "mySegment")
	segment.End()
	txn.End()
	app.ExpectSpanEvents(t, []internal.WantEvent{})
}
