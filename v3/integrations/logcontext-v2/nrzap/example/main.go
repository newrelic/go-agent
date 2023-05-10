package main

import (
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
	backgroundLogger.Info("this is a background log message")

	txn := app.StartTransaction("nrzap example transaction")
	txnCore, err := nrzap.WrapTransactionCore(core, txn)
	if err != nil && err != nrzap.ErrNilApp {
		panic(err)
	}

	txnLogger := zap.New(txnCore)
	txnLogger.Info("this is a transaction log message")

	txn.End()

	app.Shutdown(10 * time.Second)
}
