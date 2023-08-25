package nrfasthttp

import (
	"net/http"

	"github.com/valyala/fasthttp"
)

type fasthttpWrapperResponse struct {
	ctx *fasthttp.RequestCtx
}

func (rw fasthttpWrapperResponse) Header() http.Header {
	hdrs := http.Header{}
	rw.ctx.Request.Header.VisitAll(func(key, value []byte) {
		hdrs.Add(string(key), string(value))
	})
	return hdrs
}

func (rw fasthttpWrapperResponse) Write(b []byte) (int, error) {
	return rw.ctx.Write(b)
}

func (rw fasthttpWrapperResponse) WriteHeader(code int) {
	rw.ctx.SetStatusCode(code)
}
