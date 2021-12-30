// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrawssdk

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/private/protocol/rest"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/awssupport"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)

func testApp() integrationsupport.ExpectApp {
	return integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn, integrationsupport.DTEnabledCfgFn)
}

type fakeTransport struct{}

func (t fakeTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewReader([]byte(""))),
		Header: http.Header{
			"X-Amzn-Requestid": []string{requestID},
		},
	}, nil
}

type fakeCreds struct{}

func (c *fakeCreds) Retrieve() (credentials.Value, error) {
	return credentials.Value{}, nil
}
func (c *fakeCreds) IsExpired() bool { return false }

func newSession() *session.Session {
	r := "us-west-2"
	ses := session.New()
	ses.Config.Credentials = credentials.NewCredentials(&fakeCreds{})
	ses.Config.HTTPClient.Transport = &fakeTransport{}
	ses.Config.Region = &r
	return ses
}

const (
	requestID = "testing request id"
	txnName   = "aws-txn"
)

var (
	genericSpan = internal.WantEvent{
		Intrinsics: map[string]interface{}{
			"name":             "OtherTransaction/Go/" + txnName,
			"transaction.name": "OtherTransaction/Go/" + txnName,
			"sampled":          true,
			"category":         "generic",
			"priority":         internal.MatchAnything,
			"guid":             internal.MatchAnything,
			"transactionId":    internal.MatchAnything,
			"nr.entryPoint":    true,
			"traceId":          internal.MatchAnything,
		},
		UserAttributes:  map[string]interface{}{},
		AgentAttributes: map[string]interface{}{},
	}
	externalSpan = internal.WantEvent{
		Intrinsics: map[string]interface{}{
			"name":          "External/lambda.us-west-2.amazonaws.com/http/POST",
			"sampled":       true,
			"category":      "http",
			"priority":      internal.MatchAnything,
			"guid":          internal.MatchAnything,
			"transactionId": internal.MatchAnything,
			"traceId":       internal.MatchAnything,
			"parentId":      internal.MatchAnything,
			"component":     "http",
			"span.kind":     "client",
		},
		UserAttributes: map[string]interface{}{},
		AgentAttributes: map[string]interface{}{
			"aws.operation": "Invoke",
			"aws.region":    "us-west-2",
			"aws.requestId": requestID,
			"http.method":   "POST",
			"http.url":      "https://lambda.us-west-2.amazonaws.com/2015-03-31/functions/non-existent-function/invocations",
		},
	}
	externalSpanNoRequestID = internal.WantEvent{
		Intrinsics: map[string]interface{}{
			"name":          "External/lambda.us-west-2.amazonaws.com/http/POST",
			"sampled":       true,
			"category":      "http",
			"priority":      internal.MatchAnything,
			"guid":          internal.MatchAnything,
			"transactionId": internal.MatchAnything,
			"traceId":       internal.MatchAnything,
			"parentId":      internal.MatchAnything,
			"component":     "http",
			"span.kind":     "client",
		},
		UserAttributes: map[string]interface{}{},
		AgentAttributes: map[string]interface{}{
			"aws.operation": "Invoke",
			"aws.region":    "us-west-2",
			"http.method":   "POST",
			"http.url":      "https://lambda.us-west-2.amazonaws.com/2015-03-31/functions/non-existent-function/invocations",
		},
	}
	datastoreSpan = internal.WantEvent{
		Intrinsics: map[string]interface{}{
			"name":          "Datastore/statement/DynamoDB/thebesttable/DescribeTable",
			"sampled":       true,
			"category":      "datastore",
			"priority":      internal.MatchAnything,
			"guid":          internal.MatchAnything,
			"transactionId": internal.MatchAnything,
			"traceId":       internal.MatchAnything,
			"parentId":      internal.MatchAnything,
			"component":     "DynamoDB",
			"span.kind":     "client",
		},
		UserAttributes: map[string]interface{}{},
		AgentAttributes: map[string]interface{}{
			"aws.operation": "DescribeTable",
			"aws.region":    "us-west-2",
			"aws.requestId": requestID,
			"db.collection": "thebesttable",
			"db.statement":  "'DescribeTable' on 'thebesttable' using 'DynamoDB'",
			"peer.address":  "dynamodb.us-west-2.amazonaws.com:unknown",
			"peer.hostname": "dynamodb.us-west-2.amazonaws.com",
		},
	}

	txnMetrics = []internal.WantMetric{
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransaction/Go/" + txnName, Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/" + txnName, Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
	}
	externalMetrics = append([]internal.WantMetric{
		{Name: "External/all", Scope: "", Forced: true, Data: nil},
		{Name: "External/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "External/lambda.us-west-2.amazonaws.com/all", Scope: "", Forced: false, Data: nil},
		{Name: "External/lambda.us-west-2.amazonaws.com/http/POST", Scope: "OtherTransaction/Go/" + txnName, Forced: false, Data: nil},
	}, txnMetrics...)
	datastoreMetrics = append([]internal.WantMetric{
		{Name: "Datastore/DynamoDB/all", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/DynamoDB/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/all", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/instance/DynamoDB/dynamodb.us-west-2.amazonaws.com/unknown", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/operation/DynamoDB/DescribeTable", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/statement/DynamoDB/thebesttable/DescribeTable", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/statement/DynamoDB/thebesttable/DescribeTable", Scope: "OtherTransaction/Go/" + txnName, Forced: false, Data: nil},
	}, txnMetrics...)
)

func TestInstrumentRequestExternal(t *testing.T) {
	app := testApp()
	txn := app.StartTransaction(txnName)

	client := lambda.New(newSession())
	input := &lambda.InvokeInput{
		ClientContext:  aws.String("MyApp"),
		FunctionName:   aws.String("non-existent-function"),
		InvocationType: aws.String("Event"),
		LogType:        aws.String("Tail"),
		Payload:        []byte("{}"),
	}

	req, out := client.InvokeRequest(input)
	InstrumentHandlers(&req.Handlers)
	req.HTTPRequest = newrelic.RequestWithTransactionContext(req.HTTPRequest, txn)

	err := req.Send()
	if nil != err {
		t.Error(err)
	}
	if 200 != *out.StatusCode {
		t.Error("wrong status code on response", out.StatusCode)
	}

	txn.End()

	app.ExpectMetrics(t, externalMetrics)
	app.ExpectSpanEvents(t, []internal.WantEvent{
		externalSpan, genericSpan})
}

func TestInstrumentRequestDatastore(t *testing.T) {
	app := testApp()
	txn := app.StartTransaction(txnName)

	client := dynamodb.New(newSession())
	input := &dynamodb.DescribeTableInput{
		TableName: aws.String("thebesttable"),
	}

	req, _ := client.DescribeTableRequest(input)
	InstrumentHandlers(&req.Handlers)
	req.HTTPRequest = newrelic.RequestWithTransactionContext(req.HTTPRequest, txn)

	err := req.Send()
	if nil != err {
		t.Error(err)
	}

	txn.End()

	app.ExpectMetrics(t, datastoreMetrics)
	app.ExpectSpanEvents(t, []internal.WantEvent{
		datastoreSpan, genericSpan})
}

func TestInstrumentRequestExternalNoTxn(t *testing.T) {
	client := lambda.New(newSession())
	input := &lambda.InvokeInput{
		ClientContext:  aws.String("MyApp"),
		FunctionName:   aws.String("non-existent-function"),
		InvocationType: aws.String("Event"),
		LogType:        aws.String("Tail"),
		Payload:        []byte("{}"),
	}

	req, out := client.InvokeRequest(input)
	InstrumentHandlers(&req.Handlers)

	err := req.Send()
	if nil != err {
		t.Error(err)
	}
	if 200 != *out.StatusCode {
		t.Error("wrong status code on response", out.StatusCode)
	}
}

func TestInstrumentRequestDatastoreNoTxn(t *testing.T) {
	client := dynamodb.New(newSession())
	input := &dynamodb.DescribeTableInput{
		TableName: aws.String("thebesttable"),
	}

	req, _ := client.DescribeTableRequest(input)
	InstrumentHandlers(&req.Handlers)

	err := req.Send()
	if nil != err {
		t.Error(err)
	}
}

func TestInstrumentSessionExternal(t *testing.T) {
	app := testApp()
	txn := app.StartTransaction(txnName)

	ses := newSession()
	InstrumentHandlers(&ses.Handlers)
	client := lambda.New(ses)

	input := &lambda.InvokeInput{
		ClientContext:  aws.String("MyApp"),
		FunctionName:   aws.String("non-existent-function"),
		InvocationType: aws.String("Event"),
		LogType:        aws.String("Tail"),
		Payload:        []byte("{}"),
	}

	req, out := client.InvokeRequest(input)
	req.HTTPRequest = newrelic.RequestWithTransactionContext(req.HTTPRequest, txn)

	err := req.Send()
	if nil != err {
		t.Error(err)
	}
	if 200 != *out.StatusCode {
		t.Error("wrong status code on response", out.StatusCode)
	}

	txn.End()

	app.ExpectMetrics(t, externalMetrics)
	app.ExpectSpanEvents(t, []internal.WantEvent{
		externalSpan, genericSpan})
}

func TestInstrumentSessionDatastore(t *testing.T) {
	app := testApp()
	txn := app.StartTransaction(txnName)

	ses := newSession()
	InstrumentHandlers(&ses.Handlers)
	client := dynamodb.New(ses)

	input := &dynamodb.DescribeTableInput{
		TableName: aws.String("thebesttable"),
	}

	req, _ := client.DescribeTableRequest(input)
	req.HTTPRequest = newrelic.RequestWithTransactionContext(req.HTTPRequest, txn)

	err := req.Send()
	if nil != err {
		t.Error(err)
	}

	txn.End()

	app.ExpectMetrics(t, datastoreMetrics)
	app.ExpectSpanEvents(t, []internal.WantEvent{
		datastoreSpan, genericSpan})
}

func TestInstrumentSessionExternalNoTxn(t *testing.T) {
	ses := newSession()
	InstrumentHandlers(&ses.Handlers)
	client := lambda.New(ses)

	input := &lambda.InvokeInput{
		ClientContext:  aws.String("MyApp"),
		FunctionName:   aws.String("non-existent-function"),
		InvocationType: aws.String("Event"),
		LogType:        aws.String("Tail"),
		Payload:        []byte("{}"),
	}

	req, out := client.InvokeRequest(input)
	req.HTTPRequest = newrelic.RequestWithTransactionContext(req.HTTPRequest, nil)

	err := req.Send()
	if nil != err {
		t.Error(err)
	}
	if 200 != *out.StatusCode {
		t.Error("wrong status code on response", out.StatusCode)
	}
}

func TestInstrumentSessionDatastoreNoTxn(t *testing.T) {
	ses := newSession()
	InstrumentHandlers(&ses.Handlers)
	client := dynamodb.New(ses)

	input := &dynamodb.DescribeTableInput{
		TableName: aws.String("thebesttable"),
	}

	req, _ := client.DescribeTableRequest(input)
	req.HTTPRequest = newrelic.RequestWithTransactionContext(req.HTTPRequest, nil)

	err := req.Send()
	if nil != err {
		t.Error(err)
	}
}

func TestInstrumentSessionExternalTxnNotInCtx(t *testing.T) {
	app := testApp()
	txn := app.StartTransaction(txnName)

	ses := newSession()
	InstrumentHandlers(&ses.Handlers)
	client := lambda.New(ses)

	input := &lambda.InvokeInput{
		ClientContext:  aws.String("MyApp"),
		FunctionName:   aws.String("non-existent-function"),
		InvocationType: aws.String("Event"),
		LogType:        aws.String("Tail"),
		Payload:        []byte("{}"),
	}

	req, out := client.InvokeRequest(input)

	err := req.Send()
	if nil != err {
		t.Error(err)
	}
	if 200 != *out.StatusCode {
		t.Error("wrong status code on response", out.StatusCode)
	}

	txn.End()

	app.ExpectMetrics(t, txnMetrics)
}

func TestInstrumentSessionDatastoreTxnNotInCtx(t *testing.T) {
	app := testApp()
	txn := app.StartTransaction(txnName)

	ses := newSession()
	InstrumentHandlers(&ses.Handlers)
	client := dynamodb.New(ses)

	input := &dynamodb.DescribeTableInput{
		TableName: aws.String("thebesttable"),
	}

	req, _ := client.DescribeTableRequest(input)

	err := req.Send()
	if nil != err {
		t.Error(err)
	}

	txn.End()

	app.ExpectMetrics(t, txnMetrics)
}

func TestDoublyInstrumented(t *testing.T) {
	hs := &request.Handlers{}
	if found := hs.Send.Len(); 0 != found {
		t.Error("unexpected number of Send handlers found:", found)
	}

	InstrumentHandlers(hs)
	if found := hs.Send.Len(); 1 != found {
		t.Error("unexpected number of Send handlers found:", found)
	}
	if found := hs.Sign.Len(); 1 != found {
		t.Error("unexpected number of Sign handlers found:", found)
	}

	InstrumentHandlers(hs)
	if found := hs.Send.Len(); 1 != found {
		t.Error("unexpected number of Send handlers found:", found)
	}
	if found := hs.Sign.Len(); 1 != found {
		t.Error("unexpected number of Sign handlers found:", found)
	}
}

type firstFailingTransport struct {
	failing bool
}

func (t *firstFailingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if t.failing {
		t.failing = false
		return nil, errors.New("Oops this failed")
	}
	return &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewReader([]byte(""))),
		Header: http.Header{
			"X-Amzn-Requestid": []string{requestID},
		},
	}, nil
}

func TestRetrySend(t *testing.T) {
	app := testApp()
	txn := app.StartTransaction(txnName)

	ses := newSession()
	ses.Config.HTTPClient.Transport = &firstFailingTransport{failing: true}

	client := lambda.New(ses)
	input := &lambda.InvokeInput{
		ClientContext:  aws.String("MyApp"),
		FunctionName:   aws.String("non-existent-function"),
		InvocationType: aws.String("Event"),
		LogType:        aws.String("Tail"),
		Payload:        []byte("{}"),
	}

	req, out := client.InvokeRequest(input)
	InstrumentHandlers(&req.Handlers)
	req.HTTPRequest = newrelic.RequestWithTransactionContext(req.HTTPRequest, txn)

	err := req.Send()
	if nil != err {
		t.Error(err)
	}
	if 200 != *out.StatusCode {
		t.Error("wrong status code on response", out.StatusCode)
	}

	txn.End()

	app.ExpectMetrics(t, externalMetrics)
	app.ExpectSpanEvents(t, []internal.WantEvent{
		externalSpanNoRequestID, externalSpan, genericSpan})
}

func TestRequestSentTwice(t *testing.T) {
	app := testApp()
	txn := app.StartTransaction(txnName)

	client := lambda.New(newSession())
	input := &lambda.InvokeInput{
		ClientContext:  aws.String("MyApp"),
		FunctionName:   aws.String("non-existent-function"),
		InvocationType: aws.String("Event"),
		LogType:        aws.String("Tail"),
		Payload:        []byte("{}"),
	}

	req, out := client.InvokeRequest(input)
	InstrumentHandlers(&req.Handlers)
	req.HTTPRequest = newrelic.RequestWithTransactionContext(req.HTTPRequest, txn)

	firstErr := req.Send()
	if nil != firstErr {
		t.Error(firstErr)
	}
	if 200 != *out.StatusCode {
		t.Error("wrong status code on response", out.StatusCode)
	}

	secondErr := req.Send()
	if nil != secondErr {
		t.Error(secondErr)
	}
	if 200 != *out.StatusCode {
		t.Error("wrong status code on response", out.StatusCode)
	}

	txn.End()

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "External/all", Scope: "", Forced: true, Data: []float64{2}},
		{Name: "External/allOther", Scope: "", Forced: true, Data: []float64{2}},
		{Name: "External/lambda.us-west-2.amazonaws.com/all", Scope: "", Forced: false, Data: []float64{2}},
		{Name: "External/lambda.us-west-2.amazonaws.com/http/POST", Scope: "OtherTransaction/Go/" + txnName, Forced: false, Data: []float64{2}},
		{Name: "OtherTransaction/Go/" + txnName, Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/" + txnName, Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
	})
	app.ExpectSpanEvents(t, []internal.WantEvent{
		externalSpan, externalSpan, genericSpan})
}

type noRequestIDTransport struct{}

func (t *noRequestIDTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewReader([]byte(""))),
	}, nil
}

func TestNoRequestIDFound(t *testing.T) {
	app := testApp()
	txn := app.StartTransaction(txnName)

	ses := newSession()
	ses.Config.HTTPClient.Transport = &noRequestIDTransport{}

	client := lambda.New(ses)
	input := &lambda.InvokeInput{
		ClientContext:  aws.String("MyApp"),
		FunctionName:   aws.String("non-existent-function"),
		InvocationType: aws.String("Event"),
		LogType:        aws.String("Tail"),
		Payload:        []byte("{}"),
	}

	req, out := client.InvokeRequest(input)
	InstrumentHandlers(&req.Handlers)
	req.HTTPRequest = newrelic.RequestWithTransactionContext(req.HTTPRequest, txn)

	err := req.Send()
	if nil != err {
		t.Error(err)
	}
	if 200 != *out.StatusCode {
		t.Error("wrong status code on response", out.StatusCode)
	}

	txn.End()

	app.ExpectMetrics(t, externalMetrics)
	app.ExpectSpanEvents(t, []internal.WantEvent{
		externalSpanNoRequestID, genericSpan})
}

func TestGetRequestID(t *testing.T) {
	primary := "X-Amzn-Requestid"
	secondary := "X-Amz-Request-Id"

	testcases := []struct {
		hdr      http.Header
		expected string
	}{
		{hdr: http.Header{
			"hello": []string{"world"},
		}, expected: ""},

		{hdr: http.Header{
			strings.ToUpper(primary): []string{"hello"},
		}, expected: ""},

		{hdr: http.Header{
			primary: []string{"hello"},
		}, expected: "hello"},

		{hdr: http.Header{
			secondary: []string{"hello"},
		}, expected: "hello"},

		{hdr: http.Header{
			primary:   []string{"hello"},
			secondary: []string{"world"},
		}, expected: "hello"},

		{hdr: http.Header{}, expected: ""},
	}

	// Make sure our assumptions still hold against aws-sdk-go
	for _, test := range testcases {
		req := &request.Request{
			HTTPResponse: &http.Response{
				Header: test.hdr,
			},
		}
		rest.UnmarshalMeta(req)
		if out := awssupport.GetRequestID(test.hdr); req.RequestID != out {
			t.Error("requestId assumptions incorrect", out, req.RequestID,
				test.hdr, test.expected)
		}
	}
}
