// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/micro/go-micro"
	"github.com/newrelic/go-agent/v3/integrations/nrmicro"
	proto "github.com/newrelic/go-agent/v3/integrations/nrmicro/example/proto"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)

func main() {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Micro Client"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDebugLogger(os.Stdout),
	)
	if nil != err {
		panic(err)
	}
	err = app.WaitForConnection(10 * time.Second)
	if nil != err {
		panic(err)
	}
	defer app.Shutdown(10 * time.Second)

	txn := app.StartTransaction("client")
	defer txn.End()

	service := micro.NewService(
		// Add the New Relic wrapper to the client which will create External
		// segments for each out going call.
		micro.WrapClient(nrmicro.ClientWrapper()),
	)
	service.Init()
	ctx := newrelic.NewContext(context.Background(), txn)
	c := proto.NewGreeterService("greeter", service.Client())

	rsp, err := c.Hello(ctx, &proto.HelloRequest{
		Name: "John",
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(rsp.Greeting)
}
