// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrecho

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo"
	"github.com/newrelic/go-agent/v4/internal"
	"github.com/newrelic/go-agent/v4/internal/integrationsupport"
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
		Name:  "GET /hello",
		IsWeb: true,
	})
	app.ExpectSpanEvents(t, []internal.WantSpan{{
		Name:          "GET /hello",
		ParentID:      internal.MatchNoParent,
		SkipAttrsTest: true,
		Attributes: map[string]interface{}{
			"nr.apdexPerfZone":             "S",
			"httpResponseCode":             "200",
			"http.statusCode":              "200",
			"request.method":               "GET",
			"response.headers.contentType": "text/html",
			"request.uri":                  "/hello",
		},
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
		defer FromContext(c).StartSegment("segment").End()
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
		Name:  "GET /hello",
		IsWeb: true,
	})
	app.ExpectSpanEvents(t, []internal.WantSpan{
		{
			Name:       "segment",
			ParentID:   internal.MatchAnyParent,
			Attributes: map[string]interface{}{},
		},
		{
			Name:     "GET /hello",
			ParentID: internal.MatchNoParent,
			Attributes: map[string]interface{}{
				"http.flavor":      "1.1",
				"http.method":      "GET",
				"http.scheme":      "http",
				"http.status_code": int64(200),
				"http.status_text": "OK",
				"http.target":      "",
				"net.transport":    "IP.TCP",
			},
		},
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
		Name:      "NotFoundHandler",
		IsWeb:     true,
		NumErrors: 1,
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
		Name:      "MethodNotAllowedHandler",
		IsWeb:     true,
		NumErrors: 1,
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
		Name:      "GET /hello",
		IsWeb:     true,
		NumErrors: 1,
	})
	app.ExpectSpanEvents(t, []internal.WantSpan{{
		Name:          "GET /hello",
		ParentID:      internal.MatchNoParent,
		StatusCode:    3,
		SkipAttrsTest: true,
		Attributes: map[string]interface{}{
			"nr.apdexPerfZone": "F",
			"httpResponseCode": "418",
			"http.statusCode":  "418",
			"request.method":   "GET",
			"request.uri":      "/hello",
		},
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
		Name:      "GET /hello",
		IsWeb:     true,
		NumErrors: 1,
	})
	app.ExpectSpanEvents(t, []internal.WantSpan{{
		Name:          "GET /hello",
		ParentID:      internal.MatchNoParent,
		StatusCode:    13,
		SkipAttrsTest: true,
		Attributes: map[string]interface{}{
			"nr.apdexPerfZone": "F",
			"httpResponseCode": "500",
			"http.statusCode":  "500",
			"request.method":   "GET",
			"request.uri":      "/hello",
		},
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
		Name:      "GET /hello",
		IsWeb:     true,
		NumErrors: 1,
	})
	app.ExpectSpanEvents(t, []internal.WantSpan{{
		Name:          "GET /hello",
		ParentID:      internal.MatchNoParent,
		StatusCode:    3,
		SkipAttrsTest: true,
		Attributes: map[string]interface{}{
			"nr.apdexPerfZone":             "F",
			"httpResponseCode":             "418",
			"http.statusCode":              "418",
			"request.method":               "GET",
			"response.headers.contentType": "text/html",
			"request.uri":                  "/hello",
		},
	}})
}
