// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrmicro instruments https://github.com/micro/go-micro.
//
// This package can be used to instrument Micro Servers, Clients, Producers,
// and Subscribers.
//
// Micro Servers
//
// To instrument a Micro Server, use the `micro.WrapHandler`
// (https://godoc.org/github.com/micro/go-micro#WrapHandler) option with
// `nrmicro.HandlerWrapper` and your `newrelic.Application` and pass it to the
// `micro.NewService` method.  Example:
//
//	cfg := newrelic.NewConfig("Micro Server", os.Getenv("NEW_RELIC_LICENSE_KEY"))
//	app, _ := newrelic.NewApplication(cfg)
//	service := micro.NewService(
//		micro.WrapHandler(nrmicro.HandlerWrapper(app)),
//	)
//
// Alternatively, use the `server.WrapHandler`
// (https://godoc.org/github.com/micro/go-micro/server#WrapHandler) option with
// `nrmicro.HandlerWrapper` and your `newrelic.Application` and pass it to the
// `server.NewServer` method.  Example:
//
//	cfg := newrelic.NewConfig("Micro Server", os.Getenv("NEW_RELIC_LICENSE_KEY"))
//	app, _ := newrelic.NewApplication(cfg)
//	svr := server.NewServer(
//		server.WrapHandler(nrmicro.HandlerWrapper(app)),
//	)
//
// If more than one wrapper is passed to `micro.WrapHandler` or
// `server.WrapHandler` as a list, be sure that the `nrmicro.HandlerWrapper` is
// first in this list.
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
//
// Full server example:
// https://github.com/newrelic/go-agent/blob/master/_integrations/nrmicro/example/server/server.go
//
// Micro Clients
//
// There are three different ways to instrument a Micro Client and create
// External segments for `Call`, `Publish`, and `Stream` methods.
//
// No matter which way the Micro `client.Client` is wrapped, all calls to
// `Client.Call`, `Client.Publish`, or `Client.Stream` must be done with a
// context which contains a `newrelic.Transaction`.
//
//	ctx = newrelic.NewContext(ctx, txn)
//	err := cli.Call(ctx, req, &rsp)
//
// 1. The first option is to wrap the `Call`, `Publish`, and `Stream` methods
// on a client using the `micro.WrapClient`
// (https://godoc.org/github.com/micro/go-micro#WrapClient) option with
// `nrmicro.ClientWrapper` and pass it to the `micro.NewService` method.  If
// more than one wrapper is passed to `micro.WrapClient`, ensure that the
// `nrmicro.ClientWrapper` is the first in the list.  `ExternalSegment`s will be
// created each time a `Call` or `Stream` method is called on the
// client.  `MessageProducerSegment`s will be created each time a `Publish`
// method is called on the client.  Example:
//
//	service := micro.NewService(
//		micro.WrapClient(nrmicro.ClientWrapper()),
//	)
//	cli := service.Client()
//
// It is also possible to use the `client.Wrap`
// (https://godoc.org/github.com/micro/go-micro/client#Wrap) option with
// `nrmicro.ClientWrapper` and pass it to the `client.NewClient` method to
// achieve the same result.
//
//	cli := client.NewClient(
//		client.Wrap(nrmicro.ClientWrapper()),
//	)
//
// 2. The second option is to wrap just the `Call` method on a client using the
// `micro.WrapCall` (https://godoc.org/github.com/micro/go-micro#WrapCall)
// option with `nrmicro.CallWrapper` and pass it to the `micro.NewService`
// method.  If more than one wrapper is passed to `micro.WrapCall`, ensure that
// the `nrmicro.CallWrapper` is the first in the list.  External segments will
// be created each time a `Call` method is called on the client.  Example:
//
//	service := micro.NewService(
//		micro.WrapCall(nrmicro.CallWrapper()),
//	)
//	cli := service.Client()
//
// It is also possible to use the `client.WrapCall`
// (https://godoc.org/github.com/micro/go-micro/client#WrapCall) option with
// `nrmicro.CallWrapper` and pass it to the `client.NewClient` method to
// achieve the same result.
//
//	cli := client.NewClient(
//		client.WrapCall(nrmicro.CallWrapper()),
//	)
//
// 3. The third option is to wrap the Micro Client directly using
// `nrmicro.ClientWrapper`.  `ExternalSegment`s will be created each time a
// `Call` or `Stream` method is called on the client.
// `MessageProducerSegment`s will be created each time a `Publish` method is
// called on the client.  Example:
//
//	cli := client.NewClient()
//	cli = nrmicro.ClientWrapper()(cli)
//
// Full client example:
// https://github.com/newrelic/go-agent/blob/master/_integrations/nrmicro/example/client/client.go
//
// Micro Producers
//
// To instrument a Micro Producer, wrap the Micro Client using the
// `nrmico.ClientWrapper` as described in option 1 or 3 above.
// `MessageProducerSegment`s will be created each time a `Publish` method is
// called on the client.  Be sure the context passed to the `Publish` method
// contains a `newrelic.Transaction`.
//
//	service := micro.NewService(
//		micro.WrapClient(nrmicro.ClientWrapper()),
//	)
//	cli := service.Client()
//
//	// Add the transaction to the context
//	ctx := newrelic.NewContext(context.Background(), txn)
//	msg := cli.NewMessage("my.example.topic", "hello world")
//	err := cli.Publish(ctx, msg)
//
// Full Publisher/Subscriber example:
// https://github.com/newrelic/go-agent/blob/master/_integrations/nrmicro/example/pubsub/main.go
//
// Micro Subscribers
//
// To instrument a Micro Subscriber use the `micro.WrapSubscriber`
// (https://godoc.org/github.com/micro/go-micro#WrapSubscriber) option with
// `nrmicro.SubscriberWrapper` and your `newrelic.Application` and pass it to
// the `micro.NewService` method.  Example:
//
//	cfg := newrelic.NewConfig("Micro Subscriber", os.Getenv("NEW_RELIC_LICENSE_KEY"))
//	app, _ := newrelic.NewApplication(cfg)
//	service := micro.NewService(
//		micro.WrapSubscriber(nrmicro.SubscriberWrapper(app)),
//	)
//
// Alternatively, use the `server.WrapSubscriber`
// (https://godoc.org/github.com/micro/go-micro/server#WrapSubscriber) option
// with `nrmicro.SubscriberWrapper` and your `newrelic.Application` and pass it
// to the `server.NewServer` method.  Example:
//
//	cfg := newrelic.NewConfig("Micro Subscriber", os.Getenv("NEW_RELIC_LICENSE_KEY"))
//	app, _ := newrelic.NewApplication(cfg)
//	svr := server.NewServer(
//		server.WrapSubscriber(nrmicro.SubscriberWrapper(app)),
//	)
//
// If more than one wrapper is passed to `micro.WrapSubscriber` or
// `server.WrapSubscriber` as a list, be sure that the `nrmicro.SubscriberWrapper` is
// first in this list.
//
// This wrapper creates background transactions for inbound calls.  The
// transaction is added to the subscriber context and can be accessed in your
// subscriber handlers using `newrelic.FromContext`.
//
// If a Subscriber returns an error, it will be recorded and reported.
//
// Full Publisher/Subscriber example:
// https://github.com/newrelic/go-agent/blob/master/_integrations/nrmicro/example/pubsub/main.go
package nrmicro

import "github.com/newrelic/go-agent/internal"

func init() { internal.TrackUsage("integration", "framework", "micro") }
