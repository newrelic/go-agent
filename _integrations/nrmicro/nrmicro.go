package nrmicro

import (
	"context"
	"net/url"
	"strings"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/metadata"
	"github.com/micro/go-micro/registry"
	newrelic "github.com/newrelic/go-agent"
)

type nrWrapper struct {
	client.Client
}

var addrMap = make(map[string]string)

func startExternal(ctx context.Context, procedure, host string) (context.Context, newrelic.ExternalSegment) {
	var seg newrelic.ExternalSegment
	if txn := newrelic.FromContext(ctx); nil != txn {
		seg = newrelic.ExternalSegment{
			StartTime: newrelic.StartSegmentNow(txn),
			Procedure: procedure,
			Library:   "Micro",
			Host:      host,
		}
		payload := txn.CreateDistributedTracePayload()
		if txt := payload.Text(); "" != txt {
			md, _ := metadata.FromContext(ctx)
			md = metadata.Copy(md)
			md[newrelic.DistributedTracePayloadHeader] = txt
			ctx = metadata.NewContext(ctx, md)
		}
	}
	return ctx, seg
}

func extractHost(addr string) string {
	if host, ok := addrMap[addr]; ok {
		return host
	}

	host := addr
	if strings.HasPrefix(host, "unix://") {
		host = "localhost"
	} else if u, err := url.Parse(host); nil == err {
		if "" != u.Host {
			host = u.Host
		} else {
			host = u.Path
		}
	}

	addrMap[addr] = host
	return host
}

func (n *nrWrapper) Publish(ctx context.Context, msg client.Message, opts ...client.PublishOption) error {
	host := extractHost(n.Options().Broker.Address())
	ctx, seg := startExternal(ctx, "Publish", host)
	defer seg.End()
	return n.Client.Publish(ctx, msg, opts...)
}

func (n *nrWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	ctx, seg := startExternal(ctx, req.Endpoint(), req.Service())
	defer seg.End()
	return n.Client.Call(ctx, req, rsp, opts...)
}

func ClientWrapper() client.Wrapper {
	return func(c client.Client) client.Client {
		return &nrWrapper{c}
	}
}

func CallWrapper() client.CallWrapper {
	return func(cf client.CallFunc) client.CallFunc {
		return func(ctx context.Context, node *registry.Node, req client.Request, rsp interface{}, opts client.CallOptions) error {
			var seg newrelic.ExternalSegment
			ctx, seg = startExternal(ctx, req.Endpoint(), req.Service())
			defer seg.End()
			return cf(ctx, node, req, rsp, opts)
		}
	}
}
