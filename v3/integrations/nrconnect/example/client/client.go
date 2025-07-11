// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"connectrpc.com/connect"
	"github.com/newrelic/go-agent/v3/newrelic"
	"golang.org/x/net/http2"

	"github.com/newrelic/go-agent/v3/integrations/nrconnect"
	"github.com/newrelic/go-agent/v3/integrations/nrconnect/example/sampleapp"
	"github.com/newrelic/go-agent/v3/integrations/nrconnect/example/sampleapp/sampleappconnect"
)

func doUnaryUnary(ctx context.Context, client sampleappconnect.SampleApplicationClient) {
	resp, err := client.DoUnaryUnary(ctx, connect.NewRequest(&sampleapp.Message{Text: "Hello DoUnaryUnary"}))
	if err != nil {
		panic(err)
	}
	log.Println(resp.Msg.Text)
}

func doUnaryStream(ctx context.Context, client sampleappconnect.SampleApplicationClient) {
	stream, err := client.DoUnaryStream(ctx, connect.NewRequest(&sampleapp.Message{Text: "Hello DoUnaryStream"}))
	if err != nil {
		panic(err)
	}
	for stream.Receive() {
		msg := stream.Msg()
		log.Println(msg.Text)
	}
	if err := stream.Err(); err != nil {
		panic(err)
	}
}

func doStreamUnary(ctx context.Context, client sampleappconnect.SampleApplicationClient) {
	stream := client.DoStreamUnary(ctx)
	for i := 0; i < 3; i++ {
		if err := stream.Send(&sampleapp.Message{Text: "Hello DoStreamUnary"}); err != nil {
			panic(err)
		}
	}
	resp, err := stream.CloseAndReceive()
	if err != nil {
		panic(err)
	}
	log.Println(resp.Msg.Text)
}

func doStreamStream(ctx context.Context, client sampleappconnect.SampleApplicationClient) {
	stream := client.DoStreamStream(ctx)

	waitc := make(chan struct{})
	go func() {
		for {
			msg, err := stream.Receive()
			if errors.Is(err, io.EOF) {
				close(waitc)
				return
			}
			if err != nil {
				panic(err)
			}
			log.Println(msg.Text)
		}
	}()

	for i := 0; i < 3; i++ {
		if err := stream.Send(&sampleapp.Message{Text: "Hello DoStreamStream"}); err != nil {
			panic(err)
		}
	}

	if err := stream.CloseRequest(); err != nil {
		panic(err)
	}
	<-waitc
}

func main() {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Connect Client"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDebugLogger(os.Stdout),
	)
	if err != nil {
		panic(err)
	}
	defer app.Shutdown(10 * time.Second)

	app.WaitForConnection(10 * time.Second)
	txn := app.StartTransaction("main")
	defer txn.End()

	client := sampleappconnect.NewSampleApplicationClient(
		newInsecureClient(),
		"http://localhost:8080",
		connect.WithInterceptors(nrconnect.Interceptor(app)), connect.WithGRPC(),
	)

	ctx := newrelic.NewContext(context.Background(), txn)

	doUnaryUnary(ctx, client)
	doUnaryStream(ctx, client)
	doStreamUnary(ctx, client)
	doStreamStream(ctx, client)
}

// https://connectrpc.com/docs/go/common-errors/#client-missing-h2c-configuration
func newInsecureClient() *http.Client {
	return &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
		},
	}
}
