// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// An application that illustrates Distributed Tracing with Logs-in-Context
// when using http.Server or similar frameworks.
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/newrelic/go-agent/v3/integrations/logcontext-v2/nrlogrus"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
	"github.com/sirupsen/logrus"
)

type handler struct {
	App *newrelic.Application
}

func (h *handler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	// The call to StartTransaction must include the response writer and the
	// request.
	txn := h.App.StartTransaction("server-txn")
	defer txn.End()

	txnLogger := logrus.WithContext(newrelic.NewContext(context.Background(), txn))

	writer = txn.SetWebResponse(writer)
	txn.SetWebRequestHTTP(req)

	if req.URL.String() == "/segments" {
		defer txn.StartSegment("f1").End()

		txnLogger.Infof("/segments just started")

		func() {
			defer txn.StartSegment("f2").End()

			io.WriteString(writer, "segments!")
			time.Sleep(10 * time.Millisecond)

			txnLogger.Infof("segment func just about to complete")
		}()
		time.Sleep(10 * time.Millisecond)
	} else {
		// Transaction.WriteHeader has to be used instead of invoking
		// WriteHeader on the response writer.
		writer.WriteHeader(http.StatusNotFound)
	}
	txnLogger.Infof("handler completing")
}

func makeApplication() (*newrelic.Application, error) {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("HTTP Server App"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
	)
	if nil != err {
		return nil, err
	}
	nrlogrusFormatter := nrlogrus.NewFormatter(app, &logrus.TextFormatter{})
	logrus.SetFormatter(nrlogrusFormatter)
	// Alternatively and if preferred, create a new logger and use that logger
	// for logging with
	// log := logrus.New()
	// log.SetFormatter(nrlogrusFormatter)

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

	logrus.Infof("Application Starting")

	server := http.Server{
		Addr:    ":8000",
		Handler: &handler{App: app},
	}

	server.ListenAndServe()
}
