package nrmicro

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/errors"
	"github.com/micro/go-micro/metadata"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/server"
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

func (n *nrWrapper) Stream(ctx context.Context, req client.Request, opts ...client.CallOption) (client.Stream, error) {
	ctx, seg := startExternal(ctx, req.Endpoint(), req.Service())
	defer seg.End()
	return n.Client.Stream(ctx, req, opts...)
}

func (n *nrWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	ctx, seg := startExternal(ctx, req.Endpoint(), req.Service())
	defer seg.End()
	return n.Client.Call(ctx, req, rsp, opts...)
}

// ClientWrapper TODO
func ClientWrapper() client.Wrapper {
	return func(c client.Client) client.Client {
		return &nrWrapper{c}
	}
}

// CallWrapper TODO
func CallWrapper() client.CallWrapper {
	return func(cf client.CallFunc) client.CallFunc {
		return func(ctx context.Context, node *registry.Node, req client.Request, rsp interface{}, opts client.CallOptions) error {
			ctx, seg := startExternal(ctx, req.Endpoint(), req.Service())
			defer seg.End()
			return cf(ctx, node, req, rsp, opts)
		}
	}
}

// HandlerWrapper TODO
func HandlerWrapper(app newrelic.Application) server.HandlerWrapper {
	return func(fn server.HandlerFunc) server.HandlerFunc {
		if app == nil {
			return fn
		}
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			txn := startWebTransaction(ctx, app, req)
			defer txn.End()
			err := fn(newrelic.NewContext(ctx, txn), req, rsp)
			var code int
			if err != nil {
				if t, ok := err.(*errors.Error); ok {
					code = int(t.Code)
				} else {
					code = 500
				}
			} else {
				code = 200
			}
			txn.WriteHeader(code)
			return err
		}
	}
}

// SubscriberWrapper TODO
func SubscriberWrapper(app newrelic.Application) server.SubscriberWrapper {
	return func(fn server.SubscriberFunc) server.SubscriberFunc {
		if app == nil {
			return fn
		}
		return func(ctx context.Context, m server.Message) (err error) {
			txn := app.StartTransaction(m.Topic(), nil, nil)
			defer txn.End()
			md, ok := metadata.FromContext(ctx)
			if ok {
				txn.AcceptDistributedTracePayload(newrelic.TransportHTTP, md[newrelic.DistributedTracePayloadHeader])
			}
			ctx = newrelic.NewContext(ctx, txn)
			err = fn(ctx, m)
			if err != nil {
				txn.NoticeError(err)
			}
			return err
		}
	}
}

func startWebTransaction(ctx context.Context, app newrelic.Application, req server.Request) newrelic.Transaction {
	var hdrs http.Header
	if md, ok := metadata.FromContext(ctx); ok {
		hdrs = make(http.Header, len(md))
		for k, v := range md {
			hdrs.Add(k, v)
		}
	}
	txn := app.StartTransaction(req.Endpoint(), nil, nil)
	u := &url.URL{
		Scheme: "micro",
		Host:   req.Service(),
		Path:   req.Endpoint(),
	}

	webReq := newrelic.NewStaticWebRequest(hdrs, u, req.Method(), newrelic.TransportHTTP)
	txn.SetWebRequest(webReq)

	return txn
}
