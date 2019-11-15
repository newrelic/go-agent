package nrzap

import (
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
	"go.uber.org/zap"
)

func Example() {
	// Create a new zap logger:
	z, _ := zap.NewProduction()

	newrelic.NewApplication(
		newrelic.ConfigAppName("Example App"),
		newrelic.ConfigLicense("__YOUR_NEWRELIC_LICENSE_KEY__"),
		// Use nrzap to register the logger with the agent:
		ConfigLogger(z.Named("newrelic")),
	)
}
