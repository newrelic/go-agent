package nrgrpc

import (
	"context"
	"io"
	"net/url"
	"strings"

	newrelic "github.com/newrelic/go-agent"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func getURL(method, target string) *url.URL {
	var host string
	// target can be anything from
	// https://github.com/grpc/grpc/blob/master/doc/naming.md
	// see https://godoc.org/google.golang.org/grpc#DialContext
	if strings.HasPrefix(target, "unix:") {
		host = "localhost"
	} else {
		host = strings.TrimPrefix(target, "dns:///")
	}
	return &url.URL{
		Scheme: "grpc",
		Host:   host,
		Path:   method,
	}
}

// startClientSegment starts an ExternalSegment and adds Distributed Trace
// headers to the outgoing grpc metadata in the context.
func startClientSegment(ctx context.Context, method, target string) (*newrelic.ExternalSegment, context.Context) {
	var seg *newrelic.ExternalSegment
	if txn := newrelic.FromContext(ctx); nil != txn {
		seg = newrelic.StartExternalSegment(txn, nil)

		method = strings.TrimPrefix(method, "/")
		seg.Host = getURL(method, target).Host
		seg.Library = "gRPC"
		seg.Procedure = method

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
	seg, ctx := startClientSegment(ctx, method, cc.Target())
	defer seg.End()
	return invoker(ctx, method, req, reply, cc, opts...)
}

type wrappedClientStream struct {
	grpc.ClientStream
	segment       *newrelic.ExternalSegment
	isUnaryServer bool
}

func (s wrappedClientStream) RecvMsg(m interface{}) error {
	err := s.ClientStream.RecvMsg(m)
	if err == io.EOF || s.isUnaryServer {
		s.segment.End()
	}
	return err
}

// StreamClientInterceptor TODO
func StreamClientInterceptor(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	seg, ctx := startClientSegment(ctx, method, cc.Target())
	s, err := streamer(ctx, desc, cc, method, opts...)
	if err != nil {
		return s, err
	}
	return wrappedClientStream{
		segment:       seg,
		ClientStream:  s,
		isUnaryServer: !desc.ServerStreams,
	}, nil
}
