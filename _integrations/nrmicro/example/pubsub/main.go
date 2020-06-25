// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/micro/go-micro"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/_integrations/nrmicro"
	proto "github.com/newrelic/go-agent/_integrations/nrmicro/example/proto"
)

func mustGetEnv(key string) string {
	if val := os.Getenv(key); "" != val {
		return val
	}
	panic(fmt.Sprintf("environment variable %s unset", key))
}

func subEv(ctx context.Context, msg *proto.HelloRequest) error {
	fmt.Println("Message received from", msg.GetName())
	return nil
}

func publish(s micro.Service, app newrelic.Application) {
	c := s.Client()

	for range time.NewTicker(time.Second).C {
		txn := app.StartTransaction("publish", nil, nil)
		msg := c.NewMessage("example.topic.pubsub", &proto.HelloRequest{Name: "Sally"})
		ctx := newrelic.NewContext(context.Background(), txn)
		fmt.Println("Sending message")
		if err := c.Publish(ctx, msg); nil != err {
			log.Fatal(err)
		}
		txn.End()
	}
}

func main() {
	cfg := newrelic.NewConfig("Micro Pub/Sub", mustGetEnv("NEW_RELIC_LICENSE_KEY"))
	cfg.Logger = newrelic.NewDebugLogger(os.Stdout)
	app, err := newrelic.NewApplication(cfg)
	if nil != err {
		panic(err)
	}
	err = app.WaitForConnection(10 * time.Second)
	if nil != err {
		panic(err)
	}
	defer app.Shutdown(10 * time.Second)

	s := micro.NewService(
		micro.Name("go.micro.srv.pubsub"),
		// Add the New Relic wrapper to the client which will create
		// MessageProducerSegments for each Publish call.
		micro.WrapClient(nrmicro.ClientWrapper()),
		// Add the New Relic wrapper to the subscriber which will start a new
		// transaction for each Subscriber invocation.
		micro.WrapSubscriber(nrmicro.SubscriberWrapper(app)),
	)
	s.Init()

	go publish(s, app)

	micro.RegisterSubscriber("example.topic.pubsub", s.Server(), subEv)

	if err := s.Run(); err != nil {
		log.Fatal(err)
	}
}
