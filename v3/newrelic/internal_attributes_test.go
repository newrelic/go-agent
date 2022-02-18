// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"errors"
	"math"
	"net/http"
	"net/url"
	"testing"

	"github.com/newrelic/go-agent/v3/internal"
)

func TestAddAttributeHighSecurity(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.HighSecurity = true
		cfg.DistributedTracer.Enabled = false
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("hello")

	txn.AddAttribute(`key`, 1)
	app.expectSingleLoggedError(t, "unable to add attribute", map[string]interface{}{
		"reason": errHighSecurityEnabled.Error(),
	})
	txn.End()

	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name": "OtherTransaction/Go/hello",
		},
		AgentAttributes: nil,
		UserAttributes:  map[string]interface{}{},
	}})
}

func TestAddAttributeSecurityPolicyDisablesParameters(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) {
		reply.SecurityPolicies.CustomParameters.SetEnabled(false)
	}
	app := testApp(replyfn, ConfigDistributedTracerEnabled(false), t)
	txn := app.StartTransaction("hello")

	txn.AddAttribute(`key`, 1)
	app.expectSingleLoggedError(t, "unable to add attribute", map[string]interface{}{
		"reason": errSecurityPolicy.Error(),
	})
	txn.End()

	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name": "OtherTransaction/Go/hello",
		},
		AgentAttributes: nil,
		UserAttributes:  map[string]interface{}{},
	}})
}

func TestAddAttributeSecurityPolicyDisablesInclude(t *testing.T) {
	replyfn := func(reply *internal.ConnectReply) {
		reply.SecurityPolicies.AttributesInclude.SetEnabled(false)
	}
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = false
		cfg.TransactionEvents.Attributes.Include = append(cfg.TransactionEvents.Attributes.Include,
			AttributeRequestUserAgent)
	}
	val := "dont-include-me-in-txn-events"
	app := testApp(replyfn, cfgfn, t)
	req := &http.Request{}
	req.Header = make(http.Header)
	req.Header.Add("User-Agent", val)
	txn := app.StartTransaction("hello")
	txn.SetWebRequestHTTP(req)
	txn.NoticeError(errors.New("hello"))
	txn.End()
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             "WebTransaction/Go/hello",
			"nr.apdexPerfZone": "F",
		},
		AgentAttributes: map[string]interface{}{},
		UserAttributes:  map[string]interface{}{},
	}})
	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "WebTransaction/Go/hello",
		Msg:     "hello",
		Klass:   "*errors.errorString",
		AgentAttributes: map[string]interface{}{
			AttributeRequestUserAgent:           val,
			AttributeRequestUserAgentDeprecated: val,
		},
		UserAttributes: map[string]interface{}{},
	}})
}

func TestUserAttributeBasics(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.TransactionTracer.Threshold.IsApdexFailing = false
		cfg.TransactionTracer.Threshold.Duration = 0
		cfg.DistributedTracer.Enabled = false
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("hello")

	txn.NoticeError(errors.New("zap"))
	txn.AddAttribute(`int\key`, 1)
	app.expectNoLoggedErrors(t)
	txn.AddAttribute(`str\key`, `zip\zap`)
	app.expectNoLoggedErrors(t)
	txn.AddAttribute("invalid_value", struct{}{})
	app.expectSingleLoggedError(t, "unable to add attribute", map[string]interface{}{
		"reason": `attribute 'invalid_value' value of type struct {} is invalid`,
	})
	txn.AddAttribute("nil_value", nil)
	app.expectSingleLoggedError(t, "unable to add attribute", map[string]interface{}{
		"reason": `attribute 'nil_value' value of type <nil> is invalid`,
	})
	txn.End()
	txn.AddAttribute("already_ended", "zap")
	app.expectSingleLoggedError(t, "unable to add attribute", map[string]interface{}{
		"reason": errAlreadyEnded.Error(),
	})

	agentAttributes := map[string]interface{}{}
	userAttributes := map[string]interface{}{`int\key`: 1, `str\key`: `zip\zap`}

	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name": "OtherTransaction/Go/hello",
		},
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrors(t, []internal.WantError{{
		TxnName:         "OtherTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "*errors.errorString",
			"error.message":   "zap",
			"transactionName": "OtherTransaction/Go/hello",
		},
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName:      "OtherTransaction/Go/hello",
		NumSegments:     0,
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
}

func TestUserAttributeConfiguration(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.DistributedTracer.Enabled = false
		cfg.TransactionEvents.Attributes.Exclude = []string{"only_errors", "only_txn_traces"}
		cfg.ErrorCollector.Attributes.Exclude = []string{"only_txn_events", "only_txn_traces"}
		cfg.TransactionTracer.Attributes.Exclude = []string{"only_txn_events", "only_errors"}
		cfg.Attributes.Exclude = []string{"completed_excluded"}
		cfg.TransactionTracer.Threshold.IsApdexFailing = false
		cfg.TransactionTracer.Threshold.Duration = 0
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("hello")

	txn.NoticeError(errors.New("zap"))

	txn.AddAttribute("only_errors", 1)
	app.expectNoLoggedErrors(t)
	txn.AddAttribute("only_txn_events", 2)
	app.expectNoLoggedErrors(t)
	txn.AddAttribute("only_txn_traces", 3)
	app.expectNoLoggedErrors(t)
	txn.AddAttribute("completed_excluded", 4)
	app.expectNoLoggedErrors(t)
	txn.End()

	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name": "OtherTransaction/Go/hello",
		},
		AgentAttributes: map[string]interface{}{},
		UserAttributes:  map[string]interface{}{"only_txn_events": 2},
	}})
	app.ExpectErrors(t, []internal.WantError{{
		TxnName:         "OtherTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		AgentAttributes: map[string]interface{}{},
		UserAttributes:  map[string]interface{}{"only_errors": 1},
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "*errors.errorString",
			"error.message":   "zap",
			"transactionName": "OtherTransaction/Go/hello",
		},
		AgentAttributes: map[string]interface{}{},
		UserAttributes:  map[string]interface{}{"only_errors": 1},
	}})
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName:      "OtherTransaction/Go/hello",
		NumSegments:     0,
		AgentAttributes: map[string]interface{}{},
		UserAttributes:  map[string]interface{}{"only_txn_traces": 3},
	}})
}

var (
	// Agent attributes expected in txn events from usualAttributeTestTransaction.
	agent1 = map[string]interface{}{
		AttributeHostDisplayName:        `my\host\display\name`,
		AttributeResponseCode:           `404`,
		AttributeResponseCodeDeprecated: `404`,
		AttributeResponseContentType:    `text/plain; charset=us-ascii`,
		AttributeResponseContentLength:  345,
		AttributeRequestMethod:          "GET",
		AttributeRequestAccept:          "text/plain",
		AttributeRequestContentType:     "text/html; charset=utf-8",
		AttributeRequestContentLength:   753,
		AttributeRequestHost:            "my_domain.com",
		AttributeRequestURI:             "/hello",
	}
	// Agent attributes expected in errors and traces from usualAttributeTestTransaction.
	agent2 = mergeAttributes(agent1, map[string]interface{}{
		AttributeRequestUserAgent:           "Mozilla/5.0",
		AttributeRequestUserAgentDeprecated: "Mozilla/5.0",
		AttributeRequestReferer:             "http://en.wikipedia.org/zip",
	})
	// User attributes expected from usualAttributeTestTransaction.
	user1 = map[string]interface{}{
		"myStr": "hello",
	}
)

func agentAttributeTestcase(t testing.TB, cfgfn func(cfg *Config), e AttributeExpect) {
	app := testApp(nil, func(cfg *Config) {
		cfg.HostDisplayName = `my\host\display\name`
		cfg.TransactionTracer.Threshold.IsApdexFailing = false
		cfg.TransactionTracer.Threshold.Duration = 0
		cfg.DistributedTracer.Enabled = false
		if nil != cfgfn {
			cfgfn(cfg)
		}
	}, t)
	w := newCompatibleResponseRecorder()
	txn := app.StartTransaction("hello")
	rw := txn.SetWebResponse(w)
	txn.SetWebRequestHTTP(helloRequest)
	txn.NoticeError(errors.New("zap"))

	hdr := rw.Header()
	hdr.Set("Content-Type", `text/plain; charset=us-ascii`)
	hdr.Set("Content-Length", `345`)

	rw.WriteHeader(404)
	txn.AddAttribute("myStr", "hello")

	txn.End()

	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             "WebTransaction/Go/hello",
			"nr.apdexPerfZone": "F",
		},
		AgentAttributes: e.TxnEvent.Agent,
		UserAttributes:  e.TxnEvent.User,
	}})
	app.ExpectErrors(t, []internal.WantError{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		AgentAttributes: e.Error.Agent,
		UserAttributes:  e.Error.User,
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "*errors.errorString",
			"error.message":   "zap",
			"transactionName": "WebTransaction/Go/hello",
		},
		AgentAttributes: e.Error.Agent,
		UserAttributes:  e.Error.User,
	}})
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName:      "WebTransaction/Go/hello",
		NumSegments:     0,
		AgentAttributes: e.TxnTrace.Agent,
		UserAttributes:  e.TxnTrace.User,
	}})
}

type UserAgent struct {
	User  map[string]interface{}
	Agent map[string]interface{}
}

type AttributeExpect struct {
	TxnEvent UserAgent
	Error    UserAgent
	TxnTrace UserAgent
}

func TestAgentAttributes(t *testing.T) {
	agentAttributeTestcase(t, nil, AttributeExpect{
		TxnEvent: UserAgent{
			Agent: agent1,
			User:  user1},
		Error: UserAgent{
			Agent: agent2,
			User:  user1},
	})
}

func TestAttributesDisabled(t *testing.T) {
	agentAttributeTestcase(t, func(cfg *Config) {
		cfg.Attributes.Enabled = false
		cfg.DistributedTracer.Enabled = false
	}, AttributeExpect{
		TxnEvent: UserAgent{
			Agent: map[string]interface{}{},
			User:  map[string]interface{}{}},
		Error: UserAgent{
			Agent: map[string]interface{}{},
			User:  map[string]interface{}{}},
		TxnTrace: UserAgent{
			Agent: map[string]interface{}{},
			User:  map[string]interface{}{}},
	})
}

func TestDefaultResponseCode(t *testing.T) {
	app := testApp(nil, ConfigDistributedTracerEnabled(false), t)
	w := newCompatibleResponseRecorder()
	txn := app.StartTransaction("hello")
	rw := txn.SetWebResponse(w)
	txn.SetWebRequestHTTP(&http.Request{})
	rw.Write([]byte("hello"))
	txn.End()

	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             "WebTransaction/Go/hello",
			"nr.apdexPerfZone": "S",
		},
		AgentAttributes: map[string]interface{}{
			AttributeResponseCode:           200,
			AttributeResponseCodeDeprecated: 200,
		},
		UserAttributes: map[string]interface{}{},
	}})
}

func TestNoResponseCode(t *testing.T) {
	app := testApp(nil, ConfigDistributedTracerEnabled(false), t)
	w := newCompatibleResponseRecorder()
	txn := app.StartTransaction("hello")
	txn.SetWebResponse(w)
	txn.SetWebRequestHTTP(&http.Request{})
	txn.End()

	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             "WebTransaction/Go/hello",
			"nr.apdexPerfZone": "S",
		},
		AgentAttributes: map[string]interface{}{},
		UserAttributes:  map[string]interface{}{},
	}})
}

func TestTxnEventAttributesDisabled(t *testing.T) {
	agentAttributeTestcase(t, func(cfg *Config) {
		cfg.TransactionEvents.Attributes.Enabled = false
		cfg.DistributedTracer.Enabled = false
	}, AttributeExpect{
		TxnEvent: UserAgent{
			Agent: map[string]interface{}{},
			User:  map[string]interface{}{}},
		Error: UserAgent{
			Agent: agent2,
			User:  user1},
		TxnTrace: UserAgent{
			Agent: agent2,
			User:  user1},
	})
}

func TestErrorAttributesDisabled(t *testing.T) {
	agentAttributeTestcase(t, func(cfg *Config) {
		cfg.ErrorCollector.Attributes.Enabled = false
		cfg.DistributedTracer.Enabled = false
	}, AttributeExpect{
		TxnEvent: UserAgent{
			Agent: agent1,
			User:  user1},
		Error: UserAgent{
			Agent: map[string]interface{}{},
			User:  map[string]interface{}{}},
		TxnTrace: UserAgent{
			Agent: agent2,
			User:  user1},
	})
}

func TestTxnTraceAttributesDisabled(t *testing.T) {
	agentAttributeTestcase(t, func(cfg *Config) {
		cfg.TransactionTracer.Attributes.Enabled = false
		cfg.DistributedTracer.Enabled = false
	}, AttributeExpect{
		TxnEvent: UserAgent{
			Agent: agent1,
			User:  user1},
		Error: UserAgent{
			Agent: agent2,
			User:  user1},
		TxnTrace: UserAgent{
			Agent: map[string]interface{}{},
			User:  map[string]interface{}{}},
	})
}

var (
	allAgentAttributeNames = []string{
		AttributeResponseCode,
		AttributeResponseCodeDeprecated,
		AttributeRequestMethod,
		AttributeRequestAccept,
		AttributeRequestContentType,
		AttributeRequestContentLength,
		AttributeRequestHost,
		AttributeRequestURI,
		AttributeResponseContentType,
		AttributeResponseContentLength,
		AttributeHostDisplayName,
		AttributeRequestUserAgent,
		AttributeRequestUserAgentDeprecated,
		AttributeRequestReferer,
	}
)

func TestAgentAttributesExcluded(t *testing.T) {
	agentAttributeTestcase(t, func(cfg *Config) {
		cfg.Attributes.Exclude = allAgentAttributeNames
		cfg.DistributedTracer.Enabled = false
	}, AttributeExpect{
		TxnEvent: UserAgent{
			Agent: map[string]interface{}{},
			User:  user1},
		Error: UserAgent{
			Agent: map[string]interface{}{},
			User:  user1},
		TxnTrace: UserAgent{
			Agent: map[string]interface{}{},
			User:  user1},
	})
}

func TestAgentAttributesExcludedFromErrors(t *testing.T) {
	agentAttributeTestcase(t, func(cfg *Config) {
		cfg.ErrorCollector.Attributes.Exclude = allAgentAttributeNames
		cfg.DistributedTracer.Enabled = false
	}, AttributeExpect{
		TxnEvent: UserAgent{
			Agent: agent1,
			User:  user1},
		Error: UserAgent{
			Agent: map[string]interface{}{},
			User:  user1},
		TxnTrace: UserAgent{
			Agent: agent2,
			User:  user1},
	})
}

func TestAgentAttributesExcludedFromTxnEvents(t *testing.T) {
	agentAttributeTestcase(t, func(cfg *Config) {
		cfg.TransactionEvents.Attributes.Exclude = allAgentAttributeNames
		cfg.DistributedTracer.Enabled = false
	}, AttributeExpect{
		TxnEvent: UserAgent{
			Agent: map[string]interface{}{},
			User:  user1},
		Error: UserAgent{
			Agent: agent2,
			User:  user1},
		TxnTrace: UserAgent{
			Agent: agent2,
			User:  user1},
	})
}

func TestAgentAttributesExcludedFromTxnTraces(t *testing.T) {
	agentAttributeTestcase(t, func(cfg *Config) {
		cfg.TransactionTracer.Attributes.Exclude = allAgentAttributeNames
		cfg.DistributedTracer.Enabled = false
	}, AttributeExpect{
		TxnEvent: UserAgent{
			Agent: agent1,
			User:  user1},
		Error: UserAgent{
			Agent: agent2,
			User:  user1},
		TxnTrace: UserAgent{
			Agent: map[string]interface{}{},
			User:  user1},
	})
}

func TestRequestURIPresent(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.TransactionTracer.Threshold.IsApdexFailing = false
		cfg.TransactionTracer.Threshold.Duration = 0
		cfg.DistributedTracer.Enabled = false
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("hello")
	u, err := url.Parse("/hello?remove=me")
	if nil != err {
		t.Error(err)
	}
	txn.SetWebRequest(WebRequest{URL: u})
	txn.NoticeError(errors.New("zap"))
	txn.End()

	agentAttributes := map[string]interface{}{"request.uri": "/hello"}
	userAttributes := map[string]interface{}{}

	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             "WebTransaction/Go/hello",
			"nr.apdexPerfZone": "F",
		},
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrors(t, []internal.WantError{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "*errors.errorString",
			"error.message":   "zap",
			"transactionName": "WebTransaction/Go/hello",
		},
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName:      "WebTransaction/Go/hello",
		NumSegments:     0,
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
}

func TestRequestURIExcluded(t *testing.T) {
	cfgfn := func(cfg *Config) {
		cfg.TransactionTracer.Threshold.IsApdexFailing = false
		cfg.TransactionTracer.Threshold.Duration = 0
		cfg.DistributedTracer.Enabled = false
		cfg.Attributes.Exclude = append(cfg.Attributes.Exclude, AttributeRequestURI)
	}
	app := testApp(nil, cfgfn, t)
	txn := app.StartTransaction("hello")
	u, err := url.Parse("/hello?remove=me")
	if nil != err {
		t.Error(err)
	}
	txn.SetWebRequest(WebRequest{URL: u})
	txn.NoticeError(errors.New("zap"))
	txn.End()

	agentAttributes := map[string]interface{}{}
	userAttributes := map[string]interface{}{}

	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             "WebTransaction/Go/hello",
			"nr.apdexPerfZone": "F",
		},
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrors(t, []internal.WantError{{
		TxnName:         "WebTransaction/Go/hello",
		Msg:             "zap",
		Klass:           "*errors.errorString",
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "*errors.errorString",
			"error.message":   "zap",
			"transactionName": "WebTransaction/Go/hello",
		},
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName:      "WebTransaction/Go/hello",
		NumSegments:     0,
		AgentAttributes: agentAttributes,
		UserAttributes:  userAttributes,
	}})
}

func TestMessageAttributes(t *testing.T) {
	// test that adding message attributes as agent attributes filters them,
	// but as user attributes does not filter them.
	app := testApp(nil, ConfigDistributedTracerEnabled(false), t)

	txn := app.StartTransaction("hello1")
	txn.Private.(internal.AddAgentAttributer).AddAgentAttribute(AttributeMessageRoutingKey, "myRoutingKey", nil)
	txn.Private.(internal.AddAgentAttributer).AddAgentAttribute(AttributeMessageExchangeType, "myExchangeType", nil)
	txn.Private.(internal.AddAgentAttributer).AddAgentAttribute(AttributeMessageCorrelationID, "myCorrelationID", nil)
	txn.Private.(internal.AddAgentAttributer).AddAgentAttribute(AttributeMessageQueueName, "myQueueName", nil)
	txn.Private.(internal.AddAgentAttributer).AddAgentAttribute(AttributeMessageReplyTo, "myReplyTo", nil)
	txn.End()

	txn = app.StartTransaction("hello2")
	txn.AddAttribute(AttributeMessageRoutingKey, "myRoutingKey")
	txn.AddAttribute(AttributeMessageExchangeType, "myExchangeType")
	txn.AddAttribute(AttributeMessageCorrelationID, "myCorrelationID")
	txn.AddAttribute(AttributeMessageQueueName, "myQueueName")
	txn.AddAttribute(AttributeMessageReplyTo, "myReplyTo")
	txn.End()

	app.ExpectTxnEvents(t, []internal.WantEvent{
		{
			UserAttributes: map[string]interface{}{},
			AgentAttributes: map[string]interface{}{
				"message.queueName":  "myQueueName",
				"message.routingKey": "myRoutingKey",
			},
			Intrinsics: map[string]interface{}{
				"name": "OtherTransaction/Go/hello1",
			},
		},
		{
			UserAttributes: map[string]interface{}{
				"message.queueName":     "myQueueName",
				"message.routingKey":    "myRoutingKey",
				"message.exchangeType":  "myExchangeType",
				"message.replyTo":       "myReplyTo",
				"message.correlationId": "myCorrelationID",
			},
			AgentAttributes: map[string]interface{}{},
			Intrinsics: map[string]interface{}{
				"name": "OtherTransaction/Go/hello2",
			},
		},
	})
}

func TestAddSpanAttr_BasicSegment_AllTypes(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("txn")
	sg := txn.StartSegment("SegmentName")
	sg.AddAttribute("attr-string", "this is a string")
	sg.AddAttribute("attr-float-32", float32(1.5))
	sg.AddAttribute("attr-float-64", float64(1.5))
	sg.AddAttribute("attr-int", 2)
	sg.AddAttribute("attr-int-8", int8(3))
	sg.AddAttribute("attr-int-16", int16(4))
	sg.AddAttribute("attr-int-32", int32(5))
	sg.AddAttribute("attr-int-64", int64(6))
	sg.AddAttribute("attr-uint", uint(7))
	sg.AddAttribute("attr-uint-8", uint8(8))
	sg.AddAttribute("attr-uint-16", uint16(9))
	sg.AddAttribute("attr-uint-32", uint32(10))
	sg.AddAttribute("attr-uint-64", uint64(11))
	sg.AddAttribute("attr-uint-ptr", uintptr(12))
	sg.AddAttribute("attr-bool", true)
	sg.End()
	txn.End()

	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":          "Custom/SegmentName",
				"sampled":       true,
				"category":      "generic",
				"priority":      internal.MatchAnything,
				"guid":          "9566c74d10d1e2c6",
				"transactionId": "52fdfc072182654f",
				"traceId":       "52fdfc072182654f163f5f0f9a621d72",
				"parentId":      "4981855ad8681d0d",
			},
			UserAttributes: map[string]interface{}{
				"attr-string":   "this is a string",
				"attr-float-32": 1.5,
				"attr-float-64": 1.5,
				"attr-int":      2,
				"attr-int-8":    3,
				"attr-int-16":   4,
				"attr-int-32":   5,
				"attr-int-64":   6,
				"attr-uint":     7,
				"attr-uint-8":   8,
				"attr-uint-16":  9,
				"attr-uint-32":  10,
				"attr-uint-64":  11,
				"attr-uint-ptr": 12,
				"attr-bool":     true,
			},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"transaction.name": "OtherTransaction/Go/txn",
				"name":             "OtherTransaction/Go/txn",
				"sampled":          true,
				"category":         "generic",
				"priority":         internal.MatchAnything,
				"guid":             "4981855ad8681d0d",
				"transactionId":    "52fdfc072182654f",
				"nr.entryPoint":    true,
				"traceId":          "52fdfc072182654f163f5f0f9a621d72",
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
}

func TestAddSpanAttr_DatastoreSegment(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("txn")
	ds := &DatastoreSegment{
		StartTime:  txn.StartSegmentNow(),
		Product:    "MySQL",
		Collection: "users_table",
		Operation:  "SELECT",
	}
	ds.AddAttribute("attr-string", "this is a string")
	ds.End()
	txn.End()

	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":          "Datastore/statement/MySQL/users_table/SELECT",
				"sampled":       true,
				"category":      "datastore",
				"priority":      internal.MatchAnything,
				"guid":          "9566c74d10d1e2c6",
				"transactionId": "52fdfc072182654f",
				"traceId":       "52fdfc072182654f163f5f0f9a621d72",
				"parentId":      "4981855ad8681d0d",
				"span.kind":     "client",
				"component":     "MySQL",
			},
			UserAttributes: map[string]interface{}{
				"attr-string": "this is a string",
			},
			AgentAttributes: map[string]interface{}{
				"db.statement":  "'SELECT' on 'users_table' using 'MySQL'",
				"db.collection": "users_table",
			},
		},
		{
			Intrinsics: map[string]interface{}{
				"transaction.name": "OtherTransaction/Go/txn",
				"name":             "OtherTransaction/Go/txn",
				"sampled":          true,
				"category":         "generic",
				"priority":         internal.MatchAnything,
				"guid":             "4981855ad8681d0d",
				"transactionId":    "52fdfc072182654f",
				"nr.entryPoint":    true,
				"traceId":          "52fdfc072182654f163f5f0f9a621d72",
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
}

func TestAddSpanAttr_MessageProducerSegment(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("txn")
	seg := &MessageProducerSegment{
		StartTime:       txn.StartSegmentNow(),
		Library:         "RabbitMQ",
		DestinationType: "Exchange",
		DestinationName: "myExchange",
	}
	seg.AddAttribute("attr-string", "this is a string")
	seg.End()
	txn.End()

	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":          "MessageBroker/RabbitMQ/Exchange/Produce/Named/myExchange",
				"sampled":       true,
				"category":      "generic",
				"priority":      internal.MatchAnything,
				"guid":          "9566c74d10d1e2c6",
				"transactionId": "52fdfc072182654f",
				"traceId":       "52fdfc072182654f163f5f0f9a621d72",
				"parentId":      "4981855ad8681d0d",
			},
			UserAttributes: map[string]interface{}{
				"attr-string": "this is a string",
			},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"transaction.name": "OtherTransaction/Go/txn",
				"name":             "OtherTransaction/Go/txn",
				"sampled":          true,
				"category":         "generic",
				"priority":         internal.MatchAnything,
				"guid":             "4981855ad8681d0d",
				"transactionId":    "52fdfc072182654f",
				"nr.entryPoint":    true,
				"traceId":          "52fdfc072182654f163f5f0f9a621d72",
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
}

func TestAddSpanAttr_ExternalSegment(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("txn")
	seg := ExternalSegment{
		StartTime: txn.StartSegmentNow(),
		URL:       "http://www.example.com",
	}
	seg.AddAttribute("attr-string", "this is a string")
	seg.End()
	txn.End()

	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":          "External/www.example.com/http",
				"sampled":       true,
				"category":      "http",
				"priority":      internal.MatchAnything,
				"guid":          "9566c74d10d1e2c6",
				"transactionId": "52fdfc072182654f",
				"traceId":       "52fdfc072182654f163f5f0f9a621d72",
				"parentId":      "4981855ad8681d0d",
				"component":     "http",
				"span.kind":     "client",
			},
			UserAttributes: map[string]interface{}{
				"attr-string": "this is a string",
			},
			AgentAttributes: map[string]interface{}{
				"http.url": "http://www.example.com",
			},
		},
		{
			Intrinsics: map[string]interface{}{
				"transaction.name": "OtherTransaction/Go/txn",
				"name":             "OtherTransaction/Go/txn",
				"sampled":          true,
				"category":         "generic",
				"priority":         internal.MatchAnything,
				"guid":             "4981855ad8681d0d",
				"transactionId":    "52fdfc072182654f",
				"nr.entryPoint":    true,
				"traceId":          "52fdfc072182654f163f5f0f9a621d72",
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
}

func TestAddSpanAttr_SpanEventsDisabled_TxnTracesNoAttrs(t *testing.T) {
	app := testApp(distributedTracingReplyFields, func(c *Config) {
		enableBetterCAT(c)
		c.SpanEvents.Enabled = false
		c.TransactionTracer.Threshold.IsApdexFailing = false
		c.TransactionTracer.Threshold.Duration = 0
	}, t)
	txn := app.StartTransaction("txn")
	sg := txn.StartSegment("SegmentName")
	sg.AddAttribute("attr-string", "this is a string")
	sg.End()
	txn.End()

	app.ExpectSpanEvents(t, nil)
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName:  "OtherTransaction/Go/txn",
		NumSegments: 0,
		// Ensure the custom attrs weren't added to the txn trace
		UserAttributes:  map[string]interface{}{},
		AgentAttributes: map[string]interface{}{},
	}})
}

func TestAddSpanAttr_SpanEventsEnabled_TxnTracesNoAttrs(t *testing.T) {
	app := testApp(distributedTracingReplyFields, func(c *Config) {
		enableBetterCAT(c)
		c.SpanEvents.Enabled = true
		c.TransactionTracer.Threshold.IsApdexFailing = false
		c.TransactionTracer.Threshold.Duration = 0
	}, t)
	txn := app.StartTransaction("txn")
	sg := txn.StartSegment("SegmentName")
	sg.AddAttribute("attr-string", "this is a string")
	sg.End()
	txn.End()

	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":          "Custom/SegmentName",
				"sampled":       true,
				"category":      "generic",
				"priority":      internal.MatchAnything,
				"guid":          "9566c74d10d1e2c6",
				"transactionId": "52fdfc072182654f",
				"traceId":       "52fdfc072182654f163f5f0f9a621d72",
				"parentId":      "4981855ad8681d0d",
			},
			UserAttributes: map[string]interface{}{
				"attr-string": "this is a string",
			},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"transaction.name": "OtherTransaction/Go/txn",
				"name":             "OtherTransaction/Go/txn",
				"sampled":          true,
				"category":         "generic",
				"priority":         internal.MatchAnything,
				"guid":             "4981855ad8681d0d",
				"transactionId":    "52fdfc072182654f",
				"nr.entryPoint":    true,
				"traceId":          "52fdfc072182654f163f5f0f9a621d72",
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
	app.ExpectTxnTraces(t, []internal.WantTxnTrace{{
		MetricName:  "OtherTransaction/Go/txn",
		NumSegments: 0,
		// Ensure the custom attrs weren't added to the txn trace
		UserAttributes:  map[string]interface{}{},
		AgentAttributes: map[string]interface{}{},
	}})
}

func TestAddSpanAttr_NilSegment(t *testing.T) {
	// Ensure no panics with a nil segment
	var sg Segment
	sg.AddAttribute("attr-string", "this is a string")
	sg.End()
}

func TestAddSpanAttr_ValidatedValues(t *testing.T) {
	app := testApp(distributedTracingReplyFields, enableBetterCAT, t)
	txn := app.StartTransaction("txn")
	sg := txn.StartSegment("SegmentName")

	sg.AddAttribute("attr-float-32-inf", float32(math.Inf(1)))
	app.expectSingleLoggedError(t, "unable to add segment attribute", map[string]interface{}{
		"reason": "attribute 'attr-float-32-inf' of type float contains an invalid value: +Inf",
	})
	sg.AddAttribute("attr-float-32-nan", float32(math.NaN()))
	app.expectSingleLoggedError(t, "unable to add segment attribute", map[string]interface{}{
		"reason": "attribute 'attr-float-32-nan' of type float contains an invalid value: NaN",
	})
	sg.AddAttribute("attr-float-64-inf", float64(math.Inf(1)))
	app.expectSingleLoggedError(t, "unable to add segment attribute", map[string]interface{}{
		"reason": "attribute 'attr-float-64-inf' of type float contains an invalid value: +Inf",
	})
	sg.AddAttribute("attr-float-64-nan", float64(math.NaN()))
	app.expectSingleLoggedError(t, "unable to add segment attribute", map[string]interface{}{
		"reason": "attribute 'attr-float-64-nan' of type float contains an invalid value: NaN",
	})

	longString := "012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789001234567890123456789012345678900123456789012345678901234567890abcdefghijklmnopqrstuvwxyz"
	sg.AddAttribute(longString, "some-string")
	app.expectSingleLoggedError(t, "unable to add segment attribute", map[string]interface{}{
		"reason": "attribute key '01234567890123456789012345678901...' exceeds length limit 255",
	})

	sg.AddAttribute("attr-struct", struct{ name string }{name: "invalid struct value"})
	app.expectSingleLoggedError(t, "unable to add segment attribute", map[string]interface{}{
		"reason": "attribute 'attr-struct' value of type struct { name string } is invalid",
	})

	sg.AddAttribute("attr-with-long-value", longString)
	app.expectNoLoggedErrors(t)

	sg.End()
	txn.End()

	app.ExpectSpanEvents(t, []internal.WantEvent{
		{
			Intrinsics: map[string]interface{}{
				"name":          "Custom/SegmentName",
				"sampled":       true,
				"category":      "generic",
				"priority":      internal.MatchAnything,
				"guid":          "9566c74d10d1e2c6",
				"transactionId": "52fdfc072182654f",
				"traceId":       "52fdfc072182654f163f5f0f9a621d72",
				"parentId":      "4981855ad8681d0d",
			},
			// Only the truncated long value should make it through validation.
			UserAttributes: map[string]interface{}{
				"attr-with-long-value": "012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789001234567890123",
			},
			AgentAttributes: map[string]interface{}{},
		},
		{
			Intrinsics: map[string]interface{}{
				"transaction.name": "OtherTransaction/Go/txn",
				"name":             "OtherTransaction/Go/txn",
				"sampled":          true,
				"category":         "generic",
				"priority":         internal.MatchAnything,
				"guid":             "4981855ad8681d0d",
				"transactionId":    "52fdfc072182654f",
				"nr.entryPoint":    true,
				"traceId":          "52fdfc072182654f163f5f0f9a621d72",
			},
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
		},
	})
}
