// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrecho

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo"
	"github.com/newrelic/go-agent/internal"
	"github.com/newrelic/go-agent/internal/integrationsupport"
)

func TestBasicRoute(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()

	e := echo.New()
	e.Use(Middleware(app))
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
		Name:  "hello",
		IsWeb: true,
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             "WebTransaction/Go/hello",
			"nr.apdexPerfZone": "S",
		},
		AgentAttributes: map[string]interface{}{
			"httpResponseCode":             "200",
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
	e.Use(Middleware(app))
	e.GET("/hello", func(c echo.Context) error {
		txn := FromContext(c)
		if nil != txn {
			txn.NoticeError(errors.New("ooops"))
		}
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
		Name:      "hello",
		IsWeb:     true,
		NumErrors: 1,
	})
}

func TestNotFoundHandler(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()

	e := echo.New()
	e.Use(Middleware(app))

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/hello?remove=me", nil)
	if err != nil {
		t.Fatal(err)
	}

	e.ServeHTTP(response, req)
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:  "NotFoundHandler",
		IsWeb: true,
	})
}

func TestMethodNotAllowedHandler(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()

	e := echo.New()
	e.Use(Middleware(app))
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
		Name:      "MethodNotAllowedHandler",
		IsWeb:     true,
		NumErrors: 1,
	})
}

func TestReturnsHTTPError(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()

	e := echo.New()
	e.Use(Middleware(app))
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
		Name:      "hello",
		IsWeb:     true,
		NumErrors: 1,
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             "WebTransaction/Go/hello",
			"nr.apdexPerfZone": "F",
		},
		AgentAttributes: map[string]interface{}{
			"httpResponseCode": "418",
			"request.method":   "GET",
			"request.uri":      "/hello",
		},
		UserAttributes: map[string]interface{}{},
	}})
}

func TestReturnsError(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()

	e := echo.New()
	e.Use(Middleware(app))
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
		Name:      "hello",
		IsWeb:     true,
		NumErrors: 1,
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             "WebTransaction/Go/hello",
			"nr.apdexPerfZone": "F",
		},
		AgentAttributes: map[string]interface{}{
			"httpResponseCode": "500",
			"request.method":   "GET",
			"request.uri":      "/hello",
		},
		UserAttributes: map[string]interface{}{},
	}})
}

func TestResponseCode(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()

	e := echo.New()
	e.Use(Middleware(app))
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
		Name:      "hello",
		IsWeb:     true,
		NumErrors: 1,
	})
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             "WebTransaction/Go/hello",
			"nr.apdexPerfZone": "F",
		},
		AgentAttributes: map[string]interface{}{
			"httpResponseCode":             "418",
			"request.method":               "GET",
			"response.headers.contentType": "text/html",
			"request.uri":                  "/hello",
		},
		UserAttributes: map[string]interface{}{},
	}})
}
