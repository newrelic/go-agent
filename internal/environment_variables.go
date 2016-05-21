package internal

import "os"

var (
	debugLogging = os.Getenv("NEW_RELIC_DEBUG_LOGGING")
	redirectHost = func() string {
		if s := os.Getenv("NEW_RELIC_HOST"); "" != s {
			return s
		}
		return "collector.newrelic.com"
	}()
)
