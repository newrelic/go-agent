package nrlogxi_test

import (
	log "github.com/mgutz/logxi/v1"
	newrelic "github.com/newrelic/go-agent"
	nrlogxi "github.com/newrelic/go-agent/_integrations/nrlogxi/v1"
)

func Example() {
	// Create a new logxi logger:
	l := log.New("newrelic")
	l.SetLevel(log.LevelInfo)

	newrelic.NewApplication(
		newrelic.ConfigAppName("Example App"),
		newrelic.ConfigLicense("__YOUR_NEWRELIC_LICENSE_KEY__"),
		// Use nrlogxi to register the logger with the agent:
		nrlogxi.ConfigLogger(l),
	)
}
