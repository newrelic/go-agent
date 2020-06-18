// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrgorilla

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/internal"
	"github.com/newrelic/go-agent/internal/integrationsupport"
)

func makeHandler(text string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(text))
	})
}

func TestBasicRoute(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	r := mux.NewRouter()
	r.Handle("/alpha", makeHandler("alpha response"))
	InstrumentRoutes(r, app)
	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/alpha", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "alpha response" {
		t.Error("wrong response body", respBody)
	}
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:  "alpha",
		IsWeb: true,
	})
}

func TestSubrouterRoute(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	r := mux.NewRouter()
	users := r.PathPrefix("/users").Subrouter()
	users.Handle("/add", makeHandler("adding user"))
	InstrumentRoutes(r, app)
	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/users/add", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "adding user" {
		t.Error("wrong response body", respBody)
	}
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:  "users/add",
		IsWeb: true,
	})
}

func TestNamedRoute(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	r := mux.NewRouter()
	r.Handle("/named", makeHandler("named route")).Name("special-name-route")
	InstrumentRoutes(r, app)
	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/named", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "named route" {
		t.Error("wrong response body", respBody)
	}
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:  "special-name-route",
		IsWeb: true,
	})
}

func TestRouteNotFound(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	r := mux.NewRouter()
	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte("not found"))
	})
	// Tests that routes do not get double instrumented when
	// InstrumentRoutes is called twice by expecting error metrics with a
	// count of 1.
	InstrumentRoutes(r, app)
	InstrumentRoutes(r, app)
	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/foo", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "not found" {
		t.Error("wrong response body", respBody)
	}
	if response.Code != 500 {
		t.Error("wrong response code", response.Code)
	}
	// Error metrics test the 500 response code capture.
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:      "NotFoundHandler",
		IsWeb:     true,
		NumErrors: 1,
	})
}

func TestNilApp(t *testing.T) {
	var app newrelic.Application
	r := mux.NewRouter()
	r.Handle("/alpha", makeHandler("alpha response"))
	InstrumentRoutes(r, app)
	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/alpha", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "alpha response" {
		t.Error("wrong response body", respBody)
	}
}
