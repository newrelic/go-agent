// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// An application that illustrates Distributed Tracing or Cross Application
// Tracing when using http.Server or similar frameworks.
package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	newrelic "github.com/newrelic/go-agent"
)

type handler struct {
	App newrelic.Application
}

func (h *handler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	// The call to StartTransaction must include the response writer and the
	// request.
	txn := h.App.StartTransaction("server-txn", writer, req)
	defer txn.End()

	if req.URL.String() == "/segments" {
		defer newrelic.StartSegment(txn, "f1").End()

		func() {
			defer newrelic.StartSegment(txn, "f2").End()

			io.WriteString(writer, "segments!")
			time.Sleep(10 * time.Millisecond)
		}()
		time.Sleep(10 * time.Millisecond)
	} else {
		// Transaction.WriteHeader has to be used instead of invoking
		// WriteHeader on the response writer.
		txn.WriteHeader(http.StatusNotFound)
	}
}

func mustGetEnv(key string) string {
	if val := os.Getenv(key); "" != val {
		return val
	}
	panic(fmt.Sprintf("environment variable %s unset", key))
}

func makeApplication() (newrelic.Application, error) {
	cfg := newrelic.NewConfig("HTTP Server App", mustGetEnv("NEW_RELIC_LICENSE_KEY"))
	cfg.Logger = newrelic.NewDebugLogger(os.Stdout)
	cfg.DistributedTracer.Enabled = true
	app, err := newrelic.NewApplication(cfg)

	if nil != err {
		return nil, err
	}

	// Wait for the application to connect.
	if err = app.WaitForConnection(5 * time.Second); nil != err {
		return nil, err
	}

	return app, nil
}

func main() {

	app, err := makeApplication()
	if nil != err {
		fmt.Println(err)
		os.Exit(1)
	}

	server := http.Server{
		Addr:    ":8000",
		Handler: &handler{App: app},
	}

	server.ListenAndServe()
}
