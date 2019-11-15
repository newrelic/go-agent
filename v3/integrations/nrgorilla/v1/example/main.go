package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	nrgorilla "github.com/newrelic/go-agent/v3/integrations/nrgorilla/v1"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)

func makeHandler(text string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(text))
	})
}

func main() {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Gorilla App"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDebugLogger(os.Stdout),
	)
	if nil != err {
		fmt.Println(err)
		os.Exit(1)
	}

	r := mux.NewRouter()
	r.Handle("/", makeHandler("index"))
	r.Handle("/alpha", makeHandler("alpha"))

	users := r.PathPrefix("/users").Subrouter()
	users.Handle("/add", makeHandler("adding user"))
	users.Handle("/delete", makeHandler("deleting user"))

	// The route name will be used as the transaction name if one is set.
	r.Handle("/named", makeHandler("named route")).Name("special-name-route")

	// The NotFoundHandler will be instrumented if it is set.
	r.NotFoundHandler = makeHandler("not found")

	http.ListenAndServe(":8000", nrgorilla.InstrumentRoutes(r, app))
}
