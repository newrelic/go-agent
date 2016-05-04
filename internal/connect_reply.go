package internal

import (
	"strings"
	"time"

	"go.datanerd.us/p/will/go-sdk/log"
)

type AgentRunID string

func (id AgentRunID) String() string {
	return string(id)
}

type ConnectReply struct {
	RunID AgentRunID `json:"agent_run_id"`

	// Transaction Name Modifiers
	SegmentTerms SegmentRules `json:"transaction_segment_terms"`
	TxnNameRules MetricRules  `json:"transaction_name_rules"`
	URLRules     MetricRules  `json:"url_rules"`
	MetricRules  MetricRules  `json:"metric_name_rules"`

	// Cross Process
	EncodingKey     string `json:"encoding_key"`
	CrossProcessID  string `json:"cross_process_id"`
	TrustedAccounts []int  `json:"trusted_account_ids"`

	// Settings
	KeyTxnApdex            map[string]float64 `json:"web_transactions_apdex"`
	ApdexThresholdSeconds  float64            `json:"apdex_t"`
	CollectAnalyticsEvents bool               `json:"collect_analytics_events"`
	CollectCustomEvents    bool               `json:"collect_custom_events"`
	CollectTraces          bool               `json:"collect_traces"`
	CollectErrors          bool               `json:"collect_errors"`
	CollectErrorEvents     bool               `json:"collect_error_events"`

	// RUM
	AgentLoader string `json:"js_agent_loader"`
	Beacon      string `json:"beacon"`
	BrowserKey  string `json:"browser_key"`
	AppID       string `json:"application_id"`
	ErrorBeacon string `json:"error_beacon"`
	JSAgentFile string `json:"js_agent_file"`
}

func ConnectReplyDefaults() *ConnectReply {
	// TODO: Compare these values to other agents.
	return &ConnectReply{
		ApdexThresholdSeconds:  0.5,
		CollectAnalyticsEvents: true,
		CollectCustomEvents:    true,
		CollectTraces:          true,
		CollectErrors:          true,
		CollectErrorEvents:     true,
	}
}

func calculateApdexThreshold(c *ConnectReply, txnName string) time.Duration {
	if t, ok := c.KeyTxnApdex[txnName]; ok {
		return floatSecondsToDuration(t)
	}
	return floatSecondsToDuration(c.ApdexThresholdSeconds)
}

// TODO: Where does this belong?
func CreateFullTxnName(input string, reply *ConnectReply, isWeb bool) string {
	var afterURLRules string
	if "" != input {
		afterURLRules = reply.URLRules.Apply(input)
		if "" == afterURLRules {
			log.Debug("transaction ignored by url rules", log.Context{
				"input": input,
			})
			return ""
		}
	}

	prefix := backgroundMetricPrefix
	if isWeb {
		prefix = webMetricPrefix
	}

	var beforeNameRules string
	if strings.HasPrefix(afterURLRules, "/") {
		beforeNameRules = prefix + afterURLRules
	} else {
		beforeNameRules = prefix + "/" + afterURLRules
	}

	afterNameRules := reply.TxnNameRules.Apply(beforeNameRules)
	if "" == afterNameRules {
		log.Debug("transaction ignored by txn name rules", log.Context{
			"input": beforeNameRules,
		})
		return ""
	}

	return reply.SegmentTerms.Apply(afterNameRules)
}
