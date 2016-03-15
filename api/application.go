package api

import "net/http"

type Application interface {
	// RecordCustomEvent adds a custom event to the application.  Each
	// application holds and reports up to 10*1000 custom events per minute.
	// Once this limit is reached, sampling will occur.  This feature is
	// incompatible with high security mode.
	//
	// eventType must consist of alphanumeric characters, underscores, and
	// colons, and must contain fewer than 255 bytes.
	//
	// Each value in the params map must be a number, string, or boolean.
	// Keys must be less than 255 bytes.  The params map may not contain
	// more than 64 attributes.  For more information, and a set of
	// restricted keywords, see:
	//
	// https://docs.newrelic.com/docs/insights/new-relic-insights/adding-querying-data/inserting-custom-events-new-relic-apm-agents
	RecordCustomEvent(eventType string, params map[string]interface{}) error

	// StartTransaction begins a Transaction.  The Transaction can always be
	// used safely, as nil will never be returned.
	StartTransaction(name string, w http.ResponseWriter, r *http.Request) Transaction
}
