package nrawssdk

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go/aws/endpoints"
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

func newConfig(instrument bool) aws.Config {
	cfg, _ := external.LoadDefaultAWSConfig()
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
		{"External/all", "", true, nil},
		{"External/allOther", "", true, nil},
		{"External/lambda.us-west-2.amazonaws.com/all", "", false, nil},
		{"External/lambda.us-west-2.amazonaws.com/all", "OtherTransaction/Go/lambda-txn", false, nil},
		{"OtherTransaction/Go/lambda-txn", "", true, nil},
		{"OtherTransaction/all", "", true, nil},
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
		{"Datastore/DynamoDB/all", "", true, nil},
		{"Datastore/DynamoDB/allOther", "", true, nil},
		{"Datastore/all", "", true, nil},
		{"Datastore/allOther", "", true, nil},
		{"Datastore/instance/DynamoDB/dynamodb.us-west-2.amazonaws.com/unknown", "", false, nil},
		{"Datastore/operation/DynamoDB/DescribeTable", "", false, nil},
		{"Datastore/statement/DynamoDB/thebesttable/DescribeTable", "", false, nil},
		{"Datastore/statement/DynamoDB/thebesttable/DescribeTable", "OtherTransaction/Go/dynamodb-txn", false, nil},
		{"OtherTransaction/Go/dynamodb-txn", "", true, nil},
		{"OtherTransaction/all", "", true, nil},
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
		{"External/all", "", true, nil},
		{"External/allOther", "", true, nil},
		{"External/lambda.us-west-2.amazonaws.com/all", "", false, nil},
		{"External/lambda.us-west-2.amazonaws.com/all", "OtherTransaction/Go/lambda-txn", false, nil},
		{"OtherTransaction/Go/lambda-txn", "", true, nil},
		{"OtherTransaction/all", "", true, nil},
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
		{"Datastore/DynamoDB/all", "", true, nil},
		{"Datastore/DynamoDB/allOther", "", true, nil},
		{"Datastore/all", "", true, nil},
		{"Datastore/allOther", "", true, nil},
		{"Datastore/instance/DynamoDB/dynamodb.us-west-2.amazonaws.com/unknown", "", false, nil},
		{"Datastore/operation/DynamoDB/DescribeTable", "", false, nil},
		{"Datastore/statement/DynamoDB/thebesttable/DescribeTable", "", false, nil},
		{"Datastore/statement/DynamoDB/thebesttable/DescribeTable", "OtherTransaction/Go/dynamodb-txn", false, nil},
		{"OtherTransaction/Go/dynamodb-txn", "", true, nil},
		{"OtherTransaction/all", "", true, nil},
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
		{"OtherTransaction/Go/lambda-txn", "", true, nil},
		{"OtherTransaction/all", "", true, nil},
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
		{"OtherTransaction/Go/dynamodb-txn", "", true, nil},
		{"OtherTransaction/all", "", true, nil},
	})
}

func TestDoublyInstrumented(t *testing.T) {
	countHandlers := func(hs *aws.Handlers, t *testing.T, expected int) {
		if found := hs.Validate.Len(); expected != found {
			t.Error("unexpected number of Validate handlers found:", found)
		}
		if found := hs.Complete.Len(); expected != found {
			t.Error("unexpected number of Complete handlers found:", found)
		}
	}

	hs := &aws.Handlers{}
	countHandlers(hs, t, 0)

	InstrumentHandlers(hs)
	countHandlers(hs, t, 1)

	InstrumentHandlers(hs)
	countHandlers(hs, t, 1)
}
