// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrlambda

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
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
		{Name: "*SQSEvent nil", Input: (*events.SQSEvent)(nil), Arn: ""},
		{Name: "*SQSEvent", Input: &events.SQSEvent{
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
		{Name: "*SNSEvent nil", Input: (*events.SNSEvent)(nil), Arn: ""},
		{Name: "*SNSEvent", Input: &events.SNSEvent{
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
		{Name: "*S3Event nil", Input: (*events.S3Event)(nil), Arn: ""},
		{Name: "*S3Event", Input: &events.S3Event{
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
		{Name: "*DynamoDBEvent nil", Input: (*events.DynamoDBEvent)(nil), Arn: ""},
		{Name: "*DynamoDBEvent", Input: &events.DynamoDBEvent{
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
		{Name: "*CodeCommitEvent nil", Input: (*events.CodeCommitEvent)(nil), Arn: ""},
		{Name: "*CodeCommitEvent", Input: &events.CodeCommitEvent{
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
		{Name: "*KinesisEvent nil", Input: (*events.KinesisEvent)(nil), Arn: ""},
		{Name: "*KinesisEvent", Input: &events.KinesisEvent{
			Records: []events.KinesisEventRecord{{
				EventSourceArn: "ARN",
			}},
		}, Arn: "ARN"},

		{Name: "KinesisFirehoseEvent empty", Input: events.KinesisFirehoseEvent{}, Arn: ""},
		{Name: "KinesisFirehoseEvent", Input: events.KinesisFirehoseEvent{
			DeliveryStreamArn: "ARN",
		}, Arn: "ARN"},
		{Name: "*KinesisFirehoseEvent nil", Input: (*events.KinesisFirehoseEvent)(nil), Arn: ""},
		{Name: "*KinesisFirehoseEvent", Input: &events.KinesisFirehoseEvent{
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
			testname:   "empty APIGatewayProxyRequest",
			input:      events.APIGatewayProxyRequest{},
			numHeaders: 0,
			method:     "",
			urlString:  "",
			transport:  newrelic.TransportUnknown,
		},
		{
			testname: "populated APIGatewayProxyRequest",
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
			testname:   "nil *APIGatewayProxyRequest",
			input:      (*events.APIGatewayProxyRequest)(nil),
			numHeaders: 0,
			method:     "",
			urlString:  "",
			transport:  newrelic.TransportUnknown,
		},
		{
			testname: "populated *APIGatewayProxyRequest",
			input: &events.APIGatewayProxyRequest{
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
			testname:   "empty ALBTargetGroupRequest",
			input:      events.ALBTargetGroupRequest{},
			numHeaders: 0,
			method:     "",
			urlString:  "",
			transport:  newrelic.TransportUnknown,
		},
		{
			testname: "populated ALBTargetGroupRequest",
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
		{
			testname:   "nil *ALBTargetGroupRequest",
			input:      (*events.ALBTargetGroupRequest)(nil),
			numHeaders: 0,
			method:     "",
			urlString:  "",
			transport:  newrelic.TransportUnknown,
		},
		{
			testname: "populated *ALBTargetGroupRequest",
			input: &events.ALBTargetGroupRequest{
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
		if h := req.Header; len(h) != tc.numHeaders {
			t.Error(tc.testname, "header len mismatch", h, tc.numHeaders)
		}
		if u := req.URL.String(); u != tc.urlString {
			t.Error(tc.testname, "url mismatch", u, tc.urlString)
		}
		if m := req.Method; m != tc.method {
			t.Error(tc.testname, "method mismatch", m, tc.method)
		}
		if tr := req.Transport; tr != tc.transport {
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

	runTest := func(t *testing.T, input any, want *response) {
		resp = eventResponse(input)
		if resp == nil {
			t.Fatal("no response returned")
		}

		if !reflect.DeepEqual(resp.header, want.header) {
			t.Error("header mismatch", resp.header, want.header)
		}

		if resp.code != want.code {
			t.Error("status code mismatch", resp.code, want.code)
		}
	}

	testcases := []struct {
		testname          string
		headers           map[string]string
		multiValueHeaders map[string][]string
		wantHeaders       http.Header
	}{
		{
			testname: "with Headers",
			headers: map[string]string{
				"x-custom-header": "my custom header value",
			},
			wantHeaders: http.Header{
				"X-Custom-Header": {"my custom header value"},
			},
		},
		{
			testname: "with MultiValueHeaders",
			multiValueHeaders: map[string][]string{
				"x-custom-header": {"my custom header value", "another value"},
			},
			wantHeaders: http.Header{
				"X-Custom-Header": {"my custom header value"},
			},
		},
		{
			testname: "with Headers and MultiValueHeaders",
			headers: map[string]string{
				"x-custom-header": "my custom header value",
			},
			multiValueHeaders: map[string][]string{
				"x-custom-header-2": {"my second custom header value"},
				"empty-header":      {},
			},
			wantHeaders: http.Header{
				"X-Custom-Header":   {"my custom header value"},
				"X-Custom-Header-2": {"my second custom header value"},
			},
		},
		{
			testname: "with overlapping Headers and MultiValueHeaders",
			headers: map[string]string{
				"x-custom-header": "my custom header value",
			},
			multiValueHeaders: map[string][]string{
				"X-CUSTOM-HEADER": {"my second custom header value"},
			},
			wantHeaders: http.Header{
				"X-Custom-Header": {"my second custom header value"},
			},
		},
	}

	t.Run("APIGatewayProxyResponse", func(t *testing.T) {
		t.Run("empty", func(t *testing.T) {
			input := events.APIGatewayProxyResponse{}
			runTest(t, input, &response{
				header: http.Header{},
				code:   0,
			})
		})

		for _, tc := range testcases {
			t.Run(tc.testname, func(t *testing.T) {
				input := events.APIGatewayProxyResponse{
					StatusCode:        200,
					Headers:           tc.headers,
					MultiValueHeaders: tc.multiValueHeaders,
				}
				runTest(t, input, &response{
					header: tc.wantHeaders,
					code:   200,
				})
			})
		}
	})
	t.Run("*APIGatewayProxyResponse", func(t *testing.T) {
		t.Run("nil", func(t *testing.T) {
			input := (*events.APIGatewayProxyResponse)(nil)
			runTest(t, input, &response{
				header: http.Header{},
				code:   0,
			})
		})

		for _, tc := range testcases {
			t.Run(tc.testname, func(t *testing.T) {
				input := &events.APIGatewayProxyResponse{
					StatusCode:        200,
					Headers:           tc.headers,
					MultiValueHeaders: tc.multiValueHeaders,
				}
				runTest(t, input, &response{
					header: tc.wantHeaders,
					code:   200,
				})
			})
		}
	})

	t.Run("ALBTargetGroupResponse", func(t *testing.T) {
		t.Run("empty", func(t *testing.T) {
			input := events.ALBTargetGroupResponse{}
			runTest(t, input, &response{
				header: http.Header{},
				code:   0,
			})
		})

		for _, tc := range testcases {
			t.Run(tc.testname, func(t *testing.T) {
				input := events.ALBTargetGroupResponse{
					StatusCode:        200,
					Headers:           tc.headers,
					MultiValueHeaders: tc.multiValueHeaders,
				}
				runTest(t, input, &response{
					header: tc.wantHeaders,
					code:   200,
				})
			})
		}
	})
	t.Run("*ALBTargetGroupResponse", func(t *testing.T) {
		t.Run("nil", func(t *testing.T) {
			input := (*events.ALBTargetGroupResponse)(nil)
			runTest(t, input, &response{
				header: http.Header{},
				code:   0,
			})
		})

		for _, tc := range testcases {
			t.Run(tc.testname, func(t *testing.T) {
				input := &events.ALBTargetGroupResponse{
					StatusCode:        200,
					Headers:           tc.headers,
					MultiValueHeaders: tc.multiValueHeaders,
				}
				runTest(t, input, &response{
					header: tc.wantHeaders,
					code:   200,
				})
			})
		}
	})
}
