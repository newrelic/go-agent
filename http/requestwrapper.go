package http

import (
	"net/http"
	"net/url"
)

type RequestWrapper struct {
	*http.Request
}

func (r RequestWrapper) Header() Header {
	return r.Request.Header
}

func (r RequestWrapper) Method() string {
	return r.Request.Method
}

func (r RequestWrapper) URL() *url.URL {
	return r.Request.URL
}
