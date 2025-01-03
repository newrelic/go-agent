package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/newrelic/go-agent/v3/integrations/logcontext-v2/nrslog"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func logWithGroup(logger *slog.Logger) {
	logger.WithGroup("program_info").With("pid", os.Getpid()).Info("log message with group and attribute")
}

func logWithGroupAndAttributes(logger *slog.Logger) {
	logger.Info("I am a log group inside a log message",
		slog.Group("logGroup",
			slog.String("key1", "val"),
			slog.Int("key2", 1),
		),
	)

}
func logWithLogAttrs(app *newrelic.Application, logger *slog.Logger) {
	txn := app.StartTransaction("test-LogAttrs")
	ctx := newrelic.NewContext(context.Background(), txn)

	logger.WithGroup("s").LogAttrs(ctx, slog.LevelInfo, "Log message with log attributes", slog.Int("a", 1), slog.Int("b", 2))
	txn.End()

}

func logWithTxnContext(app *newrelic.Application, logger *slog.Logger) {
	txn := app.StartTransaction("test")
	ctx := newrelic.NewContext(context.Background(), txn)
	logger.InfoContext(ctx, "log has tracing on new relic")
	txn.End()
}

func main() {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("slog example"),
		newrelic.ConfigFromEnvironment(),
		newrelic.ConfigAppLogEnabled(true),
	)
	if err != nil {
		panic(err)
	}

	app.WaitForConnection(time.Second * 5)
	logger := slog.New(nrslog.TextHandler(app, os.Stdout, &slog.HandlerOptions{}))

	logWithGroup(logger)
	logWithGroupAndAttributes(logger)
	logWithTxnContext(app, logger)
	logWithLogAttrs(app, logger)
	logger.Info("All Done!")

	app.Shutdown(time.Second * 10)
}
