package main

import (
	"os"
	"time"

	"github.com/newrelic/go-agent/v3/integrations/logcontext-v2/zerologWriter"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/rs/zerolog"
)

func main() {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("nrwriter log writer example"),
		newrelic.ConfigFromEnvironment(),
		newrelic.ConfigInfoLogger(os.Stdout),
		newrelic.ConfigAppLogDecoratingEnabled(true),
	)
	if err != nil {
		panic(err)
	}

	app.WaitForConnection(5 * time.Second)

	writer := zerologWriter.New(os.Stdout, app)
	logger := zerolog.New(writer)

	logger.Print("Application connected to New Relic.")

	txnName := "Example Transaction"
	txn := app.StartTransaction(txnName)

	// Always create a new logger in order to avoid changing the context of the logger for
	// other threads that may be logging outside of this transaction
	txnLogger := logger.Output(writer.WithTransaction(txn))
	txnLogger.Printf("In transaction %s.", txnName)

	// simulate doing something
	time.Sleep(time.Microsecond * 100)

	txnLogger.Printf("Ending transaction %s.", txnName)
	txn.End()

	app.Shutdown(10 * time.Second)
}
