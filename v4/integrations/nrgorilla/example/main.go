// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/newrelic/go-agent/v3/integrations/nrgorilla"
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
	r.Use(nrgorilla.Middleware(app))

	r.Handle("/", makeHandler("index"))
	r.Handle("/alpha", makeHandler("alpha"))

	users := r.PathPrefix("/users").Subrouter()
	users.Handle("/add", makeHandler("adding user"))
	users.Handle("/delete", makeHandler("deleting user"))

	// The route name will be used as the transaction name if one is set.
	r.Handle("/named", makeHandler("named route")).Name("special-name-route")

	// The NotFoundHandler and MethodNotAllowedHandler must be instrumented
	// separately.
	_, r.NotFoundHandler = newrelic.WrapHandle(app, "NotFoundHandler", makeHandler("not found"))
	_, r.MethodNotAllowedHandler = newrelic.WrapHandle(app, "MethodNotAllowedHandler", makeHandler("method not allowed"))

	http.ListenAndServe(":8000", r)
}
