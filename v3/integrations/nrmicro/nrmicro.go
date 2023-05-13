// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

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

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	"github.com/newrelic/go-agent/v3/newrelic"
)

type nrWrapper struct {
	client.Client
}

var addrMap = make(map[string]string)

func startExternal(ctx context.Context, procedure, host string) (context.Context, newrelic.ExternalSegment) {
	var seg newrelic.ExternalSegment
	if txn := newrelic.FromContext(ctx); nil != txn {
		seg = newrelic.ExternalSegment{
			StartTime: txn.StartSegmentNow(),
			Procedure: procedure,
			Library:   "Micro",
			Host:      host,
		}
		ctx = addDTPayloadToContext(ctx, txn)
	}
	return ctx, seg
}

func startMessage(ctx context.Context, topic string) (context.Context, *newrelic.MessageProducerSegment) {
	var seg *newrelic.MessageProducerSegment
	if txn := newrelic.FromContext(ctx); nil != txn {
		seg = &newrelic.MessageProducerSegment{
			StartTime:       txn.StartSegmentNow(),
			Library:         "Micro",
			DestinationType: newrelic.MessageTopic,
			DestinationName: topic,
		}
		ctx = addDTPayloadToContext(ctx, txn)
	}
	return ctx, seg
}

func addDTPayloadToContext(ctx context.Context, txn *newrelic.Transaction) context.Context {
	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)
	if len(hdrs) > 0 {
		md, _ := metadata.FromContext(ctx)
		md = metadata.Copy(md)
		for k := range hdrs {
			if v := hdrs.Get(k); v != "" {
				md[k] = v
			}
		}
		ctx = metadata.NewContext(ctx, md)
	}
	return ctx
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
	ctx, seg := startMessage(ctx, msg.Topic())
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

// ClientWrapper wraps a Micro `client.Client`
// (https://godoc.org/github.com/micro/go-micro/client#Client) instance.  External
// segments will be created for each call to the client's `Call`, `Publish`, or
// `Stream` methods.  The `newrelic.Transaction` must be put into the context
// using `newrelic.NewContext`
// (https://godoc.org/github.com/newrelic/go-agent#NewContext) when calling one
// of those methods.
func ClientWrapper() client.Wrapper {
	return func(c client.Client) client.Client {
		return &nrWrapper{c}
	}
}

// CallWrapper wraps the `Call` method of a Micro `client.Client`
// (https://godoc.org/github.com/micro/go-micro/client#Client) instance.
// External segments will be created for each call to the client's `Call`
// method.  The `newrelic.Transaction` must be put into the context using
// `newrelic.NewContext`
// (https://godoc.org/github.com/newrelic/go-agent#NewContext) when calling
// `Call`.
func CallWrapper() client.CallWrapper {
	return func(cf client.CallFunc) client.CallFunc {
		return func(ctx context.Context, node *registry.Node, req client.Request, rsp interface{}, opts client.CallOptions) error {
			ctx, seg := startExternal(ctx, req.Endpoint(), req.Service())
			defer seg.End()
			return cf(ctx, node, req, rsp, opts)
		}
	}
}

// HandlerWrapper wraps a Micro `server.Server`
// (https://godoc.org/github.com/micro/go-micro/server#Server) handler.
//
// This wrapper creates transactions for inbound calls.  The transaction is
// added to the call context and can be accessed in your method handlers using
// `newrelic.FromContext`
// (https://godoc.org/github.com/newrelic/go-agent#FromContext).
//
// When an error is returned and it is of type Micro `errors.Error`
// (https://godoc.org/github.com/micro/go-micro/errors#Error), the error that
// is recorded is based on the HTTP response code (found in the Code field).
// Values above 400 or below 100 that are not in the IgnoreStatusCodes
// (https://godoc.org/github.com/newrelic/go-agent#Config) configuration list
// are recorded as errors. A 500 response code and corresponding error is
// recorded when the error is of any other type. A 200 response code is
// recorded if no error is returned.
func HandlerWrapper(app *newrelic.Application) server.HandlerWrapper {
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
			txn.SetWebResponse(nil).WriteHeader(code)
			return err
		}
	}
}

// SubscriberWrapper wraps a Micro `server.Subscriber`
// (https://godoc.org/github.com/micro/go-micro/server#Subscriber) instance.
//
// This wrapper creates background transactions for inbound calls.  The
// transaction is added to the subscriber context and can be accessed in your
// subscriber handlers using `newrelic.FromContext`
// (https://godoc.org/github.com/newrelic/go-agent#FromContext).
//
// The attribute `"message.routingKey"` is added to the transaction and will
// appear on transaction events, transaction traces, error events, and error
// traces. It corresponds to the `server.Message`'s Topic
// (https://godoc.org/github.com/micro/go-micro/server#Message).
//
// If a Subscriber returns an error, it will be recorded and reported.
func SubscriberWrapper(app *newrelic.Application) server.SubscriberWrapper {
	return func(fn server.SubscriberFunc) server.SubscriberFunc {
		if app == nil {
			return fn
		}
		return func(ctx context.Context, m server.Message) (err error) {
			namer := internal.MessageMetricKey{
				Library:         "Micro",
				DestinationType: string(newrelic.MessageTopic),
				DestinationName: m.Topic(),
				Consumer:        true,
			}
			txn := app.StartTransaction(namer.Name())
			defer txn.End()
			integrationsupport.AddAgentAttribute(txn, newrelic.AttributeMessageRoutingKey, m.Topic(), nil)
			if md, ok := metadata.FromContext(ctx); ok {
				hdrs := http.Header{}
				for k, v := range md {
					hdrs.Set(k, v)
				}
				txn.AcceptDistributedTraceHeaders(newrelic.TransportHTTP, hdrs)
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

func startWebTransaction(ctx context.Context, app *newrelic.Application, req server.Request) *newrelic.Transaction {
	var hdrs http.Header
	if md, ok := metadata.FromContext(ctx); ok {
		hdrs = make(http.Header, len(md))
		for k, v := range md {
			hdrs.Add(k, v)
		}
	}
	txn := app.StartTransaction(req.Endpoint())
	u := &url.URL{
		Scheme: "micro",
		Host:   req.Service(),
		Path:   req.Endpoint(),
	}

	webReq := newrelic.WebRequest{
		Header:    hdrs,
		URL:       u,
		Method:    req.Method(),
		Transport: newrelic.TransportHTTP,
		Body:      req.Body,
		Type:      "HTTP",
	}
	txn.SetWebRequest(webReq)

	return txn
}
