package http

import "net/http"

type ResponseWrapper struct {
	*http.Response
}

func (r ResponseWrapper) Header() Header {
	return r.Response.Header
}

func (r ResponseWrapper) Code() int {
	return r.Response.StatusCode
}

func (r ResponseWrapper) Request() Request {
	return RequestWrapper{r.Response.Request}
}
