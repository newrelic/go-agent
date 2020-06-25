// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// An application that illustrates Distributed Tracing or Cross Application
// Tracing when using NewRoundTripper.
package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/newrelic/go-agent"
)

func mustGetEnv(key string) string {
	if val := os.Getenv(key); "" != val {
		return val
	}
	panic(fmt.Sprintf("environment variable %s unset", key))
}

func doRequest(txn newrelic.Transaction) error {
	for _, addr := range []string{"segments", "mysql"} {
		url := fmt.Sprintf("http://localhost:8000/%s", addr)
		req, err := http.NewRequest("GET", url, nil)
		if nil != err {
			return err
		}
		client := &http.Client{}

		// Using NewRoundTripper automatically instruments all request
		// for Distributed Tracing and Cross Application Tracing.
		client.Transport = newrelic.NewRoundTripper(txn, nil)

		resp, err := client.Do(req)
		if nil != err {
			return err
		}
		fmt.Println("response code is", resp.StatusCode)
	}
	return nil
}

func main() {
	cfg := newrelic.NewConfig("Client App RoundTripper", mustGetEnv("NEW_RELIC_LICENSE_KEY"))
	cfg.Logger = newrelic.NewDebugLogger(os.Stdout)
	cfg.DistributedTracer.Enabled = true
	app, err := newrelic.NewApplication(cfg)
	if nil != err {
		fmt.Println(err)
		os.Exit(1)
	}

	// Wait for the application to connect.
	if err = app.WaitForConnection(5 * time.Second); nil != err {
		fmt.Println(err)
	}

	txn := app.StartTransaction("client-txn", nil, nil)
	err = doRequest(txn)
	if nil != err {
		txn.NoticeError(err)
	}
	txn.End()

	// Shut down the application to flush data to New Relic.
	app.Shutdown(10 * time.Second)
}
