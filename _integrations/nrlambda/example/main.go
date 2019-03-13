package main

import (
	"context"
	"fmt"

	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/_integrations/nrlambda"
)

func handler(ctx context.Context) {
	// The nrlambda handler instrumentation will add the transaction to  the
	// context.  Access it using newrelic.FromContext to add additional
	// instrumentation.
	if txn := newrelic.FromContext(ctx); nil != txn {
		txn.AddAttribute("userLevel", "gold")
	}
	fmt.Println("hello world")
}

func main() {
	// nrlambda.NewConfig should be used in place of newrelic.NewConfig
	// since it sets Lambda specific configuration settings including
	// Config.ServerlessMode.Enabled.
	cfg := nrlambda.NewConfig()
	// Here is the opportunity to change configuration settings before the
	// application is created.
	app, err := newrelic.NewApplication(cfg)
	if nil != err {
		fmt.Println("error creating app (invalid config):", err)
	}
	// nrlambda.Start should be used in place of lambda.Start.
	// nrlambda.StartHandler should be used in place of lambda.StartHandler.
	nrlambda.Start(handler, app)
}
