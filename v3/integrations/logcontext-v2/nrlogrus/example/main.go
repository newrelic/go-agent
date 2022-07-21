package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/newrelic/go-agent/v3/integrations/logcontext-v2/nrlogrus"
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
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Logrus Logs In Context Example"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigInfoLogger(os.Stdout),
		newrelic.ConfigAppLogForwardingEnabled(true),

		// If you wanted to forward your logs using a log forwarder instead
		// newrelic.ConfigAppLogDecoratingEnabled(true),
		// newrelic.ConfigAppLogForwardingEnabled(false),
	)
	if nil != err {
		log.Panic("Failed to create application", err)
	}

	log := logrus.New()
	log.SetLevel(logrus.TraceLevel)
	// Enable New Relic log decoration
	log.SetFormatter(nrlogrus.NewFormatter(app, &logrus.TextFormatter{}))
	log.Trace("waiting for connection to New Relic...")

	err = app.WaitForConnection(10 * time.Second)
	if nil != err {
		log.Panic("Failed to connect application", err)
	}
	defer app.Shutdown(10 * time.Second)
	log.Info("application connected to New Relic")
	log.Debug("Starting transaction now")
	txn := app.StartTransaction("main")

	// Add the transaction context to the logger. Only once this happens will
	// the logs be properly decorated with all required fields.
	e := log.WithContext(newrelic.NewContext(context.Background(), txn))

	doFunction1(txn, e)

	e.Info("Ending transaction")
	txn.End()
}
