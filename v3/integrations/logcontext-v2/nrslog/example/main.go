package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/newrelic/go-agent/v3/integrations/logcontext-v2/nrslog"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func main() {
	app, err := newrelic.NewApplication(
		newrelic.ConfigFromEnvironment(),
		newrelic.ConfigAppLogEnabled(true),
	)
	if err != nil {
		panic(err)
	}

	app.WaitForConnection(time.Second * 5)
	log := slog.New(nrslog.TextHandler(app, os.Stdout, &slog.HandlerOptions{}))

	log.Info("I am a log message")

	txn := app.StartTransaction("example transaction")
	txnLogger := nrslog.WithTransaction(txn, log)
	txnLogger.Info("I am a log inside a transaction")

	// pretend to do some work
	time.Sleep(500 * time.Millisecond)
	txnLogger.Warn("Uh oh, something important happened!")
	txn.End()

	log.Info("All Done!")

	app.Shutdown(time.Second * 10)
}
