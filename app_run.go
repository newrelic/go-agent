package newrelic

import (
	"net/http"
	"time"

	"github.com/newrelic/go-agent/internal"
)

// appRun contains information regarding a single connection session with the
// collector.  It is immutable after creation at application connect.
type appRun struct {
	Reply *internal.ConnectReply

	// AttributeConfig is calculated on every connect since it depends on
	// the security policies.
	AttributeConfig *internal.AttributeConfig
	Config          Config
}

func newAppRun(config Config, reply *internal.ConnectReply) *appRun {
	return &appRun{
		Reply: reply,
		AttributeConfig: internal.CreateAttributeConfig(internal.AttributeConfigInput{
			Attributes:        convertAttributeDestinationConfig(config.Attributes),
			ErrorCollector:    convertAttributeDestinationConfig(config.ErrorCollector.Attributes),
			TransactionEvents: convertAttributeDestinationConfig(config.TransactionEvents.Attributes),
			TransactionTracer: convertAttributeDestinationConfig(config.TransactionTracer.Attributes),
			BrowserMonitoring: convertAttributeDestinationConfig(config.BrowserMonitoring.Attributes),
			SpanEvents:        convertAttributeDestinationConfig(config.SpanEvents.Attributes),
			TraceSegments:     convertAttributeDestinationConfig(config.TransactionTracer.Segments.Attributes),
		}, reply.SecurityPolicies.AttributesInclude.Enabled()),
		Config: config,
	}
}

const (
	// https://source.datanerd.us/agents/agent-specs/blob/master/Lambda.md#distributed-tracing
	serverlessDefaultPrimaryAppID = "Unknown"
)

const (
	// https://source.datanerd.us/agents/agent-specs/blob/master/Lambda.md#adaptive-sampling
	serverlessSamplerPeriod = 60 * time.Second
	serverlessSamplerTarget = 10
)

func newServerlessConnectReply(config Config) *internal.ConnectReply {
	reply := internal.ConnectReplyDefaults()

	reply.ApdexThresholdSeconds = config.ServerlessMode.ApdexThreshold.Seconds()

	reply.AccountID = config.ServerlessMode.AccountID
	reply.TrustedAccountKey = config.ServerlessMode.TrustedAccountKey
	reply.PrimaryAppID = config.ServerlessMode.PrimaryAppID

	if "" == reply.TrustedAccountKey {
		// The trust key does not need to be provided by customers whose
		// account ID is the same as the trust key.
		reply.TrustedAccountKey = reply.AccountID
	}

	if "" == reply.PrimaryAppID {
		reply.PrimaryAppID = serverlessDefaultPrimaryAppID
	}

	reply.AdaptiveSampler = internal.NewAdaptiveSampler(serverlessSamplerPeriod,
		serverlessSamplerTarget, time.Now())

	return reply
}

func (run *appRun) slowQueriesEnabled() bool {
	return run.Config.DatastoreTracer.SlowQuery.Enabled &&
		run.Reply.CollectTraces
}

func (run *appRun) txnTracesEnabled() bool {
	return run.Config.TransactionTracer.Enabled &&
		run.Reply.CollectTraces
}

func (run *appRun) txnEventsEnabled() bool {
	return run.Config.TransactionEvents.Enabled &&
		run.Reply.CollectAnalyticsEvents
}

func (run *appRun) errorEventsEnabled() bool {
	return run.Config.ErrorCollector.CaptureEvents &&
		run.Reply.CollectErrorEvents
}

func (run *appRun) crossApplicationTracingEnabled() bool {
	// Distributed tracing takes priority over cross-app-tracing per:
	// https://source.datanerd.us/agents/agent-specs/blob/master/Distributed-Tracing.md#distributed-trace-payload
	return run.Config.CrossApplicationTracer.Enabled &&
		!run.Config.DistributedTracer.Enabled
}

func (run *appRun) responseCodeIsError(code int) bool {
	if code < http.StatusBadRequest { // 400
		return false
	}
	for _, ignoreCode := range run.Config.ErrorCollector.IgnoreStatusCodes {
		if code == ignoreCode {
			return false
		}
	}
	return true
}
