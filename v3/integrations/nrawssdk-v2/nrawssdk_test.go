// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrawssdk

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	"github.com/newrelic/go-agent/v3/newrelic"
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

func newConfig(ctx context.Context, txn *newrelic.Transaction) aws.Config {
	cfg, _ := config.LoadDefaultConfig(ctx)
	cfg.Credentials = fakeCreds.(aws.CredentialsProvider)
	cfg.Region = awsRegion
	cfg.HTTPClient = &http.Client{
		Transport: &fakeTransport{},
	}

	AppendMiddlewares(&cfg.APIOptions, txn)

	return cfg
}

const (
	requestID = "testing request id"
	txnName   = "aws-txn"
	awsRegion = "us-west-2"
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
			"aws.operation":   "Invoke",
			"aws.region":      awsRegion,
			"aws.requestId":   requestID,
			"http.method":     "POST",
			"http.url":        "https://lambda.us-west-2.amazonaws.com/2015-03-31/functions/non-existent-function/invocations",
			"http.statusCode": "200",
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
			"aws.operation":   "Invoke",
			"aws.region":      awsRegion,
			"http.method":     "POST",
			"http.url":        "https://lambda.us-west-2.amazonaws.com/2015-03-31/functions/non-existent-function/invocations",
			"http.statusCode": "200",
		},
	}
	datastoreSpan = internal.WantEvent{
		Intrinsics: map[string]interface{}{
			"name":          "Datastore/operation/DynamoDB/DescribeTable",
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
			"aws.operation":   "DescribeTable",
			"aws.region":      awsRegion,
			"aws.requestId":   requestID,
			"db.statement":    "'DescribeTable' on 'unknown' using 'DynamoDB'",
			"peer.address":    "dynamodb.us-west-2.amazonaws.com:unknown",
			"peer.hostname":   "dynamodb.us-west-2.amazonaws.com",
			"http.statusCode": "200",
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
		{Name: "Datastore/DynamoDB/all", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/DynamoDB/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/all", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/instance/DynamoDB/dynamodb.us-west-2.amazonaws.com/unknown", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/operation/DynamoDB/DescribeTable", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/operation/DynamoDB/DescribeTable", Scope: "OtherTransaction/Go/aws-txn", Forced: false, Data: nil},
	}...)
)

type testTableEntry struct {
	Name string

	BuildContext func(txn *newrelic.Transaction) context.Context
	BuildConfig  func(ctx context.Context, txn *newrelic.Transaction) aws.Config
}

func runTestTable(t *testing.T, table []*testTableEntry, executeEntry func(t *testing.T, entry *testTableEntry)) {
	for _, entry := range table {
		entry := entry // Pin range variable

		t.Run(entry.Name, func(t *testing.T) {
			executeEntry(t, entry)
		})
	}
}

func TestInstrumentRequestExternal(t *testing.T) {
	runTestTable(t,
		[]*testTableEntry{
			{
				Name: "with manually set transaction",

				BuildContext: func(txn *newrelic.Transaction) context.Context {
					return context.Background()
				},
				BuildConfig: newConfig,
			},
			{
				Name: "with transaction set in context",

				BuildContext: func(txn *newrelic.Transaction) context.Context {
					return newrelic.NewContext(context.Background(), txn)
				},
				BuildConfig: func(ctx context.Context, txn *newrelic.Transaction) aws.Config {
					return newConfig(ctx, nil) // Set txn to nil to ensure transaction is retrieved from the context
				},
			},
		},

		func(t *testing.T, entry *testTableEntry) {
			app := testApp()
			txn := app.StartTransaction(txnName)
			ctx := entry.BuildContext(txn)

			client := lambda.NewFromConfig(entry.BuildConfig(ctx, txn))

			input := &lambda.InvokeInput{
				ClientContext:  aws.String("MyApp"),
				FunctionName:   aws.String("non-existent-function"),
				InvocationType: types.InvocationTypeRequestResponse,
				LogType:        types.LogTypeTail,
				Payload:        []byte("{}"),
			}

			_, err := client.Invoke(ctx, input)
			if err != nil {
				t.Error(err)
			}

			txn.End()

			app.ExpectMetrics(t, externalMetrics)
			app.ExpectSpanEvents(t, []internal.WantEvent{
				externalSpan, genericSpan})
		},
	)
}

func TestInstrumentRequestDatastore(t *testing.T) {
	runTestTable(t,
		[]*testTableEntry{
			{
				Name: "with manually set transaction",

				BuildContext: func(txn *newrelic.Transaction) context.Context {
					return context.Background()
				},
				BuildConfig: newConfig,
			},
			{
				Name: "with transaction set in context",

				BuildContext: func(txn *newrelic.Transaction) context.Context {
					return newrelic.NewContext(context.Background(), txn)
				},
				BuildConfig: func(ctx context.Context, txn *newrelic.Transaction) aws.Config {
					return newConfig(ctx, nil) // Set txn to nil to ensure transaction is retrieved from the context
				},
			},
		},

		func(t *testing.T, entry *testTableEntry) {
			app := testApp()
			txn := app.StartTransaction(txnName)
			ctx := entry.BuildContext(txn)

			client := dynamodb.NewFromConfig(entry.BuildConfig(ctx, txn))

			input := &dynamodb.DescribeTableInput{
				TableName: aws.String("thebesttable"),
			}

			_, err := client.DescribeTable(ctx, input)
			if err != nil {
				t.Error(err)
			}

			txn.End()

			app.ExpectMetrics(t, datastoreMetrics)
			app.ExpectSpanEvents(t, []internal.WantEvent{
				datastoreSpan, genericSpan})
		},
	)
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
	runTestTable(t,
		[]*testTableEntry{
			{
				Name: "with manually set transaction",

				BuildContext: func(txn *newrelic.Transaction) context.Context {
					return context.Background()
				},
				BuildConfig: newConfig,
			},
			{
				Name: "with transaction set in context",

				BuildContext: func(txn *newrelic.Transaction) context.Context {
					return newrelic.NewContext(context.Background(), txn)
				},
				BuildConfig: func(ctx context.Context, txn *newrelic.Transaction) aws.Config {
					return newConfig(ctx, nil) // Set txn to nil to ensure transaction is retrieved from the context
				},
			},
		},

		func(t *testing.T, entry *testTableEntry) {
			app := testApp()
			txn := app.StartTransaction(txnName)
			ctx := entry.BuildContext(txn)

			cfg := entry.BuildConfig(ctx, txn)

			cfg.HTTPClient = &http.Client{
				Transport: &firstFailingTransport{failing: true},
			}

			customRetry := retry.NewStandard(func(o *retry.StandardOptions) {
				o.MaxAttempts = 2
			})
			client := lambda.NewFromConfig(cfg, func(o *lambda.Options) {
				o.Retryer = customRetry
			})

			input := &lambda.InvokeInput{
				ClientContext:  aws.String("MyApp"),
				FunctionName:   aws.String("non-existent-function"),
				InvocationType: types.InvocationTypeRequestResponse,
				LogType:        types.LogTypeTail,
				Payload:        []byte("{}"),
			}

			_, err := client.Invoke(ctx, input)
			if err != nil {
				t.Error(err)
			}

			txn.End()

			app.ExpectMetrics(t, externalMetrics)

			app.ExpectSpanEvents(t, []internal.WantEvent{
				{
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
						"aws.operation":   "Invoke",
						"aws.region":      awsRegion,
						"http.method":     "POST",
						"http.url":        "https://lambda.us-west-2.amazonaws.com/2015-03-31/functions/non-existent-function/invocations",
						"http.statusCode": "0",
					},
				}, {
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
						"aws.operation":   "Invoke",
						"aws.region":      awsRegion,
						"aws.requestId":   requestID,
						"http.method":     "POST",
						"http.url":        "https://lambda.us-west-2.amazonaws.com/2015-03-31/functions/non-existent-function/invocations",
						"http.statusCode": "200",
					},
				}, {
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
				}})
		},
	)
}

func TestRequestSentTwice(t *testing.T) {
	runTestTable(t,
		[]*testTableEntry{
			{
				Name: "with manually set transaction",

				BuildContext: func(txn *newrelic.Transaction) context.Context {
					return context.Background()
				},
				BuildConfig: newConfig,
			},
			{
				Name: "with transaction set in context",

				BuildContext: func(txn *newrelic.Transaction) context.Context {
					return newrelic.NewContext(context.Background(), txn)
				},
				BuildConfig: func(ctx context.Context, txn *newrelic.Transaction) aws.Config {
					return newConfig(ctx, nil) // Set txn to nil to ensure transaction is retrieved from the context
				},
			},
		},

		func(t *testing.T, entry *testTableEntry) {
			app := testApp()
			txn := app.StartTransaction(txnName)
			ctx := entry.BuildContext(txn)

			client := lambda.NewFromConfig(entry.BuildConfig(ctx, txn))

			input := &lambda.InvokeInput{
				ClientContext:  aws.String("MyApp"),
				FunctionName:   aws.String("non-existent-function"),
				InvocationType: types.InvocationTypeRequestResponse,
				LogType:        types.LogTypeTail,
				Payload:        []byte("{}"),
			}

			_, firstErr := client.Invoke(ctx, input)
			if firstErr != nil {
				t.Error(firstErr)
			}

			_, secondErr := client.Invoke(ctx, input)
			if secondErr != nil {
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
			app.ExpectSpanEvents(t, []internal.WantEvent{
				externalSpan, externalSpan, genericSpan})
		},
	)
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
	runTestTable(t,
		[]*testTableEntry{
			{
				Name: "with manually set transaction",

				BuildContext: func(txn *newrelic.Transaction) context.Context {
					return context.Background()
				},
				BuildConfig: newConfig,
			},
			{
				Name: "with transaction set in context",

				BuildContext: func(txn *newrelic.Transaction) context.Context {
					return newrelic.NewContext(context.Background(), txn)
				},
				BuildConfig: func(ctx context.Context, txn *newrelic.Transaction) aws.Config {
					return newConfig(ctx, nil) // Set txn to nil to ensure transaction is retrieved from the context
				},
			},
		},

		func(t *testing.T, entry *testTableEntry) {
			app := testApp()
			txn := app.StartTransaction(txnName)
			ctx := entry.BuildContext(txn)

			cfg := entry.BuildConfig(ctx, txn)
			cfg.HTTPClient = &http.Client{
				Transport: &noRequestIDTransport{},
			}
			client := lambda.NewFromConfig(cfg)

			input := &lambda.InvokeInput{
				ClientContext:  aws.String("MyApp"),
				FunctionName:   aws.String("non-existent-function"),
				InvocationType: types.InvocationTypeRequestResponse,
				LogType:        types.LogTypeTail,
				Payload:        []byte("{}"),
			}
			_, err := client.Invoke(ctx, input)
			if err != nil {
				t.Error(err)
			}

			txn.End()

			app.ExpectMetrics(t, externalMetrics)
			app.ExpectSpanEvents(t, []internal.WantEvent{
				externalSpanNoRequestID, genericSpan})
		},
	)
}
