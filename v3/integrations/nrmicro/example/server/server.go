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
	"github.com/newrelic/go-agent/v3/integrations/nrmicro"
	proto "github.com/newrelic/go-agent/v3/integrations/nrmicro/example/proto"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)

// Greeter is the server struct
type Greeter struct{}

// Hello is the method on the server being called
func (g *Greeter) Hello(ctx context.Context, req *proto.HelloRequest, rsp *proto.HelloResponse) error {
	name := req.GetName()
	txn := newrelic.FromContext(ctx)
	txn.AddAttribute("Name", name)
	fmt.Println("Request received from", name)
	rsp.Greeting = "Hello " + name
	return nil
}

func main() {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Micro Server"),
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

	service := micro.NewService(
		micro.Name("greeter"),
		// Add the New Relic middleware which will start a new transaction for
		// each Handler invocation.
		micro.WrapHandler(nrmicro.HandlerWrapper(app)),
	)

	service.Init()

	proto.RegisterGreeterHandler(service.Server(), new(Greeter))

	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
