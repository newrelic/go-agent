package newrelic

import (
	"net/http"

	nrhttp "github.com/newrelic/go-agent/http"
)

type simpleHttpResponse struct {
	header     http.Header
	statusCode int
	request    nrhttp.Request
}

func (r simpleHttpResponse) Header() nrhttp.Header {
	return r.header
}

func (r simpleHttpResponse) Code() int {
	return r.statusCode
}

func (r simpleHttpResponse) Request() nrhttp.Request {
	return r.request
}
