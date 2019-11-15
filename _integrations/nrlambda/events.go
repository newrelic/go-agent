package nrlambda

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	newrelic "github.com/newrelic/go-agent"
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

type webRequest struct {
	header    http.Header
	method    string
	u         *url.URL
	transport newrelic.TransportType
}

func (r webRequest) Header() http.Header               { return r.header }
func (r webRequest) URL() *url.URL                     { return r.u }
func (r webRequest) Method() string                    { return r.method }
func (r webRequest) Transport() newrelic.TransportType { return r.transport }

func eventWebRequest(event interface{}) newrelic.WebRequest {
	var path string
	var request webRequest
	var headers map[string]string

	switch r := event.(type) {
	case events.APIGatewayProxyRequest:
		request.method = r.HTTPMethod
		path = r.Path
		headers = r.Headers
	case events.ALBTargetGroupRequest:
		// https://docs.aws.amazon.com/elasticloadbalancing/latest/application/lambda-functions.html#receive-event-from-load-balancer
		request.method = r.HTTPMethod
		path = r.Path
		headers = r.Headers
	default:
		return nil
	}

	request.header = make(http.Header, len(headers))
	for k, v := range headers {
		request.header.Set(k, v)
	}

	var host string
	if port := request.header.Get("X-Forwarded-Port"); port != "" {
		host = ":" + port
	}
	request.u = &url.URL{
		Path: path,
		Host: host,
	}

	proto := strings.ToLower(request.header.Get("X-Forwarded-Proto"))
	switch proto {
	case "https":
		request.transport = newrelic.TransportHTTPS
	case "http":
		request.transport = newrelic.TransportHTTP
	default:
		request.transport = newrelic.TransportUnknown
	}

	return request
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
