// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrecho

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
)

func TestBasicRoute(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()

	e := echo.New()
	e.Use(Middleware(app.Application))
	e.GET("/hello", func(c echo.Context) error {
		return c.Blob(http.StatusOK, "text/html", []byte("Hello, World!"))
	})

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/hello?remove=me", nil)
	if err != nil {
		t.Fatal(err)
	}

	e.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "Hello, World!" {
		t.Error("wrong response body", respBody)
	}
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:          "GET /hello",
		IsWeb:         true,
		UnknownCaller: true,
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             "WebTransaction/Go/GET /hello",
			"nr.apdexPerfZone": "S",
			"sampled":          false,
			// Note: "*" is a wildcard value
			"guid":     "*",
			"traceId":  "*",
			"priority": "*",
		},
		AgentAttributes: map[string]interface{}{
			"httpResponseCode":             "200",
			"http.statusCode":              "200",
			"request.method":               "GET",
			"response.headers.contentType": "text/html",
			"request.uri":                  "/hello",
		},
		UserAttributes: map[string]interface{}{},
	}})
}

func TestNilApp(t *testing.T) {
	e := echo.New()
	e.Use(Middleware(nil))
	e.GET("/hello", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/hello?remove=me", nil)
	if err != nil {
		t.Fatal(err)
	}

	e.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "Hello, World!" {
		t.Error("wrong response body", respBody)
	}
}

func TestTransactionContext(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()

	e := echo.New()
	e.Use(Middleware(app.Application))
	e.GET("/hello", func(c echo.Context) error {
		txn := FromContext(c)
		txn.NoticeError(errors.New("ooops"))
		return c.String(http.StatusOK, "Hello, World!")
	})

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/hello?remove=me", nil)
	if err != nil {
		t.Fatal(err)
	}

	e.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "Hello, World!" {
		t.Error("wrong response body", respBody)
	}
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:          "GET /hello",
		IsWeb:         true,
		NumErrors:     1,
		UnknownCaller: true,
		ErrorByCaller: true,
	})
}

func TestNotFoundHandler(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()

	e := echo.New()
	e.Use(Middleware(app.Application))

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/hello?remove=me", nil)
	if err != nil {
		t.Fatal(err)
	}

	e.ServeHTTP(response, req)
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:          "NotFoundHandler",
		IsWeb:         true,
		UnknownCaller: true,
	})
}

func TestMethodNotAllowedHandler(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()

	e := echo.New()
	e.Use(Middleware(app.Application))
	e.GET("/hello", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})

	response := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/hello?remove=me", nil)
	if err != nil {
		t.Fatal(err)
	}

	e.ServeHTTP(response, req)
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:          "MethodNotAllowedHandler",
		IsWeb:         true,
		NumErrors:     1,
		UnknownCaller: true,
		ErrorByCaller: true,
	})
}

func TestReturnsHTTPError(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()

	e := echo.New()
	e.Use(Middleware(app.Application))
	e.GET("/hello", func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusTeapot, "I'm a teapot!")
	})

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/hello?remove=me", nil)
	if err != nil {
		t.Fatal(err)
	}

	e.ServeHTTP(response, req)
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:          "GET /hello",
		IsWeb:         true,
		NumErrors:     1,
		UnknownCaller: true,
		ErrorByCaller: true,
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             "WebTransaction/Go/GET /hello",
			"nr.apdexPerfZone": "F",
			"sampled":          false,
			"guid":             "*",
			"traceId":          "*",
			"priority":         "*",
		},
		AgentAttributes: map[string]interface{}{
			"httpResponseCode": "418",
			"http.statusCode":  "418",
			"request.method":   "GET",
			"request.uri":      "/hello",
		},
		UserAttributes: map[string]interface{}{},
	}})
}

func TestReturnsError(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()

	e := echo.New()
	e.Use(Middleware(app.Application))
	e.GET("/hello", func(c echo.Context) error {
		return errors.New("ooooooooops")
	})

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/hello?remove=me", nil)
	if err != nil {
		t.Fatal(err)
	}

	e.ServeHTTP(response, req)
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:          "GET /hello",
		IsWeb:         true,
		NumErrors:     1,
		UnknownCaller: true,
		ErrorByCaller: true,
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             "WebTransaction/Go/GET /hello",
			"nr.apdexPerfZone": "F",
			"sampled":          false,
			"guid":             "*",
			"traceId":          "*",
			"priority":         "*",
		},
		AgentAttributes: map[string]interface{}{
			"httpResponseCode": "500",
			"http.statusCode":  "500",
			"request.method":   "GET",
			"request.uri":      "/hello",
		},
		UserAttributes: map[string]interface{}{},
	}})
}

func TestResponseCode(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()

	e := echo.New()
	e.Use(Middleware(app.Application))
	e.GET("/hello", func(c echo.Context) error {
		return c.Blob(http.StatusTeapot, "text/html", []byte("Hello, World!"))
	})

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/hello?remove=me", nil)
	if err != nil {
		t.Fatal(err)
	}

	e.ServeHTTP(response, req)
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:          "GET /hello",
		IsWeb:         true,
		NumErrors:     1,
		UnknownCaller: true,
		ErrorByCaller: true,
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             "WebTransaction/Go/GET /hello",
			"nr.apdexPerfZone": "F",
			"sampled":          false,
			"guid":             "*",
			"traceId":          "*",
			"priority":         "*",
		},
		AgentAttributes: map[string]interface{}{
			"httpResponseCode":             "418",
			"http.statusCode":              "418",
			"request.method":               "GET",
			"response.headers.contentType": "text/html",
			"request.uri":                  "/hello",
		},
		UserAttributes: map[string]interface{}{},
	}})
}
