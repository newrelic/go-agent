package nrawssdk

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/internal"
)

func testApp(t *testing.T) newrelic.Application {
	cfg := newrelic.NewConfig("appname", "0123456789012345678901234567890123456789")
	cfg.Enabled = false
	app, err := newrelic.NewApplication(cfg)
	if nil != err {
		t.Fatal(err)
	}
	internal.HarvestTesting(app, nil)
	return app
}

type fakeTransport struct{}

func (t fakeTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewReader([]byte(""))),
	}, nil
}

type fakeCreds struct{}

func (c fakeCreds) Retrieve() (aws.Credentials, error) {
	return aws.Credentials{}, nil
}

func newConfig(instrument bool) aws.Config {
	cfg, _ := external.LoadDefaultAWSConfig()
	cfg.Credentials = fakeCreds{}
	cfg.Region = endpoints.UsWest2RegionID
	cfg.HTTPClient.Transport = &fakeTransport{}

	if instrument {
		cfg = InstrumentConfig(cfg)
	}
	return cfg
}

func TestInstrumentRequestExternal(t *testing.T) {
	app := testApp(t)
	txn := app.StartTransaction("lambda-txn", nil, nil)

	client := lambda.New(newConfig(false))
	input := &lambda.InvokeInput{
		ClientContext:  aws.String("MyApp"),
		FunctionName:   aws.String("non-existent-function"),
		InvocationType: lambda.InvocationTypeEvent,
		LogType:        lambda.LogTypeTail,
		Payload:        []byte("{}"),
	}
	req := client.InvokeRequest(input)
	req.Request = InstrumentRequest(req.Request, txn)

	_, err := req.Send()
	if nil != err {
		t.Error(err)
	}

	txn.End()

	app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "External/all", Scope: "", Forced: true, Data: nil},
		{Name: "External/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "External/lambda.us-west-2.amazonaws.com/all", Scope: "", Forced: false, Data: nil},
		{Name: "External/lambda.us-west-2.amazonaws.com/all", Scope: "OtherTransaction/Go/lambda-txn", Forced: false, Data: nil},
		{Name: "OtherTransaction/Go/lambda-txn", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
	})
}

func TestInstrumentRequestDatastore(t *testing.T) {
	app := testApp(t)
	txn := app.StartTransaction("dynamodb-txn", nil, nil)

	client := dynamodb.New(newConfig(false))
	input := &dynamodb.DescribeTableInput{
		TableName: aws.String("thebesttable"),
	}

	req := client.DescribeTableRequest(input)
	req.Request = InstrumentRequest(req.Request, txn)

	_, err := req.Send()
	if nil != err {
		t.Error(err)
	}

	txn.End()

	app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "Datastore/DynamoDB/all", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/DynamoDB/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/all", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/instance/DynamoDB/dynamodb.us-west-2.amazonaws.com/unknown", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/operation/DynamoDB/DescribeTable", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/statement/DynamoDB/thebesttable/DescribeTable", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/statement/DynamoDB/thebesttable/DescribeTable", Scope: "OtherTransaction/Go/dynamodb-txn", Forced: false, Data: nil},
		{Name: "OtherTransaction/Go/dynamodb-txn", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
	})
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
	req.Request = InstrumentRequest(req.Request, nil)

	_, err := req.Send()
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
	req.Request = InstrumentRequest(req.Request, nil)

	_, err := req.Send()
	if nil != err {
		t.Error(err)
	}
}

func TestInstrumentConfigExternal(t *testing.T) {
	app := testApp(t)
	txn := app.StartTransaction("lambda-txn", nil, nil)

	client := lambda.New(newConfig(true))

	input := &lambda.InvokeInput{
		ClientContext:  aws.String("MyApp"),
		FunctionName:   aws.String("non-existent-function"),
		InvocationType: lambda.InvocationTypeEvent,
		LogType:        lambda.LogTypeTail,
		Payload:        []byte("{}"),
	}

	req := client.InvokeRequest(input)
	req.HTTPRequest = newrelic.RequestWithTransactionContext(req.HTTPRequest, txn)

	_, err := req.Send()
	if nil != err {
		t.Error(err)
	}

	txn.End()

	app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "External/all", Scope: "", Forced: true, Data: nil},
		{Name: "External/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "External/lambda.us-west-2.amazonaws.com/all", Scope: "", Forced: false, Data: nil},
		{Name: "External/lambda.us-west-2.amazonaws.com/all", Scope: "OtherTransaction/Go/lambda-txn", Forced: false, Data: nil},
		{Name: "OtherTransaction/Go/lambda-txn", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
	})
}

func TestInstrumentConfigDatastore(t *testing.T) {
	app := testApp(t)
	txn := app.StartTransaction("dynamodb-txn", nil, nil)

	client := dynamodb.New(newConfig(true))

	input := &dynamodb.DescribeTableInput{
		TableName: aws.String("thebesttable"),
	}

	req := client.DescribeTableRequest(input)
	req.HTTPRequest = newrelic.RequestWithTransactionContext(req.HTTPRequest, txn)

	_, err := req.Send()
	if nil != err {
		t.Error(err)
	}

	txn.End()

	app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "Datastore/DynamoDB/all", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/DynamoDB/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/all", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/allOther", Scope: "", Forced: true, Data: nil},
		{Name: "Datastore/instance/DynamoDB/dynamodb.us-west-2.amazonaws.com/unknown", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/operation/DynamoDB/DescribeTable", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/statement/DynamoDB/thebesttable/DescribeTable", Scope: "", Forced: false, Data: nil},
		{Name: "Datastore/statement/DynamoDB/thebesttable/DescribeTable", Scope: "OtherTransaction/Go/dynamodb-txn", Forced: false, Data: nil},
		{Name: "OtherTransaction/Go/dynamodb-txn", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
	})
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
	req.HTTPRequest = newrelic.RequestWithTransactionContext(req.HTTPRequest, nil)

	_, err := req.Send()
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
	req.HTTPRequest = newrelic.RequestWithTransactionContext(req.HTTPRequest, nil)

	_, err := req.Send()
	if nil != err {
		t.Error(err)
	}
}

func TestInstrumentConfigExternalTxnNotInCtx(t *testing.T) {
	app := testApp(t)
	txn := app.StartTransaction("lambda-txn", nil, nil)

	client := lambda.New(newConfig(true))

	input := &lambda.InvokeInput{
		ClientContext:  aws.String("MyApp"),
		FunctionName:   aws.String("non-existent-function"),
		InvocationType: lambda.InvocationTypeEvent,
		LogType:        lambda.LogTypeTail,
		Payload:        []byte("{}"),
	}

	req := client.InvokeRequest(input)

	_, err := req.Send()
	if nil != err {
		t.Error(err)
	}

	txn.End()

	app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/lambda-txn", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
	})
}

func TestInstrumentConfigDatastoreTxnNotInCtx(t *testing.T) {
	app := testApp(t)
	txn := app.StartTransaction("dynamodb-txn", nil, nil)

	client := dynamodb.New(newConfig(true))

	input := &dynamodb.DescribeTableInput{
		TableName: aws.String("thebesttable"),
	}

	req := client.DescribeTableRequest(input)

	_, err := req.Send()
	if nil != err {
		t.Error(err)
	}

	txn.End()

	app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/dynamodb-txn", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
	})
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

type firstFailingTransport struct{}

var failing = true

func (t firstFailingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if failing {
		failing = false
		return nil, errors.New("Oops this failed")
	}
	return &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewReader([]byte(""))),
	}, nil
}

func TestRetrySend(t *testing.T) {
	app := testApp(t)
	txn := app.StartTransaction("lambda-txn", nil, nil)

	cfg := newConfig(false)
	cfg.HTTPClient.Transport = &firstFailingTransport{}

	client := lambda.New(cfg)
	input := &lambda.InvokeInput{
		ClientContext:  aws.String("MyApp"),
		FunctionName:   aws.String("non-existent-function"),
		InvocationType: lambda.InvocationTypeEvent,
		LogType:        lambda.LogTypeTail,
		Payload:        []byte("{}"),
	}
	req := client.InvokeRequest(input)
	req.Request = InstrumentRequest(req.Request, txn)

	_, err := req.Send()
	if nil != err {
		t.Error(err)
	}

	txn.End()

	app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "External/all", Scope: "", Forced: true, Data: []float64{2}},
		{Name: "External/allOther", Scope: "", Forced: true, Data: []float64{2}},
		{Name: "External/lambda.us-west-2.amazonaws.com/all", Scope: "", Forced: false, Data: []float64{2}},
		{Name: "External/lambda.us-west-2.amazonaws.com/all", Scope: "OtherTransaction/Go/lambda-txn", Forced: false, Data: []float64{2}},
		{Name: "OtherTransaction/Go/lambda-txn", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
	})
}

func TestRequestSentTwice(t *testing.T) {
	app := testApp(t)
	txn := app.StartTransaction("lambda-txn", nil, nil)

	client := lambda.New(newConfig(false))
	input := &lambda.InvokeInput{
		ClientContext:  aws.String("MyApp"),
		FunctionName:   aws.String("non-existent-function"),
		InvocationType: lambda.InvocationTypeEvent,
		LogType:        lambda.LogTypeTail,
		Payload:        []byte("{}"),
	}
	req := client.InvokeRequest(input)
	req.Request = InstrumentRequest(req.Request, txn)

	_, firstErr := req.Send()
	if nil != firstErr {
		t.Error(firstErr)
	}

	_, secondErr := req.Send()
	if nil != secondErr {
		t.Error(secondErr)
	}

	txn.End()

	app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "External/all", Scope: "", Forced: true, Data: []float64{2}},
		{Name: "External/allOther", Scope: "", Forced: true, Data: []float64{2}},
		{Name: "External/lambda.us-west-2.amazonaws.com/all", Scope: "", Forced: false, Data: []float64{2}},
		{Name: "External/lambda.us-west-2.amazonaws.com/all", Scope: "OtherTransaction/Go/lambda-txn", Forced: false, Data: []float64{2}},
		{Name: "OtherTransaction/Go/lambda-txn", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
	})
}
