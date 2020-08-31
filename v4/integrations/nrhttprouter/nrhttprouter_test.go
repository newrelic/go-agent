// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrhttprouter

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/newrelic/go-agent/v4/internal"
	"github.com/newrelic/go-agent/v4/internal/integrationsupport"
	newrelic "github.com/newrelic/go-agent/v4/newrelic"
)

func TestMethodFunctions(t *testing.T) {

	methodFuncs := []struct {
		Method string
		Fn     func(*Router) func(string, httprouter.Handle)
	}{
		{Method: "DELETE", Fn: func(r *Router) func(string, httprouter.Handle) { return r.DELETE }},
		{Method: "GET", Fn: func(r *Router) func(string, httprouter.Handle) { return r.GET }},
		{Method: "HEAD", Fn: func(r *Router) func(string, httprouter.Handle) { return r.HEAD }},
		{Method: "OPTIONS", Fn: func(r *Router) func(string, httprouter.Handle) { return r.OPTIONS }},
		{Method: "PATCH", Fn: func(r *Router) func(string, httprouter.Handle) { return r.PATCH }},
		{Method: "POST", Fn: func(r *Router) func(string, httprouter.Handle) { return r.POST }},
		{Method: "PUT", Fn: func(r *Router) func(string, httprouter.Handle) { return r.PUT }},
	}

	for _, md := range methodFuncs {
		app := integrationsupport.NewBasicTestApp()
		router := New(app.Application)
		md.Fn(router)("/hello/:name", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
			// Test that the Transaction is used as the response writer.
			w.WriteHeader(500)
			w.Write([]byte(fmt.Sprintf("hi %s", ps.ByName("name"))))
		})
		response := httptest.NewRecorder()
		req, err := http.NewRequest(md.Method, "/hello/person", nil)
		if err != nil {
			t.Fatal(err)
		}
		router.ServeHTTP(response, req)
		if respBody := response.Body.String(); respBody != "hi person" {
			t.Error("wrong response body", respBody)
		}
		app.ExpectTxnMetrics(t, internal.WantTxn{
			Name:      md.Method + " /hello/:name",
			IsWeb:     true,
			NumErrors: 1,
		})
	}
}

func TestGetNoApplication(t *testing.T) {
	router := New(nil)

	router.GET("/hello/:name", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		w.Write([]byte(fmt.Sprintf("hi %s", ps.ByName("name"))))
	})
	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/hello/person", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "hi person" {
		t.Error("wrong response body", respBody)
	}
}

func TestHandle(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	router := New(app.Application)

	router.Handle("GET", "/hello/:name", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		// Test that the Transaction is used as the response writer.
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("hi %s", ps.ByName("name"))))
		txn := newrelic.FromContext(r.Context())
		txn.AddAttribute("color", "purple")
	})
	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/hello/person", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "hi person" {
		t.Error("wrong response body", respBody)
	}
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:      "GET /hello/:name",
		IsWeb:     true,
		NumErrors: 1,
	})
	app.ExpectSpanEvents(t, []internal.WantSpan{
		{
			Name:          "GET /hello/:name",
			ParentID:      internal.MatchNoParent,
			StatusCode:    13,
			SkipAttrsTest: true,
			Attributes: map[string]interface{}{
				"nr.apdexPerfZone": internal.MatchAnything,
				"color":            "purple",
				"httpResponseCode": 500,
				"http.statusCode":  500,
				"request.method":   "GET",
				"request.uri":      "/hello/person",
			},
		},
	})
}

func TestHandler(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	router := New(app.Application)

	router.Handler("GET", "/hello/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Test that the Transaction is used as the response writer.
		w.WriteHeader(500)
		w.Write([]byte("hi there"))
		txn := newrelic.FromContext(r.Context())
		txn.AddAttribute("color", "purple")
	}))
	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/hello/", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "hi there" {
		t.Error("wrong response body", respBody)
	}
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:      "GET /hello/",
		IsWeb:     true,
		NumErrors: 1,
	})
	app.ExpectSpanEvents(t, []internal.WantSpan{
		{
			Name:          "GET /hello/",
			ParentID:      internal.MatchNoParent,
			StatusCode:    13,
			SkipAttrsTest: true,
			Attributes: map[string]interface{}{
				"nr.apdexPerfZone": internal.MatchAnything,
				"color":            "purple",
				"httpResponseCode": 500,
				"http.statusCode":  500,
				"request.method":   "GET",
				"request.uri":      "/hello/",
			},
		},
	})
}

func TestHandlerMissingApplication(t *testing.T) {
	router := New(nil)

	router.Handler("GET", "/hello/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte("hi there"))
	}))
	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/hello/", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "hi there" {
		t.Error("wrong response body", respBody)
	}
}

func TestHandlerFunc(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	router := New(app.Application)

	router.HandlerFunc("GET", "/hello/", func(w http.ResponseWriter, r *http.Request) {
		// Test that the Transaction is used as the response writer.
		w.WriteHeader(500)
		w.Write([]byte("hi there"))
	})
	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/hello/", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "hi there" {
		t.Error("wrong response body", respBody)
	}
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:      "GET /hello/",
		IsWeb:     true,
		NumErrors: 1,
	})
}

func TestNotFound(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	router := New(app.Application)

	router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Test that the Transaction is used as the response writer.
		w.WriteHeader(500)
		w.Write([]byte("not found!"))
		txn := newrelic.FromContext(r.Context())
		txn.AddAttribute("color", "purple")
	})
	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/hello/", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "not found!" {
		t.Error("wrong response body", respBody)
	}
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:      "NotFound",
		IsWeb:     true,
		NumErrors: 1,
	})
	app.ExpectSpanEvents(t, []internal.WantSpan{
		{
			Name:          "NotFound",
			ParentID:      internal.MatchNoParent,
			StatusCode:    13,
			SkipAttrsTest: true,
			Attributes: map[string]interface{}{
				"nr.apdexPerfZone": internal.MatchAnything,
				"color":            "purple",
				"httpResponseCode": 500,
				"http.statusCode":  500,
				"request.method":   "GET",
				"request.uri":      "/hello/",
			},
		},
	})
}

func TestNotFoundMissingApplication(t *testing.T) {
	router := New(nil)

	router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Test that the Transaction is used as the response writer.
		w.WriteHeader(500)
		w.Write([]byte("not found!"))
	})
	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/hello/", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "not found!" {
		t.Error("wrong response body", respBody)
	}
}

func TestNotFoundNotSet(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	router := New(app.Application)

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/hello/", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if response.Code != 404 {
		t.Error(response.Code)
	}
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:      "NotFound",
		IsWeb:     true,
		NumErrors: 1,
	})
}
