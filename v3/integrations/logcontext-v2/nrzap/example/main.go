package main

import (
	"errors"
	"os"
	"time"

	"github.com/newrelic/go-agent/v3/integrations/logcontext-v2/nrzap"
	"github.com/newrelic/go-agent/v3/newrelic"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("nrzerolog example"),
		newrelic.ConfigInfoLogger(os.Stdout),
		newrelic.ConfigDebugLogger(os.Stdout),
		newrelic.ConfigFromEnvironment(),
	)
	if err != nil {
		panic(err)
	}

	app.WaitForConnection(5 * time.Second)

	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(os.Stdout), zap.InfoLevel)
	backgroundCore, err := nrzap.WrapBackgroundCore(core, app)
	if err != nil && err != nrzap.ErrNilApp {
		panic(err)
	}

	backgroundLogger := zap.New(backgroundCore)
	backgroundLogger.Info("this is a background log message with fields test", zap.Any("foo", 3.14))

	txn := app.StartTransaction("nrzap example transaction")
	txnCore, err := nrzap.WrapTransactionCore(core, txn)
	if err != nil && err != nrzap.ErrNilTxn {
		panic(err)
	}
	txnLogger := zap.New(txnCore)
	txnLogger.Info("this is a transaction log message",
		zap.String("region", "region-test-2"),
		zap.Int("int", 123),
		zap.Duration("duration", 200*time.Millisecond),
		zap.Any("zapmap", map[string]any{"pi": 3.14, "duration": 2 * time.Second}),
	)

	err = errors.New("OW! an error occurred")
	txnLogger.Error("this is an error log message", zap.Error(err))

	txn.End()

	app.Shutdown(10 * time.Second)
}
