// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrgorilla

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
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
	InstrumentRoutes(r, app.Application)
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
		Name:          "GET /alpha",
		IsWeb:         true,
		UnknownCaller: true,
	})
}

func TestSubrouterRoute(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	r := mux.NewRouter()
	users := r.PathPrefix("/users").Subrouter()
	users.Handle("/add", makeHandler("adding user"))
	InstrumentRoutes(r, app.Application)
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
		Name:          "GET /users/add",
		IsWeb:         true,
		UnknownCaller: true,
	})
}

func TestNamedRoute(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	r := mux.NewRouter()
	r.Handle("/named", makeHandler("named route")).Name("special-name-route")
	InstrumentRoutes(r, app.Application)
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
		Name:          "special-name-route",
		IsWeb:         true,
		UnknownCaller: true,
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
	InstrumentRoutes(r, app.Application)
	InstrumentRoutes(r, app.Application)
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
		Name:          "NotFoundHandler",
		IsWeb:         true,
		NumErrors:     1,
		UnknownCaller: true,
		ErrorByCaller: true,
	})
}

func TestNilApp(t *testing.T) {
	var app *newrelic.Application
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

func TestMiddlewareBasicRoute(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	r := mux.NewRouter()
	r.Handle("/alpha", makeHandler("alpha response"))
	r.Use(Middleware(app.Application))
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// ensure that the txn is added to the context and accessible by
			// middlewares
			newrelic.FromContext(r.Context()).NoticeError(errors.New("oops"))
			next.ServeHTTP(w, r)
		})
	})
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
		Name:          "GET /alpha",
		IsWeb:         true,
		NumErrors:     1,
		UnknownCaller: true,
		ErrorByCaller: true,
	})
}

func TestMiddlewareNilApp(t *testing.T) {
	r := mux.NewRouter()
	r.Handle("/alpha", makeHandler("alpha response"))
	r.Use(Middleware(nil))
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

func TestMiddlewareAndInstrumentRoutes(t *testing.T) {
	// Test that only one transaction is created when Middleware and
	// InstrumentRoutes are used.
	app := integrationsupport.NewBasicTestApp()
	r := mux.NewRouter()
	r.Handle("/alpha", makeHandler("alpha response"))
	r.Use(Middleware(app.Application))
	InstrumentRoutes(r, app.Application)
	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/alpha", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "alpha response" {
		t.Error("wrong response body", respBody)
	}
	app.ExpectTxnEvents(t, []internal.WantEvent{
		{},
	})
}

func TestMiddlewareNotFoundHandler(t *testing.T) {
	// This test will fail if gorilla ever decides to run the NotFoundHandler
	// through the middleware.
	app := integrationsupport.NewBasicTestApp()
	r := mux.NewRouter()
	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte("not found"))
	})
	r.Use(Middleware(app.Application))
	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/foo", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "not found" {
		t.Error("wrong response body", respBody)
	}
	if response.Code != 404 {
		t.Error("wrong response code", response.Code)
	}
	// make sure no txn events were created
	app.ExpectTxnEvents(t, []internal.WantEvent{})
}

func TestMiddlewareMethodNotAllowedHandler(t *testing.T) {
	// This test will fail if gorilla ever decides to run the
	// MethodNotAllowedHandler through the middleware.
	app := integrationsupport.NewBasicTestApp()
	r := mux.NewRouter()
	r.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(405)
		w.Write([]byte("method not allowed"))
	})
	r.Use(Middleware(app.Application))
	r.Handle("/foo", makeHandler("index")).Methods("POST")
	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/foo", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "method not allowed" {
		t.Error("wrong response body", respBody)
	}
	if response.Code != 405 {
		t.Error("wrong response code", response.Code)
	}
	// make sure no txn events were created
	app.ExpectTxnEvents(t, []internal.WantEvent{})
}
