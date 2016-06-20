package api

import "net/http"

type Application interface {
	// StartTransaction begins a Transaction.
	// * This method never returns nil.
	// * If an http.Request is provided, the Transaction is considered
	//   a web transaction.
	// * If an http.ResponseWriter is provided, the Transaction can be
	//   used as an http.ResponseWriter in place of the one given.  This
	//   allows for instrumentation of the HTTP response code and
	//   response headers.
	StartTransaction(name string, w http.ResponseWriter, r *http.Request) Transaction

	// RecordCustomEvent adds a custom event to the application.  This
	// feature is incompatible with high security mode.
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
}
