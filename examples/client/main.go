// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

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
	req, err := http.NewRequest("GET", "http://localhost:8000/segments", nil)
	if nil != err {
		return err
	}
	client := &http.Client{}
	seg := newrelic.StartExternalSegment(txn, req)
	defer seg.End()
	resp, err := client.Do(req)
	if nil != err {
		return err
	}
	fmt.Println("response code is", resp.StatusCode)
	return nil
}

func main() {
	cfg := newrelic.NewConfig("Client App", mustGetEnv("NEW_RELIC_LICENSE_KEY"))
	cfg.Logger = newrelic.NewDebugLogger(os.Stdout)
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
