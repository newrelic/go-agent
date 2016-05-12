package api

import "net/http"

type Application interface {
	// StartTransaction begins a Transaction.
	// * This method will never return nil: the Transaction can always be
	//   used safely.
	// * If an http.Request is provided, the Transaction will be considered
	//   a web transaction, otherwise the Transaction will be considered a
	//   background transaction.
	// * If an http.ResponseWriter is provided, then the Transaction can be
	//   used as an http.ResponseWriter in place of the one given.  This
	//   will allow for instrumentation of the HTTP response code, as well
	//   as response headers.  See WrapHandle for an example of this
	//   pattern.
	// * If the correct name for the transaction is not known when this
	//   method is called, it can be updated using Transaction.SetName.
	StartTransaction(name string, w http.ResponseWriter, r *http.Request) Transaction

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
}
