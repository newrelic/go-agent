package nrgrpc

import (
	"context"
	"strings"

	newrelic "github.com/newrelic/go-agent"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func getURL(method, target string) string {
	if "" == target {
		return ""
	}
	var host string
	// target can be anything from
	// https://github.com/grpc/grpc/blob/master/doc/naming.md
	// see https://godoc.org/google.golang.org/grpc#DialContext
	if strings.HasPrefix(target, "unix:") {
		host = "localhost"
	} else {
		host = strings.TrimPrefix(target, "dns:///")
	}
	return "grpc://" + host + method
}

func UnaryClientInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	if txn := newrelic.FromContext(ctx); nil != txn {
		seg := newrelic.StartExternalSegment(txn, nil)
		seg.URL = getURL(method, cc.Target())

		payload := txn.CreateDistributedTracePayload()
		if txt := payload.Text(); "" != txt {
			md := metadata.Pairs(newrelic.DistributedTracePayloadHeader, txt)
			// TODO: test that headers are preserved
			ctx = metadata.NewOutgoingContext(ctx, md)
		}

		defer seg.End()
	}
	return invoker(ctx, method, req, reply, cc, opts...)
}
