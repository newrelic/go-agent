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

func startClientSegment(ctx context.Context, url string) (*newrelic.ExternalSegment, context.Context) {
	var seg *newrelic.ExternalSegment
	if txn := newrelic.FromContext(ctx); nil != txn {
		seg = newrelic.StartExternalSegment(txn, nil)
		seg.URL = url

		payload := txn.CreateDistributedTracePayload()
		if txt := payload.Text(); "" != txt {
			md, ok := metadata.FromOutgoingContext(ctx)
			if !ok {
				md = metadata.New(nil)
			}
			md.Set(newrelic.DistributedTracePayloadHeader, txt)
			ctx = metadata.NewOutgoingContext(ctx, md)
		}
	}

	return seg, ctx
}

// UnaryClientInterceptor TODO
func UnaryClientInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	seg, ctx := startClientSegment(ctx, getURL(method, cc.Target()))
	defer seg.End()
	return invoker(ctx, method, req, reply, cc, opts...)
}
