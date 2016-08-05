package newrelic

// This file contains the names of the automatically captured attributes.
// Attributes are key value pairs attached to transaction events, error events,
// and traced errors.  You may add your own attributes using the
// Transaction.AddAttribute method (see transaction.go).
//
// These attribute names are exposed here to facilitate configuration.
//
// For more information, see:
// https://docs.newrelic.com/docs/agents/manage-apm-agents/agent-metrics/agent-attributes

// Attributes destined for Transaction Events and Errors:
const (
	// AttributeResponseCode is the response status code for a web request.
	AttributeResponseCode = "httpResponseCode"
	// AttributeRequestMethod is the request's method.
	AttributeRequestMethod = "request.method"
	// AttributeRequestAcceptHeader is the request's "Accept" header.
	AttributeRequestAcceptHeader = "request.headers.accept"
	// AttributeRequestContentType is the request's "Content-Type" header.
	AttributeRequestContentType = "request.headers.contentType"
	// AttributeRequestContentLength is the request's "Content-Length" header.
	AttributeRequestContentLength = "request.headers.contentLength"
	// AttributeRequestHeadersHost is the request's "Host" header.
	AttributeRequestHeadersHost = "request.headers.host"
	// AttributeResponseHeadersContentType is the response "Content-Type" header.
	AttributeResponseHeadersContentType = "response.headers.contentType"
	// AttributeResponseHeadersContentLength is the response "Content-Length" header.
	AttributeResponseHeadersContentLength = "response.headers.contentLength"
	// AttributeHostDisplayName contains the value of Config.HostDisplayName.
	AttributeHostDisplayName = "host.displayName"
)

// Attributes destined for Errors:
const (
	// AttributeRequestHeadersUserAgent is the request's "User-Agent" header.
	AttributeRequestHeadersUserAgent = "request.headers.User-Agent"
	// AttributeRequestHeadersReferer is the request's "Referer" header.  Query
	// string parameters are removed.
	AttributeRequestHeadersReferer = "request.headers.referer"
)
