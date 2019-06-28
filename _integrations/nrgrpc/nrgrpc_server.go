package nrgrpc

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	newrelic "github.com/newrelic/go-agent"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
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

func startTransaction(ctx context.Context, app newrelic.Application, fullMethod string) newrelic.Transaction {
	method := strings.TrimPrefix(fullMethod, "/")
	txn := app.StartTransaction(method, nil, nil)

	var hdrs http.Header
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		hdrs = make(http.Header, len(md))
		for k, vs := range md {
			for _, v := range vs {
				hdrs.Add(k, v)
			}
		}
	}

	target := hdrs.Get(":authority")
	url := getURL(method, target)

	txn.SetWebRequest(serverRequest{
		header: hdrs,
		url:    url,
		method: method,
	})

	return txn
}

// UnaryServerInterceptor TODO
func UnaryServerInterceptor(app newrelic.Application) grpc.UnaryServerInterceptor {
	if nil == app {
		return nil
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		txn := startTransaction(ctx, app, info.FullMethod)
		defer txn.End()

		ctx = newrelic.NewContext(ctx, txn)
		resp, err = handler(ctx, req)
		txn.WriteHeader(int(status.Code(err)))
		return
	}
}

type wrappedServerStream struct {
	grpc.ServerStream
	txn newrelic.Transaction
}

func (s wrappedServerStream) Context() context.Context {
	ctx := s.ServerStream.Context()
	return newrelic.NewContext(ctx, s.txn)
}

func newWrappedServerStream(stream grpc.ServerStream, txn newrelic.Transaction) grpc.ServerStream {
	return wrappedServerStream{
		ServerStream: stream,
		txn:          txn,
	}
}

// StreamServerInterceptor TODO
func StreamServerInterceptor(app newrelic.Application) grpc.StreamServerInterceptor {
	if nil == app {
		return nil
	}

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		txn := startTransaction(ss.Context(), app, info.FullMethod)
		defer txn.End()

		err := handler(srv, newWrappedServerStream(ss, txn))
		txn.WriteHeader(int(status.Code(err)))
		return err
	}
}
