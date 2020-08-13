// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrawssdk

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/private/protocol/rest"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/newrelic/go-agent/v4/internal"
	"github.com/newrelic/go-agent/v4/internal/awssupport"
	"github.com/newrelic/go-agent/v4/internal/integrationsupport"
	newrelic "github.com/newrelic/go-agent/v4/newrelic"
)

func testApp() integrationsupport.ExpectApp {
	return integrationsupport.NewTestApp(integrationsupport.DTEnabledCfgFn)
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

type fakeCredsWithoutContext struct{}

func (c fakeCredsWithoutContext) Retrieve() (aws.Credentials, error) {
	return aws.Credentials{}, nil
}

type fakeCredsWithContext struct{}

func (c fakeCredsWithContext) Retrieve(ctx context.Context) (aws.Credentials, error) {
	return aws.Credentials{}, nil
}

var fakeCreds = func() interface{} {
	var c interface{} = fakeCredsWithoutContext{}
	if _, ok := c.(aws.CredentialsProvider); ok {
		return c
	}
	return fakeCredsWithContext{}
}()

func newConfig(instrument bool) aws.Config {
	cfg, _ := external.LoadDefaultAWSConfig()
	cfg.Credentials = fakeCreds.(aws.CredentialsProvider)
	cfg.Region = "us-west-2"
	cfg.HTTPClient = &http.Client{
		Transport: &fakeTransport{},
	}

	if instrument {
		InstrumentHandlers(&cfg.Handlers)
	}
	return cfg
}

const (
	requestID = "testing request id"
	txnName   = "aws-txn"
)

var (
	genericSpan = internal.WantSpan{
		Name:          txnName,
		ParentID:      internal.MatchNoParent,
		SkipAttrsTest: true,
		Attributes: map[string]interface{}{
			"transaction.name": "OtherTransaction/Go/" + txnName,
			"sampled":          true,
			"category":         "generic",
			"priority":         internal.MatchAnything,
			"guid":             internal.MatchAnything,
			"transactionId":    internal.MatchAnything,
			"nr.entryPoint":    true,
			"traceId":          internal.MatchAnything,
		},
	}
	externalSpan = internal.WantSpan{
		Name:          "http POST lambda.us-west-2.amazonaws.com",
		ParentID:      internal.MatchAnyParent,
		SkipAttrsTest: true,
		Attributes: map[string]interface{}{
			"sampled":       true,
			"category":      "http",
			"priority":      internal.MatchAnything,
			"guid":          internal.MatchAnything,
			"transactionId": internal.MatchAnything,
			"traceId":       internal.MatchAnything,
			"parentId":      internal.MatchAnything,
			"component":     "http",
			"span.kind":     "client",
			"aws.operation": "Invoke",
			"aws.region":    "us-west-2",
			"aws.requestId": requestID,
			"http.method":   "POST",
			"http.url":      "https://lambda.us-west-2.amazonaws.com/2015-03-31/functions/non-existent-function/invocations",
		},
	}
	externalSpanNoRequestID = internal.WantSpan{
		Name:          "http POST lambda.us-west-2.amazonaws.com",
		ParentID:      internal.MatchAnyParent,
		SkipAttrsTest: true,
		Attributes: map[string]interface{}{
			"sampled":       true,
			"category":      "http",
			"priority":      internal.MatchAnything,
			"guid":          internal.MatchAnything,
			"transactionId": internal.MatchAnything,
			"traceId":       internal.MatchAnything,
			"parentId":      internal.MatchAnything,
			"component":     "http",
			"span.kind":     "client",
			"aws.operation": "Invoke",
			"aws.region":    "us-west-2",
			"http.method":   "POST",
			"http.url":      "https://lambda.us-west-2.amazonaws.com/2015-03-31/functions/non-existent-function/invocations",
		},
	}
	datastoreSpan = internal.WantSpan{
		Name:          "'DescribeTable' on 'thebesttable' using 'dynamodb'",
		ParentID:      internal.MatchAnyParent,
		SkipAttrsTest: true,
		Attributes: map[string]interface{}{
			"sampled":       true,
			"category":      "datastore",
			"priority":      internal.MatchAnything,
			"guid":          internal.MatchAnything,
			"transactionId": internal.MatchAnything,
			"traceId":       internal.MatchAnything,
			"parentId":      internal.MatchAnything,
			"component":     "dynamodb",
			"span.kind":     "client",
			"aws.operation": "DescribeTable",
			"aws.region":    "us-west-2",
			"aws.requestId": requestID,
			"db.collection": "thebesttable",
			"db.statement":  "'DescribeTable' on 'thebesttable' using 'dynamodb'",
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
	externalMetrics = append(txnMetrics, []internal.WantMetric{
		{Name: "External/all", Scope: "", Forced: true, Data: nil},
		{Name: "External/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "External/lambda.us-west-2.amazonaws.com/all", Scope: "", Forced: false, Data: nil},
		{Name: "External/lambda.us-west-2.amazonaws.com/http/POST", Scope: "OtherTransaction/Go/" + txnName, Forced: false, Data: nil},
	}...)
	datastoreMetrics = append(txnMetrics, []internal.WantMetric{
		{Name: "Datastore/dynamodb/all", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/dynamodb/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/all", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/instance/dynamodb/dynamodb.us-west-2.amazonaws.com/unknown", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/operation/dynamodb/DescribeTable", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/statement/dynamodb/thebesttable/DescribeTable", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/statement/dynamodb/thebesttable/DescribeTable", Scope: "OtherTransaction/Go/" + txnName, Forced: false, Data: nil},
	}...)
)

func TestInstrumentRequestExternal(t *testing.T) {
	app := testApp()
	txn := app.StartTransaction(txnName)

	client := lambda.New(newConfig(false))
	input := &lambda.InvokeInput{
		ClientContext:  aws.String("MyApp"),
		FunctionName:   aws.String("non-existent-function"),
		InvocationType: lambda.InvocationTypeEvent,
		LogType:        lambda.LogTypeTail,
		Payload:        []byte("{}"),
	}
	req := client.InvokeRequest(input)
	InstrumentHandlers(&req.Handlers)
	ctx := newrelic.NewContext(req.Context(), txn)

	_, err := req.Send(ctx)
	if nil != err {
		t.Error(err)
	}

	txn.End()

	app.ExpectMetrics(t, externalMetrics)
	app.ExpectSpanEvents(t, []internal.WantSpan{
		externalSpan, genericSpan})
}

func TestInstrumentRequestDatastore(t *testing.T) {
	app := testApp()
	txn := app.StartTransaction(txnName)

	client := dynamodb.New(newConfig(false))
	input := &dynamodb.DescribeTableInput{
		TableName: aws.String("thebesttable"),
	}

	req := client.DescribeTableRequest(input)
	InstrumentHandlers(&req.Handlers)
	ctx := newrelic.NewContext(req.Context(), txn)

	_, err := req.Send(ctx)
	if nil != err {
		t.Error(err)
	}

	txn.End()

	app.ExpectMetrics(t, datastoreMetrics)
	app.ExpectSpanEvents(t, []internal.WantSpan{
		datastoreSpan, genericSpan})
}

func TestInstrumentRequestExternalNoTxn(t *testing.T) {
	client := lambda.New(newConfig(false))
	input := &lambda.InvokeInput{
		ClientContext:  aws.String("MyApp"),
		FunctionName:   aws.String("non-existent-function"),
		InvocationType: lambda.InvocationTypeEvent,
		LogType:        lambda.LogTypeTail,
		Payload:        []byte("{}"),
	}

	req := client.InvokeRequest(input)
	InstrumentHandlers(&req.Handlers)
	ctx := req.Context()

	_, err := req.Send(ctx)
	if nil != err {
		t.Error(err)
	}
}

func TestInstrumentRequestDatastoreNoTxn(t *testing.T) {
	client := dynamodb.New(newConfig(false))
	input := &dynamodb.DescribeTableInput{
		TableName: aws.String("thebesttable"),
	}

	req := client.DescribeTableRequest(input)
	InstrumentHandlers(&req.Handlers)
	ctx := req.Context()

	_, err := req.Send(ctx)
	if nil != err {
		t.Error(err)
	}
}

func TestInstrumentConfigExternal(t *testing.T) {
	app := testApp()
	txn := app.StartTransaction(txnName)

	client := lambda.New(newConfig(true))

	input := &lambda.InvokeInput{
		ClientContext:  aws.String("MyApp"),
		FunctionName:   aws.String("non-existent-function"),
		InvocationType: lambda.InvocationTypeEvent,
		LogType:        lambda.LogTypeTail,
		Payload:        []byte("{}"),
	}

	req := client.InvokeRequest(input)
	ctx := newrelic.NewContext(req.Context(), txn)

	_, err := req.Send(ctx)
	if nil != err {
		t.Error(err)
	}

	txn.End()

	app.ExpectMetrics(t, externalMetrics)
	app.ExpectSpanEvents(t, []internal.WantSpan{
		externalSpan, genericSpan})
}

func TestInstrumentConfigDatastore(t *testing.T) {
	app := testApp()
	txn := app.StartTransaction(txnName)

	client := dynamodb.New(newConfig(true))

	input := &dynamodb.DescribeTableInput{
		TableName: aws.String("thebesttable"),
	}

	req := client.DescribeTableRequest(input)
	ctx := newrelic.NewContext(req.Context(), txn)

	_, err := req.Send(ctx)
	if nil != err {
		t.Error(err)
	}

	txn.End()

	app.ExpectMetrics(t, datastoreMetrics)
	app.ExpectSpanEvents(t, []internal.WantSpan{
		datastoreSpan, genericSpan})
}

func TestInstrumentConfigExternalNoTxn(t *testing.T) {
	client := lambda.New(newConfig(true))

	input := &lambda.InvokeInput{
		ClientContext:  aws.String("MyApp"),
		FunctionName:   aws.String("non-existent-function"),
		InvocationType: lambda.InvocationTypeEvent,
		LogType:        lambda.LogTypeTail,
		Payload:        []byte("{}"),
	}

	req := client.InvokeRequest(input)
	ctx := req.Context()

	_, err := req.Send(ctx)
	if nil != err {
		t.Error(err)
	}
}

func TestInstrumentConfigDatastoreNoTxn(t *testing.T) {
	client := dynamodb.New(newConfig(true))

	input := &dynamodb.DescribeTableInput{
		TableName: aws.String("thebesttable"),
	}

	req := client.DescribeTableRequest(input)
	ctx := req.Context()

	_, err := req.Send(ctx)
	if nil != err {
		t.Error(err)
	}
}

func TestInstrumentConfigExternalTxnNotInCtx(t *testing.T) {
	app := testApp()
	txn := app.StartTransaction(txnName)

	client := lambda.New(newConfig(true))

	input := &lambda.InvokeInput{
		ClientContext:  aws.String("MyApp"),
		FunctionName:   aws.String("non-existent-function"),
		InvocationType: lambda.InvocationTypeEvent,
		LogType:        lambda.LogTypeTail,
		Payload:        []byte("{}"),
	}

	req := client.InvokeRequest(input)
	ctx := req.Context()

	_, err := req.Send(ctx)
	if nil != err {
		t.Error(err)
	}

	txn.End()

	app.ExpectMetrics(t, txnMetrics)
}

func TestInstrumentConfigDatastoreTxnNotInCtx(t *testing.T) {
	app := testApp()
	txn := app.StartTransaction(txnName)

	client := dynamodb.New(newConfig(true))

	input := &dynamodb.DescribeTableInput{
		TableName: aws.String("thebesttable"),
	}

	req := client.DescribeTableRequest(input)
	ctx := req.Context()

	_, err := req.Send(ctx)
	if nil != err {
		t.Error(err)
	}

	txn.End()

	app.ExpectMetrics(t, txnMetrics)
}

func TestDoublyInstrumented(t *testing.T) {
	hs := &aws.Handlers{}
	if found := hs.Send.Len(); 0 != found {
		t.Error("unexpected number of Send handlers found:", found)
	}

	InstrumentHandlers(hs)
	if found := hs.Send.Len(); 2 != found {
		t.Error("unexpected number of Send handlers found:", found)
	}

	InstrumentHandlers(hs)
	if found := hs.Send.Len(); 2 != found {
		t.Error("unexpected number of Send handlers found:", found)
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

	cfg := newConfig(false)
	cfg.HTTPClient = &http.Client{
		Transport: &firstFailingTransport{failing: true},
	}

	client := lambda.New(cfg)
	input := &lambda.InvokeInput{
		ClientContext:  aws.String("MyApp"),
		FunctionName:   aws.String("non-existent-function"),
		InvocationType: lambda.InvocationTypeEvent,
		LogType:        lambda.LogTypeTail,
		Payload:        []byte("{}"),
	}
	req := client.InvokeRequest(input)
	InstrumentHandlers(&req.Handlers)
	ctx := newrelic.NewContext(req.Context(), txn)

	_, err := req.Send(ctx)
	if nil != err {
		t.Error(err)
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
	app.ExpectSpanEvents(t, []internal.WantSpan{
		externalSpanNoRequestID, externalSpan, genericSpan})
}

func TestRequestSentTwice(t *testing.T) {
	app := testApp()
	txn := app.StartTransaction(txnName)

	client := lambda.New(newConfig(false))
	input := &lambda.InvokeInput{
		ClientContext:  aws.String("MyApp"),
		FunctionName:   aws.String("non-existent-function"),
		InvocationType: lambda.InvocationTypeEvent,
		LogType:        lambda.LogTypeTail,
		Payload:        []byte("{}"),
	}
	req := client.InvokeRequest(input)
	InstrumentHandlers(&req.Handlers)
	ctx := newrelic.NewContext(req.Context(), txn)

	_, firstErr := req.Send(ctx)
	if nil != firstErr {
		t.Error(firstErr)
	}

	_, secondErr := req.Send(ctx)
	if nil != secondErr {
		t.Error(secondErr)
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
	app.ExpectSpanEvents(t, []internal.WantSpan{
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

	cfg := newConfig(false)
	cfg.HTTPClient = &http.Client{
		Transport: &noRequestIDTransport{},
	}

	client := lambda.New(cfg)
	input := &lambda.InvokeInput{
		ClientContext:  aws.String("MyApp"),
		FunctionName:   aws.String("non-existent-function"),
		InvocationType: lambda.InvocationTypeEvent,
		LogType:        lambda.LogTypeTail,
		Payload:        []byte("{}"),
	}
	req := client.InvokeRequest(input)
	InstrumentHandlers(&req.Handlers)
	ctx := newrelic.NewContext(req.Context(), txn)

	_, err := req.Send(ctx)
	if nil != err {
		t.Error(err)
	}

	txn.End()

	app.ExpectMetrics(t, externalMetrics)
	app.ExpectSpanEvents(t, []internal.WantSpan{
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
	}

	// Make sure our assumptions still hold against aws-sdk-go-v2
	for _, test := range testcases {
		req := &aws.Request{
			HTTPResponse: &http.Response{
				Header: test.hdr,
			},
			Data: &lambda.InvokeOutput{},
		}
		rest.UnmarshalMeta(req)
		if out := awssupport.GetRequestID(test.hdr); req.RequestID != out {
			t.Error("requestId assumptions incorrect", out, req.RequestID,
				test.hdr, test.expected)
		}
	}
}
