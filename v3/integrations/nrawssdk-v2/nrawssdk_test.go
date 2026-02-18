// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrawssdk

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdatypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/newrelic/go-agent/v3/newrelic/integrationsupport"
)

func testApp() integrationsupport.ExpectApp {
	return integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn, integrationsupport.DTEnabledCfgFn, newrelic.ConfigCodeLevelMetricsEnabled(false))
}

type fakeTransport struct{}

func (t fakeTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader([]byte(""))),
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
	return aws.Credentials{
		AccessKeyID: "",
		AccountID:   "",
	}, nil
}

type mockResolver struct {
	accountID string
	err       error
}

func (m *mockResolver) AWSAccountIdFromAWSAccessKey(_creds aws.Credentials) (string, error) {
	return m.accountID, m.err
}

var fakeCreds = func() interface{} {
	var c interface{} = fakeCredsWithoutContext{}
	if _, ok := c.(aws.CredentialsProvider); ok {
		return c
	}
	return fakeCredsWithContext{}
}()

func newConfig(ctx context.Context, txn *newrelic.Transaction) aws.Config {
	cfg, _ := config.LoadDefaultConfig(ctx, func(o *config.LoadOptions) error {
		return nil
	})
	cfg.Credentials = fakeCreds.(aws.CredentialsProvider)
	cfg.Region = awsRegion
	cfg.HTTPClient = &http.Client{
		Transport: &fakeTransport{},
	}
	// Ensure transaction is in context for NRAppendMiddlewares
	if txn != nil {
		ctx = newrelic.NewContext(ctx, txn)
	}
	NRAppendMiddlewares(&cfg.APIOptions, ctx, cfg)
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
	SQSSpan = internal.WantEvent{
		Intrinsics: map[string]interface{}{
			"name":      "External/sqs.us-west-2.amazonaws.com/http/POST",
			"category":  "http",
			"parentId":  internal.MatchAnything,
			"component": "http",
			"span.kind": "client",
			"sampled":   true,
		},
		UserAttributes: map[string]interface{}{},
		AgentAttributes: map[string]interface{}{
			"message.destination.name": "MyQueue",
			"cloud.account.id":         "123456789012",
			"cloud.region":             "us-west-2",
			"http.url":                 "https://sqs.us-west-2.amazonaws.com/",
			"http.method":              "POST",
			"messaging.system":         "aws_sqs",
			"aws.requestId":            "testing request id",
			"http.statusCode":          "200",
			"aws.region":               "us-west-2",
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
				InvocationType: lambdatypes.InvocationTypeRequestResponse,
				LogType:        lambdatypes.LogTypeTail,
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

type sqsTestTableEntry struct {
	Name         string
	BuildContext func(txn *newrelic.Transaction) context.Context
	BuildConfig  func(ctx context.Context, txn *newrelic.Transaction) aws.Config
	Input        interface{}
}

func runSQSTestTable(t *testing.T, entries []*sqsTestTableEntry, testFunc func(t *testing.T, entry *sqsTestTableEntry)) {
	for _, entry := range entries {
		t.Run(entry.Name, func(t *testing.T) {
			testFunc(t, entry)
		})
	}
}

func TestSQSMiddleware(t *testing.T) {
	runSQSTestTable(t,
		[]*sqsTestTableEntry{
			{
				Name: "DeleteQueueInput",
				BuildContext: func(txn *newrelic.Transaction) context.Context {
					return context.Background()
				},
				BuildConfig: func(ctx context.Context, txn *newrelic.Transaction) aws.Config {
					return newConfig(ctx, txn)
				},
				Input: &sqs.DeleteQueueInput{QueueUrl: aws.String("https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue")},
			},
			{
				Name: "ReceiveMessageInput",
				BuildContext: func(txn *newrelic.Transaction) context.Context {
					return context.Background()
				},
				BuildConfig: func(ctx context.Context, txn *newrelic.Transaction) aws.Config {
					return newConfig(ctx, txn)
				},
				Input: &sqs.ReceiveMessageInput{QueueUrl: aws.String("https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue")},
			},
			{
				Name: "SendMessageInput",
				BuildContext: func(txn *newrelic.Transaction) context.Context {
					return context.Background()
				},
				BuildConfig: func(ctx context.Context, txn *newrelic.Transaction) aws.Config {
					return newConfig(ctx, txn)
				},
				Input: &sqs.SendMessageInput{QueueUrl: aws.String("https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue"), MessageBody: aws.String("Hello, world!")},
			},
			{
				Name: "PurgeQueueInput",
				BuildContext: func(txn *newrelic.Transaction) context.Context {
					return context.Background()
				},
				BuildConfig: func(ctx context.Context, txn *newrelic.Transaction) aws.Config {
					return newConfig(ctx, txn)
				},
				Input: &sqs.PurgeQueueInput{QueueUrl: aws.String("https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue")},
			},
			{
				Name: "DeleteMessageInput",
				BuildContext: func(txn *newrelic.Transaction) context.Context {
					return context.Background()
				},
				BuildConfig: func(ctx context.Context, txn *newrelic.Transaction) aws.Config {
					return newConfig(ctx, txn)
				},
				Input: &sqs.DeleteMessageInput{QueueUrl: aws.String("https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue"), ReceiptHandle: aws.String("receipt-handle")},
			},
			{
				Name: "ChangeMessageVisibilityInput",
				BuildContext: func(txn *newrelic.Transaction) context.Context {
					return context.Background()
				},
				BuildConfig: func(ctx context.Context, txn *newrelic.Transaction) aws.Config {
					return newConfig(ctx, txn)
				},
				Input: &sqs.ChangeMessageVisibilityInput{QueueUrl: aws.String("https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue"), ReceiptHandle: aws.String("receipt-handle"), VisibilityTimeout: 10},
			},

			{
				Name: "ChangeMessageVisibilityBatchInput",
				BuildContext: func(txn *newrelic.Transaction) context.Context {
					return context.Background()
				},
				BuildConfig: func(ctx context.Context, txn *newrelic.Transaction) aws.Config {
					return newConfig(ctx, txn)
				},
				Input: &sqs.ChangeMessageVisibilityBatchInput{
					QueueUrl: aws.String("https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue"),
					Entries: []sqstypes.ChangeMessageVisibilityBatchRequestEntry{
						{
							Id:                aws.String("id1"),
							ReceiptHandle:     aws.String("receipt-handle"),
							VisibilityTimeout: 10,
						},
					},
				},
			},
			{
				Name: "DeleteMessageBatchInput",
				BuildContext: func(txn *newrelic.Transaction) context.Context {
					return context.Background()
				},
				BuildConfig: func(ctx context.Context, txn *newrelic.Transaction) aws.Config {
					return newConfig(ctx, txn)
				},
				Input: &sqs.DeleteMessageBatchInput{
					QueueUrl: aws.String("https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue"),
					Entries: []sqstypes.DeleteMessageBatchRequestEntry{
						{
							Id:            aws.String("id1"),
							ReceiptHandle: aws.String("receipt-handle"),
						},
					},
				},
			},
			{
				Name: "SendMessageBatchInput",
				BuildContext: func(txn *newrelic.Transaction) context.Context {
					return context.Background()
				},
				BuildConfig: func(ctx context.Context, txn *newrelic.Transaction) aws.Config {
					return newConfig(ctx, txn)
				},
				Input: &sqs.SendMessageBatchInput{
					QueueUrl: aws.String("https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue"),
					Entries: []sqstypes.SendMessageBatchRequestEntry{
						{
							Id:          aws.String("id1"),
							MessageBody: aws.String("Hello, world!"),
						},
					},
				},
			},
			{
				Name: "GetQueueAttributesInput",
				BuildContext: func(txn *newrelic.Transaction) context.Context {
					return context.Background()
				},
				BuildConfig: func(ctx context.Context, txn *newrelic.Transaction) aws.Config {
					return newConfig(ctx, txn)
				},
				Input: &sqs.GetQueueAttributesInput{
					QueueUrl: aws.String("https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue"),
					AttributeNames: []sqstypes.QueueAttributeName{
						"ApproximateNumberOfMessages",
					},
				},
			},
			{
				Name: "SetQueueAttributesInput",
				BuildContext: func(txn *newrelic.Transaction) context.Context {
					return context.Background()
				},
				BuildConfig: func(ctx context.Context, txn *newrelic.Transaction) aws.Config {
					return newConfig(ctx, txn)
				},
				Input: &sqs.SetQueueAttributesInput{
					QueueUrl: aws.String("https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue"),
					Attributes: map[string]string{
						"VisibilityTimeout": "10",
					},
				},
			},
			{
				Name: "TagQueueInput",
				BuildContext: func(txn *newrelic.Transaction) context.Context {
					return context.Background()
				},
				BuildConfig: func(ctx context.Context, txn *newrelic.Transaction) aws.Config {
					return newConfig(ctx, txn)
				},
				Input: &sqs.TagQueueInput{
					QueueUrl: aws.String("https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue"),
					Tags: map[string]string{
						"tag1": "value1",
					},
				},
			},
			{
				Name: "UntagQueueInput",
				BuildContext: func(txn *newrelic.Transaction) context.Context {
					return context.Background()
				},
				BuildConfig: func(ctx context.Context, txn *newrelic.Transaction) aws.Config {
					return newConfig(ctx, txn)
				},
				Input: &sqs.UntagQueueInput{
					QueueUrl: aws.String("https://sqs.us-west-2.amazonaws.com/123456789012/MyQueue"),
					TagKeys:  []string{"tag1"},
				},
			},
		},

		func(t *testing.T, entry *sqsTestTableEntry) {
			app := testApp()
			txn := app.StartTransaction(txnName)
			ctx := entry.BuildContext(txn)
			awsOp := ""
			client := sqs.NewFromConfig(entry.BuildConfig(ctx, txn))
			switch input := entry.Input.(type) {
			case *sqs.SendMessageInput:
				client.SendMessage(ctx, input)
				awsOp = "SendMessage"
			case *sqs.DeleteQueueInput:
				client.DeleteQueue(ctx, input)
				awsOp = "DeleteQueue"
			case *sqs.ReceiveMessageInput:
				client.ReceiveMessage(ctx, input)
				awsOp = "ReceiveMessage"
			case *sqs.DeleteMessageInput:
				client.DeleteMessage(ctx, input)
				awsOp = "DeleteMessage"
			case *sqs.ChangeMessageVisibilityInput:
				client.ChangeMessageVisibility(ctx, input)
				awsOp = "ChangeMessageVisibility"
			case *sqs.ChangeMessageVisibilityBatchInput:
				client.ChangeMessageVisibilityBatch(ctx, input)
				awsOp = "ChangeMessageVisibilityBatch"
			case *sqs.DeleteMessageBatchInput:
				client.DeleteMessageBatch(ctx, input)
				awsOp = "DeleteMessageBatch"
			case *sqs.PurgeQueueInput:
				client.PurgeQueue(ctx, input)
				awsOp = "PurgeQueue"
			case *sqs.GetQueueAttributesInput:
				client.GetQueueAttributes(ctx, input)
				awsOp = "GetQueueAttributes"
			case *sqs.SetQueueAttributesInput:
				client.SetQueueAttributes(ctx, input)
				awsOp = "SetQueueAttributes"
			case *sqs.TagQueueInput:
				client.TagQueue(ctx, input)
				awsOp = "TagQueue"
			case *sqs.UntagQueueInput:
				client.UntagQueue(ctx, input)
				awsOp = "UntagQueue"
			case *sqs.SendMessageBatchInput:
				client.SendMessageBatch(ctx, input)
				awsOp = "SendMessageBatch"

			default:
				t.Errorf("unexpected input type: %T", input)

			}

			txn.End()
			SQSSpanModified := SQSSpan
			SQSSpanModified.AgentAttributes["aws.operation"] = awsOp
			app.ExpectSpanEvents(t, []internal.WantEvent{
				SQSSpan, genericSpan})

		},
	)
}

func TestInstrumentRequestDynamoDB(t *testing.T) {
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

			input := &dynamodb.GetItemInput{
				Key: map[string]dynamodbtypes.AttributeValue{
					"PartitionKey": &dynamodbtypes.AttributeValueMemberS{Value: "foo"},
				},
				TableName: aws.String("thebesttable"),
			}

			_, err := client.GetItem(ctx, input)
			if err != nil {
				t.Error(err)
			}

			txn.End()

			var datastoreMetrics []internal.WantMetric
			datastoreMetrics = append(datastoreMetrics, txnMetrics...)
			datastoreMetrics = append(datastoreMetrics, []internal.WantMetric{
				{Name: "Datastore/DynamoDB/all", Scope: "", Forced: true, Data: nil},
				{Name: "Datastore/DynamoDB/allOther", Scope: "", Forced: true, Data: nil},
				{Name: "Datastore/all", Scope: "", Forced: true, Data: nil},
				{Name: "Datastore/allOther", Scope: "", Forced: true, Data: nil},
				{Name: "Datastore/instance/DynamoDB/dynamodb.us-west-2.amazonaws.com/unknown", Scope: "", Forced: false, Data: nil},
				{Name: "Datastore/operation/DynamoDB/GetItem", Scope: "", Forced: false, Data: nil},
				{Name: "Datastore/statement/DynamoDB/thebesttable/GetItem", Scope: "", Forced: false, Data: nil},
				{Name: "Datastore/statement/DynamoDB/thebesttable/GetItem", Scope: "OtherTransaction/Go/aws-txn", Forced: false, Data: nil},
			}...)
			app.ExpectMetrics(t, datastoreMetrics)

			app.ExpectSpanEvents(t, []internal.WantEvent{
				{
					Intrinsics: map[string]interface{}{
						"name":          "Datastore/statement/DynamoDB/thebesttable/GetItem",
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
						"aws.operation":   "GetItem",
						"aws.region":      awsRegion,
						"aws.requestId":   requestID,
						"db.collection":   "thebesttable",
						"db.statement":    "'GetItem' on 'thebesttable' using 'DynamoDB'",
						"peer.address":    "dynamodb.us-west-2.amazonaws.com:unknown",
						"peer.hostname":   "dynamodb.us-west-2.amazonaws.com",
						"http.statusCode": "200",
					},
				},
				genericSpan,
			})
		},
	)
}

func TestInstrumentRequestDynamoDBWithIndex(t *testing.T) {
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

			input := &dynamodb.ScanInput{
				TableName: aws.String("thebesttable"),
				IndexName: aws.String("someindex"),
			}

			_, err := client.Scan(ctx, input)
			if err != nil {
				t.Error(err)
			}

			txn.End()

			var datastoreMetrics []internal.WantMetric
			datastoreMetrics = append(datastoreMetrics, txnMetrics...)
			datastoreMetrics = append(datastoreMetrics, []internal.WantMetric{
				{Name: "Datastore/DynamoDB/all", Scope: "", Forced: true, Data: nil},
				{Name: "Datastore/DynamoDB/allOther", Scope: "", Forced: true, Data: nil},
				{Name: "Datastore/all", Scope: "", Forced: true, Data: nil},
				{Name: "Datastore/allOther", Scope: "", Forced: true, Data: nil},
				{Name: "Datastore/instance/DynamoDB/dynamodb.us-west-2.amazonaws.com/unknown", Scope: "", Forced: false, Data: nil},
				{Name: "Datastore/operation/DynamoDB/Scan", Scope: "", Forced: false, Data: nil},
				{Name: "Datastore/statement/DynamoDB/thebesttable.someindex/Scan", Scope: "", Forced: false, Data: nil},
				{Name: "Datastore/statement/DynamoDB/thebesttable.someindex/Scan", Scope: "OtherTransaction/Go/aws-txn", Forced: false, Data: nil},
			}...)
			app.ExpectMetrics(t, datastoreMetrics)

			app.ExpectSpanEvents(t, []internal.WantEvent{
				{
					Intrinsics: map[string]interface{}{
						"name":          "Datastore/statement/DynamoDB/thebesttable.someindex/Scan",
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
						"aws.operation":   "Scan",
						"aws.region":      awsRegion,
						"aws.requestId":   requestID,
						"db.collection":   "thebesttable.someindex",
						"db.statement":    "'Scan' on 'thebesttable.someindex' using 'DynamoDB'",
						"peer.address":    "dynamodb.us-west-2.amazonaws.com:unknown",
						"peer.hostname":   "dynamodb.us-west-2.amazonaws.com",
						"http.statusCode": "200",
					},
				},
				genericSpan,
			})
		},
	)
}

func TestInstrumentRequestDynamoDBOther(t *testing.T) {
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

			input := &dynamodb.BatchGetItemInput{
				RequestItems: map[string]dynamodbtypes.KeysAndAttributes{
					"FirstTable": {
						Keys: []map[string]dynamodbtypes.AttributeValue{
							{"PartitionKey": &dynamodbtypes.AttributeValueMemberS{Value: "foo"}},
						},
					},
					"SecondTable": {
						Keys: []map[string]dynamodbtypes.AttributeValue{
							{"PartitionKey": &dynamodbtypes.AttributeValueMemberS{Value: "bar"}},
						},
					},
				},
			}

			_, err := client.BatchGetItem(ctx, input)
			if err != nil {
				t.Error(err)
			}

			txn.End()

			var datastoreMetrics []internal.WantMetric
			datastoreMetrics = append(datastoreMetrics, txnMetrics...)
			datastoreMetrics = append(datastoreMetrics, []internal.WantMetric{
				{Name: "Datastore/DynamoDB/all", Scope: "", Forced: true, Data: nil},
				{Name: "Datastore/DynamoDB/allOther", Scope: "", Forced: true, Data: nil},
				{Name: "Datastore/all", Scope: "", Forced: true, Data: nil},
				{Name: "Datastore/allOther", Scope: "", Forced: true, Data: nil},
				{Name: "Datastore/instance/DynamoDB/dynamodb.us-west-2.amazonaws.com/unknown", Scope: "", Forced: false, Data: nil},
				{Name: "Datastore/operation/DynamoDB/BatchGetItem", Scope: "", Forced: false, Data: nil},
				{Name: "Datastore/operation/DynamoDB/BatchGetItem", Scope: "OtherTransaction/Go/aws-txn", Forced: false, Data: nil},
			}...)
			app.ExpectMetrics(t, datastoreMetrics)

			app.ExpectSpanEvents(t, []internal.WantEvent{
				{
					Intrinsics: map[string]interface{}{
						"name":          "Datastore/operation/DynamoDB/BatchGetItem",
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
						"aws.operation":   "BatchGetItem",
						"aws.region":      awsRegion,
						"aws.requestId":   requestID,
						"db.statement":    "'BatchGetItem' on 'unknown' using 'DynamoDB'",
						"peer.address":    "dynamodb.us-west-2.amazonaws.com:unknown",
						"peer.hostname":   "dynamodb.us-west-2.amazonaws.com",
						"http.statusCode": "200",
					},
				},
				genericSpan,
			})
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
		Body:       io.NopCloser(bytes.NewReader([]byte(""))),
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
				InvocationType: lambdatypes.InvocationTypeRequestResponse,
				LogType:        lambdatypes.LogTypeTail,
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
				InvocationType: lambdatypes.InvocationTypeRequestResponse,
				LogType:        lambdatypes.LogTypeTail,
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
		Body:       io.NopCloser(bytes.NewReader([]byte(""))),
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
				InvocationType: lambdatypes.InvocationTypeRequestResponse,
				LogType:        lambdatypes.LogTypeTail,
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

func TestResolveAWSCredentials(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		cfg          newrelic.Config
		mockResolver mockResolver
		want         string
		wantErr      bool
		wantedErr    string
		test         bool
	}{
		{
			name: "Error from AWSAccountIDFromAWSAccessKey",
			cfg: newrelic.Config{
				CloudAWS: struct {
					AccountID       string
					AccountDecoding struct{ Enabled bool }
				}{
					AccountID: "",
					AccountDecoding: struct{ Enabled bool }{
						Enabled: true,
					},
				},
			},
			mockResolver: mockResolver{
				accountID: "",
				err:       fmt.Errorf("error from called function"),
			},
			want:      "",
			wantErr:   true,
			wantedErr: "error from called function",
		},
		{
			name: "AccountID exists in config with account encoding enabled. Should return config accountID",
			cfg: newrelic.Config{
				CloudAWS: struct {
					AccountID       string
					AccountDecoding struct{ Enabled bool }
				}{
					AccountID: "123234345456",
					AccountDecoding: struct{ Enabled bool }{
						Enabled: true,
					},
				},
			},
			mockResolver: mockResolver{
				accountID: "",
				err:       nil,
			},
			want:      "123234345456",
			wantErr:   false,
			wantedErr: "",
		},
		{
			name: "AccountID exists in config with account encoding disabled. Should return config accountID",
			cfg: newrelic.Config{
				CloudAWS: struct {
					AccountID       string
					AccountDecoding struct{ Enabled bool }
				}{
					AccountID: "123234345456",
					AccountDecoding: struct{ Enabled bool }{
						Enabled: false,
					},
				},
			},
			mockResolver: mockResolver{
				accountID: "",
				err:       nil,
			},
			want:      "123234345456",
			wantErr:   false,
			wantedErr: "",
		},
		{
			name: "AccountID exists in config with same accountID resolved and account decoding enabled. Should return config accountID",
			cfg: newrelic.Config{
				CloudAWS: struct {
					AccountID       string
					AccountDecoding struct{ Enabled bool }
				}{
					AccountID: "123234345456",
					AccountDecoding: struct{ Enabled bool }{
						Enabled: true,
					},
				},
			},
			mockResolver: mockResolver{
				accountID: "123234345456",
				err:       nil,
			},
			want:      "123234345456",
			wantErr:   false,
			wantedErr: "",
		},
		{
			name: "AccountID exists in config with different accountID resolved and account decoding enabled. Should return config accountID",
			cfg: newrelic.Config{
				CloudAWS: struct {
					AccountID       string
					AccountDecoding struct{ Enabled bool }
				}{
					AccountID: "123234345456",
					AccountDecoding: struct{ Enabled bool }{
						Enabled: true,
					},
				},
			},
			mockResolver: mockResolver{
				accountID: "123234345457",
				err:       nil,
			},
			want:      "123234345456",
			wantErr:   false,
			wantedErr: "",
		},
		{
			name: "AccountID empty in config with different accountID resolved and account decoding enabled. Should return resolved accountID",
			cfg: newrelic.Config{
				CloudAWS: struct {
					AccountID       string
					AccountDecoding struct{ Enabled bool }
				}{
					AccountDecoding: struct{ Enabled bool }{
						Enabled: true,
					},
				},
			},
			mockResolver: mockResolver{
				accountID: "123234345457",
				err:       nil,
			},
			want:      "123234345457",
			wantErr:   false,
			wantedErr: "",
		},
		{
			name: "AccountID empty in config with different accountID resolved and account decoding enabled. Should return empty config accountID",
			cfg: newrelic.Config{
				CloudAWS: struct {
					AccountID       string
					AccountDecoding struct{ Enabled bool }
				}{
					AccountDecoding: struct{ Enabled bool }{
						Enabled: false,
					},
				},
			},
			mockResolver: mockResolver{
				accountID: "123234345457",
				err:       nil,
			},
			want:      "",
			wantErr:   false,
			wantedErr: "",
		},
		{
			name: "AccountID empty in config with empty accountID resolved. Should return config accountID",
			cfg:  newrelic.Config{},
			mockResolver: mockResolver{
				accountID: "",
				err:       nil,
			},
			want:      "",
			wantErr:   false,
			wantedErr: "",
		},
	}
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			m := nrMiddleware{
				resolver: &tt.mockResolver,
			}
			gotErr := m.ResolveAWSCredentials(tt.cfg, aws.Credentials{})
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("ResolveAWSCredentials() failed: %v", gotErr)
				} else {
					if tt.wantedErr != gotErr.Error() {
						t.Errorf("ResolveAWSCredentials() error = %v, want %v", gotErr.Error(), tt.wantedErr)
					}
				}
				return
			}
			if tt.wantErr {
				t.Fatal("ResolveAWSCredentials() succeeded unexpectedly")
			}
			if m.accountID != tt.want {
				t.Errorf("ResolveAWSCredentials() = %v, want %v", m.accountID, tt.want)
			}
		})
	}
}

func TestAWSAccountIdFromAWSAccessKey(t *testing.T) {
	tests := []struct {
		name       string
		creds      aws.Credentials
		want       string
		wantErr    bool
		wantErrStr string
	}{
		{
			name: "Valid access key returns account ID",
			creds: aws.Credentials{
				AccountID:   "",
				AccessKeyID: "AKIASAWSR23456AWS357",
			},
			want:    "138954266361",
			wantErr: false,
		},
		{
			name: "AccountID already exists and access key exists. Should return AccountID immediately",
			creds: aws.Credentials{
				AccountID:   "123451234512",
				AccessKeyID: "ASKDHA123457AKJFHAKS",
			},
			want:    "123451234512",
			wantErr: false,
		},
		{
			name: "AccountID already exists and access key exists with too short of length. Should return AccountID immediately",
			creds: aws.Credentials{
				AccountID:   "123451234512",
				AccessKeyID: "a",
			},
			want:    "123451234512",
			wantErr: false,
		},
		{
			name: "AccountID already exists and access key exists with improper format. Should return AccountID immediately",
			creds: aws.Credentials{
				AccountID:   "123451234512",
				AccessKeyID: "a a a.                      ",
			},
			want:    "123451234512",
			wantErr: false,
		},
		{
			name: "AccountID already exists and access key does not exist. Should return AccountID immediately",
			creds: aws.Credentials{
				AccountID: "123451234512",
			},
			want:    "123451234512",
			wantErr: false,
		},
		{
			name:       "AccountID does not exist and access key does not exist. Should return an error",
			creds:      aws.Credentials{},
			want:       "",
			wantErr:    true,
			wantErrStr: "no access key id found",
		},
		{
			name: "AccountID does not exist and access key is in an improper format. Should return an error",
			creds: aws.Credentials{
				AccessKeyID: "123asdfas",
			},
			want:       "",
			wantErr:    true,
			wantErrStr: "improper access key id format",
		},
		{
			name: "AccountID does not exist and access key is in an improper format with only one character. Should return an error",
			creds: aws.Credentials{
				AccessKeyID: "a",
			},
			want:       "",
			wantErr:    true,
			wantErrStr: "improper access key id format",
		},
		{
			name: "AccountID does not exist and access key is in an improper format for decoding",
			creds: aws.Credentials{
				AccessKeyID: "a a a.                      ",
			},
			want:       "",
			wantErr:    true,
			wantErrStr: "error decoding access keys",
		},
		{
			name: "AccountID does not exist and access key contains non base32 characters",
			creds: aws.Credentials{
				AccessKeyID: "AKIA1234567899876541",
			},
			want:       "",
			wantErr:    true,
			wantErrStr: "error decoding access keys",
		},
		{
			name: "AccountID does not exist and access key contains non base32 characters and is too short in length",
			creds: aws.Credentials{
				AccessKeyID: "AKIA1818",
			},
			want:       "",
			wantErr:    true,
			wantErrStr: "improper access key id format",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &defaultResolver{}
			got, gotErr := resolver.AWSAccountIdFromAWSAccessKey(tt.creds)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("AWSAccountIdFromAWSAccessKey() failed: %v", gotErr)
				} else {
					if tt.wantErrStr != gotErr.Error() {
						t.Errorf("AWSAccountIdFromAWSAccessKey() error = %v, want %v", gotErr.Error(), tt.wantErrStr)
					}
				}
				return
			}
			if tt.wantErr {
				t.Fatal("AWSAccountIdFromAWSAccessKey() succeeded unexpectedly")
			}
			if tt.want != got {
				t.Errorf("AWSAccountIdFromAWSAccessKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
