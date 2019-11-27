package nrzap_test

import (
	"github.com/newrelic/go-agent/v3/integrations/nrzap"
	"github.com/newrelic/go-agent/v3/newrelic"
	"go.uber.org/zap"
)

func Example() {
	// Create a new zap logger:
	z, _ := zap.NewProduction()

	newrelic.NewApplication(
		newrelic.ConfigAppName("Example App"),
		newrelic.ConfigLicense("__YOUR_NEWRELIC_LICENSE_KEY__"),
		// Use nrzap to register the logger with the agent:
		nrzap.ConfigLogger(z.Named("newrelic")),
	)
}
