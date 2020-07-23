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
	case events.KinesisEvent:
		if len(v.Records) > 0 {
			return v.Records[0].EventSourceArn
		}
	case events.CodeCommitEvent:
		if len(v.Records) > 0 {
			return v.Records[0].EventSourceARN
		}
	case events.DynamoDBEvent:
		if len(v.Records) > 0 {
			return v.Records[0].EventSourceArn
		}
	case events.SQSEvent:
		if len(v.Records) > 0 {
			return v.Records[0].EventSourceARN
		}
	case events.S3Event:
		if len(v.Records) > 0 {
			return v.Records[0].S3.Bucket.Arn
		}
	case events.SNSEvent:
		if len(v.Records) > 0 {
			return v.Records[0].EventSubscriptionArn
		}
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
	case events.ALBTargetGroupRequest:
		// https://docs.aws.amazon.com/elasticloadbalancing/latest/application/lambda-functions.html#receive-event-from-load-balancer
		request.Method = r.HTTPMethod
		path = r.Path
		headers = r.Headers
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

	switch r := event.(type) {
	case events.APIGatewayProxyResponse:
		code = r.StatusCode
		headers = r.Headers
	case events.ALBTargetGroupResponse:
		code = r.StatusCode
		headers = r.Headers
	default:
		return nil
	}
	hdr := make(http.Header, len(headers))
	for k, v := range headers {
		hdr.Add(k, v)
	}
	return &response{
		code:   code,
		header: hdr,
	}
}
