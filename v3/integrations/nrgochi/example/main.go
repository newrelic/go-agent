// main.go
package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/newrelic/go-agent/v3/integrations/nrgochi"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func makeChiEndpoint(s string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(s))
	}
}

func endpoint404(w http.ResponseWriter, r *http.Request) {
	newrelic.FromContext(r.Context()).NoticeError(fmt.Errorf("returning 404"))

	w.WriteHeader(404)
	w.Write([]byte("returning 404"))
}

func endpointChangeCode(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(404)
	w.WriteHeader(200)
	w.Write([]byte("actually ok!"))
}

func endpointResponseHeaders(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"zip":"zap"}`))
}

func endpointNotFound(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("there's no endpoint for that!"))
}

func endpointAccessTransaction(w http.ResponseWriter, r *http.Request) {
	txn := newrelic.FromContext(r.Context())
	txn.SetName("custom-name")
	w.Write([]byte("changed the name of the transaction!"))
}

func main() {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Chi App"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDebugLogger(os.Stdout),
		newrelic.ConfigCodeLevelMetricsEnabled(true),
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	router := chi.NewRouter()
	router.Use(nrgochi.Middleware(app))

	router.Get("/404", endpoint404)
	router.Get("/change", endpointChangeCode)
	router.Get("/headers", endpointResponseHeaders)
	router.Get("/txn", endpointAccessTransaction)

	// Since the handler function name is used as the transaction name,
	// anonymous functions do not get usefully named. We encourage
	// transforming anonymous functions into named functions.
	router.Get("/anon", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("anonymous function handler"))
	})

	router.NotFound(endpointNotFound)

	http.ListenAndServe(":8000", router)
}
