// Package attributes contains the names of the automatically captured
// attributes.  Attributes are key value pairs attached to transaction events,
// error events, and traced errors.  You may add your own attributes using the
// Transaction.AddAttribute method (see transaction.go).
//
// These attribute names are exposed here to facilitate configuration.
//
// For more information, see:
// https://docs.newrelic.com/docs/agents/manage-apm-agents/agent-metrics/agent-attributes
package attributes

// Attributes destined for Transaction Events and Errors:
const (
	// ResponseCode is the response status code for a web request.
	ResponseCode = "httpResponseCode"
	// RequestMethod is the request's method.
	RequestMethod = "request.method"
	// RequestAcceptHeader is the request's "Accept" header.
	RequestAcceptHeader = "request.headers.accept"
	// RequestContentType is the request's "Content-Type" header.
	RequestContentType = "request.headers.contentType"
	// RequestContentLength is the request's "Content-Length" header.
	RequestContentLength = "request.headers.contentLength"
	// RequestHeadersHost is the request's "Host" header.
	RequestHeadersHost = "request.headers.host"
	// ResponseHeadersContentType is the response "Content-Type" header.
	ResponseHeadersContentType = "response.headers.contentType"
	// ResponseHeadersContentLength is the response "Content-Length" header.
	ResponseHeadersContentLength = "response.headers.contentLength"
	// HostDisplayName contains the value of Config.HostDisplayName.
	HostDisplayName = "host.displayName"
)

// Attributes destined for Errors:
const (
	// RequestHeadersUserAgent is the request's "User-Agent" header.
	RequestHeadersUserAgent = "request.headers.User-Agent"
	// RequestHeadersReferer is the request's "Referer" header.  Query
	// string parameters are removed.
	RequestHeadersReferer = "request.headers.referer"
)
