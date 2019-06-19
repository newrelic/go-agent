package nrgrpc

import (
	"context"
	"net/http"
	"net/url"

	newrelic "github.com/newrelic/go-agent"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type serverRequest struct {
	header http.Header
	url    *url.URL
	method string
}

func (r serverRequest) Header() http.Header               { return r.header }
func (r serverRequest) URL() *url.URL                     { return r.url }
func (r serverRequest) Method() string                    { return r.method }
func (r serverRequest) Transport() newrelic.TransportType { return newrelic.TransportHTTP }

func newServerRequest(ctx context.Context, method string) serverRequest {
	var hdrs http.Header
	var target string
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		hdrs = make(http.Header, len(md))
		for k, vs := range md {
			for _, v := range vs {
				hdrs.Set(k, v)
				if ":authority" == k {
					target = v
				}
			}
		}
	}

	url, _ := url.Parse(getURL(method, target))

	return serverRequest{
		header: hdrs,
		url:    url,
		method: method,
	}
}

func UnaryServerInterceptor(app newrelic.Application) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		txn := app.StartTransaction(info.FullMethod, nil, nil)
		txn.SetWebRequest(newServerRequest(ctx, info.FullMethod))
		defer txn.End()

		ctx = newrelic.NewContext(ctx, txn)
		resp, err = handler(ctx, req)
		if err != nil {
			// TODO: NoticeError
		}
		// TODO: Save response code
		return
	}
}
