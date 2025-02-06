package nrslog

import (
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
)

const updateFrequency = 1 * time.Minute // check infrequently because the go agent config is not expected to change --> cost 50-100 uS

// 44% faster than checking the config on every log message
type configCache struct {
	lastCheck time.Time

	// true if we have successfully gotten the config at least once to verify the agent is connected
	gotStartupConfig bool
	// true if the logs in context feature is enabled as well as either local decorating or forwarding
	enabled     bool
	enrichLogs  bool
	forwardLogs bool
}

func (c *configCache) shouldEnrichLog(app *newrelic.Application) bool {
	c.update(app)
	return c.enrichLogs
}

func (c *configCache) shouldForwardLogs(app *newrelic.Application) bool {
	c.update(app)
	return c.forwardLogs
}

// isEnabled returns true if the logs in context feature is enabled
// as well as either local decorating or forwarding.
func (c *configCache) isEnabled(app *newrelic.Application) bool {
	c.update(app)
	return c.enabled
}

// Note: this has a data race in async use cases, but it does not
// cause logical errors, only cache misses. This is acceptable in
// comparison to the cost of synchronization.
func (c *configCache) update(app *newrelic.Application) {
	// do not get the config from agent if we have successfully gotten it before
	// and it has been less than updateFrequency since the last check. This is
	// because on startup, the agent will return a dummy config until it has
	// connected and received the real config.
	if c.gotStartupConfig && time.Since(c.lastCheck) < updateFrequency {
		return
	}

	config, ok := app.Config()
	if !ok {
		c.enrichLogs = false
		c.forwardLogs = false
		c.enabled = false
		return
	}

	c.gotStartupConfig = true
	c.enrichLogs = config.ApplicationLogging.LocalDecorating.Enabled && config.ApplicationLogging.Enabled
	c.forwardLogs = config.ApplicationLogging.Forwarding.Enabled && config.ApplicationLogging.Enabled
	c.enabled = config.ApplicationLogging.Enabled && (c.enrichLogs || c.forwardLogs)

	c.lastCheck = time.Now()
}
