// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"fmt"
	"os"
	"time"

	newrelic "github.com/newrelic/go-agent/v3/newrelic"
	"github.com/valyala/fasthttp"
)

func doRequest(txn *newrelic.Transaction) error {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI("http://localhost:8080/hello")
	req.Header.SetMethod("GET")

	ctx := &fasthttp.RequestCtx{}
	seg := newrelic.StartExternalSegmentFastHTTP(txn, ctx)
	defer seg.End()

	err := fasthttp.Do(req, resp)
	if err != nil {
		return err
	}

	fmt.Println("Response Code is ", resp.StatusCode())
	return nil

}

func main() {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Client App"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDebugLogger(os.Stdout),
		newrelic.ConfigDistributedTracerEnabled(true),
	)

	if err := app.WaitForConnection(5 * time.Second); nil != err {
		fmt.Println(err)
	}
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	txn := app.StartTransaction("client-txn")
	err = doRequest(txn)
	if err != nil {
		txn.NoticeError(err)
	}
	txn.End()

	// Shut down the application to flush data to New Relic.
	app.Shutdown(10 * time.Second)
}
