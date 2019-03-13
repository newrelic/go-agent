package nrlambda

import (
	"testing"

	"github.com/aws/aws-lambda-go/events"
	newrelic "github.com/newrelic/go-agent"
)

func TestGetEventAttributes(t *testing.T) {
	testcases := []struct {
		Name  string
		Input interface{}
		Arn   string
	}{
		{Name: "nil", Input: nil, Arn: ""},
		{Name: "SQSEvent empty", Input: events.SQSEvent{}, Arn: ""},
		{Name: "SQSEvent", Input: events.SQSEvent{
			Records: []events.SQSMessage{{
				EventSourceARN: "ARN",
			}},
		}, Arn: "ARN"},
		{Name: "SNSEvent empty", Input: events.SNSEvent{}, Arn: ""},
		{Name: "SNSEvent", Input: events.SNSEvent{
			Records: []events.SNSEventRecord{{
				EventSubscriptionArn: "ARN",
			}},
		}, Arn: "ARN"},
		{Name: "S3Event empty", Input: events.S3Event{}, Arn: ""},
		{Name: "S3Event", Input: events.S3Event{
			Records: []events.S3EventRecord{{
				S3: events.S3Entity{
					Bucket: events.S3Bucket{
						Arn: "ARN",
					},
				},
			}},
		}, Arn: "ARN"},
		{Name: "DynamoDBEvent empty", Input: events.DynamoDBEvent{}, Arn: ""},
		{Name: "DynamoDBEvent", Input: events.DynamoDBEvent{
			Records: []events.DynamoDBEventRecord{{
				EventSourceArn: "ARN",
			}},
		}, Arn: "ARN"},
		{Name: "CodeCommitEvent empty", Input: events.CodeCommitEvent{}, Arn: ""},
		{Name: "CodeCommitEvent", Input: events.CodeCommitEvent{
			Records: []events.CodeCommitRecord{{
				EventSourceARN: "ARN",
			}},
		}, Arn: "ARN"},
		{Name: "KinesisEvent empty", Input: events.KinesisEvent{}, Arn: ""},
		{Name: "KinesisEvent", Input: events.KinesisEvent{
			Records: []events.KinesisEventRecord{{
				EventSourceArn: "ARN",
			}},
		}, Arn: "ARN"},
		{Name: "KinesisFirehoseEvent", Input: events.KinesisFirehoseEvent{
			DeliveryStreamArn: "ARN",
		}, Arn: "ARN"},
	}

	for _, testcase := range testcases {
		arn := getEventSourceARN(testcase.Input)
		if arn != testcase.Arn {
			t.Error(testcase.Name, arn, testcase.Arn)
		}
	}
}

func TestEventWebRequest(t *testing.T) {
	// First test a type that does not count as a web request.
	req := eventWebRequest(22)
	if nil != req {
		t.Error(req)
	}

	testcases := []struct {
		testname   string
		input      interface{}
		numHeaders int
		method     string
		urlString  string
		transport  newrelic.TransportType
	}{
		{
			testname:   "empty proxy request",
			input:      events.APIGatewayProxyRequest{},
			numHeaders: 0,
			method:     "",
			urlString:  "",
			transport:  newrelic.TransportUnknown,
		},
		{
			testname: "populated proxy request",
			input: events.APIGatewayProxyRequest{
				Headers: map[string]string{
					"x-forwarded-port":  "4000",
					"x-forwarded-proto": "HTTPS",
				},
				HTTPMethod: "GET",
				Path:       "the/path",
			},
			numHeaders: 2,
			method:     "GET",
			urlString:  "//:4000/the/path",
			transport:  newrelic.TransportHTTPS,
		},
		{
			testname:   "empty alb request",
			input:      events.ALBTargetGroupRequest{},
			numHeaders: 0,
			method:     "",
			urlString:  "",
			transport:  newrelic.TransportUnknown,
		},
		{
			testname: "populated alb request",
			input: events.ALBTargetGroupRequest{
				Headers: map[string]string{
					"x-forwarded-port":  "3000",
					"x-forwarded-proto": "HttP",
				},
				HTTPMethod: "GET",
				Path:       "the/path",
			},
			numHeaders: 2,
			method:     "GET",
			urlString:  "//:3000/the/path",
			transport:  newrelic.TransportHTTP,
		},
	}

	for _, tc := range testcases {
		req = eventWebRequest(tc.input)
		if req == nil {
			t.Error(tc.testname, "no request returned")
			continue
		}
		if h := req.Header(); len(h) != tc.numHeaders {
			t.Error(tc.testname, "header len mismatch", h, tc.numHeaders)
		}
		if u := req.URL().String(); u != tc.urlString {
			t.Error(tc.testname, "url mismatch", u, tc.urlString)
		}
		if m := req.Method(); m != tc.method {
			t.Error(tc.testname, "method mismatch", m, tc.method)
		}
		if tr := req.Transport(); tr != tc.transport {
			t.Error(tc.testname, "transport mismatch", tr, tc.transport)
		}
	}
}

func TestEventResponse(t *testing.T) {
	// First test a type that does not count as a web request.
	resp := eventResponse(22)
	if nil != resp {
		t.Error(resp)
	}

	testcases := []struct {
		testname   string
		input      interface{}
		numHeaders int
		code       int
	}{
		{
			testname:   "empty proxy response",
			input:      events.APIGatewayProxyResponse{},
			numHeaders: 0,
			code:       0,
		},
		{
			testname: "populated proxy response",
			input: events.APIGatewayProxyResponse{
				StatusCode: 200,
				Headers: map[string]string{
					"x-custom-header": "my custom header value",
				},
			},
			numHeaders: 1,
			code:       200,
		},
		{
			testname:   "empty alb response",
			input:      events.ALBTargetGroupResponse{},
			numHeaders: 0,
			code:       0,
		},
		{
			testname: "populated alb response",
			input: events.ALBTargetGroupResponse{
				StatusCode: 200,
				Headers: map[string]string{
					"x-custom-header": "my custom header value",
				},
			},
			numHeaders: 1,
			code:       200,
		},
	}

	for _, tc := range testcases {
		resp = eventResponse(tc.input)
		if resp == nil {
			t.Error(tc.testname, "no response returned")
			continue
		}
		if h := resp.Header(); len(h) != tc.numHeaders {
			t.Error(tc.testname, "header len mismatch", h, tc.numHeaders)
		}
		if resp.code != tc.code {
			t.Error(tc.testname, "status code mismatch", resp.code, tc.code)
		}
	}
}
