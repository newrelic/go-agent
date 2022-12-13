// Copyright 2022 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrchi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)


func TestBasicRoute(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	router := NewRouter(app.Application)
	router.Get("/hello/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World!"))
	})

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/hello/", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "Hello World!" {
		t.Error("wrong response body", respBody)
	}
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:          "GET/hello/",
		IsWeb:         true,
		UnknownCaller: true,
	})
}

func TestRouteNotFound(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	router := NewRouter(app.Application)
	router.Get("/bar/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World!"))
	})

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/foo", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "404 page not found\n" {
		t.Error("wrong response body", respBody)
	}
	if response.Code != 404 {
		t.Error("wrong response code", response.Code)
	}
	// Error metrics test the 404 response code capture.
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:      "GET/foo",
		IsWeb:     true,
		UnknownCaller: true,
	})
}

func TestNilApp(t *testing.T) {
	var app newrelic.Application
	router := NewRouter(&app)
	router.Get("/hello/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World!"))
	})
	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/hello/", nil)
	if err != nil {
		t.Fatal(err)
	}

	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "Hello World!" {
		t.Error("wrong response body", respBody)
	}
}
