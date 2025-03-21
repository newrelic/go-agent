package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/newrelic/go-agent/v3/integrations/logcontext-v2/nrslog"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func main() {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("slog example app"),
		newrelic.ConfigFromEnvironment(),
		newrelic.ConfigAppLogEnabled(true),
	)
	if err != nil {
		panic(err)
	}

	app.WaitForConnection(time.Second * 5)

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{})
	nrHandler := nrslog.WrapHandler(app, handler)
	log := slog.New(nrHandler)

	log.Info("I am a log message")

	txn := app.StartTransaction("example transaction")
	ctx := newrelic.NewContext(context.Background(), txn)

	log.InfoContext(ctx, "I am a log inside a transaction with custom attributes!",
		slog.String("foo", "bar"),
		slog.Int("answer", 42),
		slog.Any("some_map", map[string]interface{}{"a": 1.0, "b": 2}),
	)

	// pretend to do some work
	time.Sleep(500 * time.Millisecond)
	log.Warn("Uh oh, something important happened!")
	txn.End()

	log.Info("All Done!")

	app.Shutdown(time.Second * 10)
}
