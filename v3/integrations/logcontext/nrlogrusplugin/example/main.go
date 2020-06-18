// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"os"
	"time"

	"github.com/newrelic/go-agent/v3/integrations/logcontext/nrlogrusplugin"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
	"github.com/sirupsen/logrus"
)

func doFunction2(txn *newrelic.Transaction, e *logrus.Entry) {
	defer txn.StartSegment("doFunction2").End()
	e.Error("In doFunction2")
}

func doFunction1(txn *newrelic.Transaction, e *logrus.Entry) {
	defer txn.StartSegment("doFunction1").End()
	e.Trace("In doFunction1")
	doFunction2(txn, e)
}

func main() {
	log := logrus.New()
	// To enable New Relic log decoration, use the
	// nrlogrusplugin.ContextFormatter{}
	log.SetFormatter(nrlogrusplugin.ContextFormatter{})
	log.SetLevel(logrus.TraceLevel)

	log.Debug("Logger created")

	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Logrus Log Decoration"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDistributedTracerEnabled(true),
	)
	if nil != err {
		log.Panic("Failed to create application", err)
	}

	log.Debug("Application created, waiting for connection")

	err = app.WaitForConnection(10 * time.Second)
	if nil != err {
		log.Panic("Failed to connect application", err)
	}
	log.Info("Application connected")
	defer app.Shutdown(10 * time.Second)

	log.Debug("Starting transaction now")
	txn := app.StartTransaction("main")

	// Add the transaction context to the logger. Only once this happens will
	// the logs be properly decorated with all required fields.
	e := log.WithContext(newrelic.NewContext(context.Background(), txn))

	doFunction1(txn, e)

	e.Info("Ending transaction")
	txn.End()
}
