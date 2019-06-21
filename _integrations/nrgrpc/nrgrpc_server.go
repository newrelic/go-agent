package nrgrpc

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	newrelic "github.com/newrelic/go-agent"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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

func newServerRequest(ctx context.Context, method string) serverRequest {
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

	return serverRequest{
		header: hdrs,
		url:    url,
		method: method,
	}
}

// translateCode translates a grpc response code to its corresponding http
// response code as per https://github.com/grpc/grpc/blob/master/doc/statuscodes.md
func translateCode(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.Canceled:
		return 499
	case codes.Unknown, codes.DataLoss, codes.Internal:
		return http.StatusInternalServerError
	case codes.InvalidArgument, codes.FailedPrecondition, codes.OutOfRange:
		return http.StatusBadRequest
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists, codes.Aborted:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	}
	return 0
}

// UnaryServerInterceptor TODO
func UnaryServerInterceptor(app newrelic.Application) grpc.UnaryServerInterceptor {
	if nil == app {
		return nil
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		method := strings.TrimPrefix(info.FullMethod, "/")
		txn := app.StartTransaction(method, nil, nil)
		txn.SetWebRequest(newServerRequest(ctx, method))
		defer txn.End()

		ctx = newrelic.NewContext(ctx, txn)
		resp, err = handler(ctx, req)
		txn.WriteHeader(translateCode(status.Code(err)))
		return
	}
}

func StreamServerInterceptor(app newrelic.Application) grpc.StreamServerInterceptor {
	if nil == app {
		return nil
	}
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, ss)
	}
}
