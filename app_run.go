package newrelic

import (
	"time"

	"github.com/newrelic/go-agent/internal"
)

// appRun contains information regarding a single connection session with the
// collector.  It is immutable after creation at application connect.
type appRun struct {
	*internal.ConnectReply

	// AttributeConfig is calculated on every connect since it depends on
	// the security policies.
	AttributeConfig *internal.AttributeConfig
}

func newAppRun(config Config, reply *internal.ConnectReply) *appRun {
	return &appRun{
		ConnectReply: reply,
		AttributeConfig: internal.CreateAttributeConfig(internal.AttributeConfigInput{
			Attributes:        convertAttributeDestinationConfig(config.Attributes),
			ErrorCollector:    convertAttributeDestinationConfig(config.ErrorCollector.Attributes),
			TransactionEvents: convertAttributeDestinationConfig(config.TransactionEvents.Attributes),
			TransactionTracer: convertAttributeDestinationConfig(config.TransactionTracer.Attributes),
			BrowserMonitoring: convertAttributeDestinationConfig(config.BrowserMonitoring.Attributes),
			SpanEvents:        convertAttributeDestinationConfig(config.SpanEvents.Attributes),
			TraceSegments:     convertAttributeDestinationConfig(config.TransactionTracer.Segments.Attributes),
		}, reply.SecurityPolicies.AttributesInclude.Enabled()),
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

	if "" == reply.PrimaryAppID {
		reply.PrimaryAppID = serverlessDefaultPrimaryAppID
	}

	reply.AdaptiveSampler = internal.NewAdaptiveSampler(serverlessSamplerPeriod,
		serverlessSamplerTarget, time.Now())

	return reply
}
