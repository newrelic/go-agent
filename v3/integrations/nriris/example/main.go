// Copyright 2022 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"

	"github.com/kataras/iris/v12"
	"github.com/newrelic/go-agent/v3/integrations/nriris"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func v1login(ctx iris.Context)  { ctx.WriteString("v1 login") }
func v1submit(ctx iris.Context) { ctx.WriteString("v1 submit") }
func v1read(ctx iris.Context)   { ctx.WriteString("v1 read") }

func endpoint404(ctx iris.Context) {
	ctx.StatusCode(404)
	ctx.WriteString("returning 404")
}

func endpointChangeCode(ctx iris.Context) {
	ctx.StatusCode(404)
	ctx.StatusCode(200)
	ctx.WriteString("actually ok!")
}

func endpointResponseHeaders(ctx iris.Context) {
	ctx.ContentType("application/json")
	ctx.WriteString(`{"zip":"zap"}`)
}

func endpointNotFound(ctx iris.Context) {
	ctx.WriteString("there's no endpoint for that!")
}

func endpointAccessTransaction(ctx iris.Context) {
	txn := nriris.Transaction(ctx)
	txn.SetName("custom-name")
	ctx.WriteString("changed the name of the transaction!")
}

func main() {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Iris App"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		//	newrelic.ConfigDebugLogger(os.Stdout),
		newrelic.ConfigCodeLevelMetricsEnabled(true),
	)
	if nil != err {
		fmt.Println(err)
		os.Exit(1)
	}

	router := iris.New()
	router.Use(nriris.Middleware(app))

	router.Get("/404", endpoint404)
	router.Get("/change", endpointChangeCode)
	router.Get("/headers", endpointResponseHeaders)
	router.Get("/txn", endpointAccessTransaction)

	// Since the handler function name is used as the transaction name,
	// anonymous functions do not get usefully named.  We encourage
	// transforming anonymous functions into named functions.
	router.Get("/anon", func(ctx iris.Context) {
		ctx.WriteString("anonymous function handler")
	})

	v1 := router.Party("/v1")
	v1.Get("/login", v1login)
	v1.Get("/submit", v1submit)
	v1.Get("/read", v1read)

	router.OnErrorCode(404, endpointNotFound)

	router.Listen(":8000")
}
