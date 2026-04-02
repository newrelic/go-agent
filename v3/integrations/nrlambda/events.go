// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrlambda

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)

func getEventSourceARN(event interface{}) string {
	switch v := event.(type) {
	case events.KinesisFirehoseEvent:
		return v.DeliveryStreamArn
	case *events.KinesisFirehoseEvent:
		return getEventSourceARN(safeDereference(v))

	case events.KinesisEvent:
		if len(v.Records) > 0 {
			return v.Records[0].EventSourceArn
		}
	case *events.KinesisEvent:
		return getEventSourceARN(safeDereference(v))

	case events.CodeCommitEvent:
		if len(v.Records) > 0 {
			return v.Records[0].EventSourceARN
		}
	case *events.CodeCommitEvent:
		return getEventSourceARN(safeDereference(v))

	case events.DynamoDBEvent:
		if len(v.Records) > 0 {
			return v.Records[0].EventSourceArn
		}
	case *events.DynamoDBEvent:
		return getEventSourceARN(safeDereference(v))

	case events.SQSEvent:
		if len(v.Records) > 0 {
			return v.Records[0].EventSourceARN
		}
	case *events.SQSEvent:
		return getEventSourceARN(safeDereference(v))

	case events.S3Event:
		if len(v.Records) > 0 {
			return v.Records[0].S3.Bucket.Arn
		}
	case *events.S3Event:
		return getEventSourceARN(safeDereference(v))

	case events.SNSEvent:
		if len(v.Records) > 0 {
			return v.Records[0].EventSubscriptionArn
		}
	case *events.SNSEvent:
		return getEventSourceARN(safeDereference(v))
	}

	return ""
}

func eventWebRequest(event interface{}) *newrelic.WebRequest {
	var path string
	var request newrelic.WebRequest
	var headers map[string]string

	switch r := event.(type) {
	case events.APIGatewayProxyRequest:
		request.Method = r.HTTPMethod
		path = r.Path
		headers = r.Headers
	case *events.APIGatewayProxyRequest:
		return eventWebRequest(safeDereference(r))

	case events.ALBTargetGroupRequest:
		request.Method = r.HTTPMethod
		path = r.Path
		headers = r.Headers
	case *events.ALBTargetGroupRequest:
		return eventWebRequest(safeDereference(r))

	case events.LambdaFunctionURLRequest:
		request.Method = r.RequestContext.HTTP.Method
		path = r.RequestContext.HTTP.Path
		headers = r.Headers
	case *events.LambdaFunctionURLRequest:
		return eventWebRequest(safeDereference(r))

	default:
		return nil
	}

	request.Header = make(http.Header, len(headers))
	for k, v := range headers {
		request.Header.Set(k, v)
	}

	var host string
	if port := request.Header.Get("X-Forwarded-Port"); port != "" {
		host = ":" + port
	}
	request.URL = &url.URL{
		Path: path,
		Host: host,
	}

	proto := strings.ToLower(request.Header.Get("X-Forwarded-Proto"))
	switch proto {
	case "https":
		request.Transport = newrelic.TransportHTTPS
	case "http":
		request.Transport = newrelic.TransportHTTP
	default:
		request.Transport = newrelic.TransportUnknown
	}

	return &request
}

func eventResponse(event interface{}) *response {
	var code int
	var headers map[string]string
	var multiValueHeaders map[string][]string

	switch r := event.(type) {
	case events.APIGatewayProxyResponse:
		code = r.StatusCode
		headers = r.Headers
		multiValueHeaders = r.MultiValueHeaders
	case *events.APIGatewayProxyResponse:
		return eventResponse(safeDereference(r))

	case events.ALBTargetGroupResponse:
		code = r.StatusCode
		headers = r.Headers
		multiValueHeaders = r.MultiValueHeaders
	case *events.ALBTargetGroupResponse:
		return eventResponse(safeDereference(r))

	case events.LambdaFunctionURLResponse:
		code = r.StatusCode
		headers = r.Headers
		multiValueHeaders = nil // LambdaFunctionURLResponse does not currently support multi-value headers
	case *events.LambdaFunctionURLResponse:
		return eventResponse(safeDereference(r))

	case events.LambdaFunctionURLStreamingResponse:
		code = r.StatusCode
		headers = r.Headers
		multiValueHeaders = nil // LambdaFunctionURLStreamingResponse does not currently support multi-value headers
	case *events.LambdaFunctionURLStreamingResponse:
		return eventResponse(safeDereference(r))

	default:
		return nil
	}

	// https://docs.aws.amazon.com/apigateway/latest/developerguide/set-up-lambda-proxy-integrations.html#api-gateway-simple-proxy-for-lambda-output-format
	// 	"If you specify values for both headers and multiValueHeaders, API Gateway merges them into a single list.
	// 	If the same key-value pair is specified in both, only the values from multiValueHeaders will appear in the merged list."
	//
	// To match API Gateway's behavior, copy headers and then multiValueHeaders, so the latter takes priority for a given header key.
	hdr := make(http.Header, len(headers)+len(multiValueHeaders))
	for k, v := range headers {
		hdr.Set(k, v)
	}
	for k, v := range multiValueHeaders {
		if len(v) == 0 {
			continue
		}
		hdr.Set(k, v[0])
	}

	return &response{
		code:   code,
		header: hdr,
	}
}

func safeDereference[T any](p *T) T {
	if p == nil {
		var z T
		return z
	}
	return *p
}
