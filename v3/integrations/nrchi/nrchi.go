// Copyright 2022 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrchi instruments https://github.com/go-chi/chi applications.
//
// Example: https://github.com/newrelic/go-agent/tree/master/v3/integrations/nrchi/example/main.go
package nrchi

import (
	"net/http"

	chi "github.com/go-chi/chi/v5"
	"github.com/newrelic/go-agent/v3/internal"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)

func init() { internal.TrackUsage("integration", "framework", "go-chi") }

type NewRelicApp struct{
	app *newrelic.Application
}

// New creates a new go-chi router
func NewRouter(app *newrelic.Application) *chi.Mux {
	nrapp := NewRelicApp{
		app: app,
	}
	r := chi.NewRouter()
	r.Use(nrapp.addNewRelicContext)
	return r
}

func (nrapp NewRelicApp)addNewRelicContext(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		// start newrelic transaction
		txn := nrapp.app.StartTransaction(r.Method+r.URL.RequestURI())
		defer txn.End()

		txn.SetWebRequestHTTP(r)
		ctx := newrelic.NewContext(r.Context(), txn)

		next.ServeHTTP(w, r.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}
