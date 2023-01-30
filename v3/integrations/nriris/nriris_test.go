// Copyright 2022 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nriris

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	"github.com/newrelic/go-agent/v3/newrelic"
)

var (
	pkg = "github.com/newrelic/go-agent/v3/integrations/nriris"
)

func hello(ctx iris.Context) {
	ctx.WriteString("hello response")
}

func TestBasicRoute(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	router := iris.New()
	router.Use(Middleware(app.Application))
	router.Get("/hello", hello)

	if err := router.Build(); err != nil {
		t.Fatal(err)
	}

	txnName := "GET " + pkg + ".hello"

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/hello", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "hello response" {
		t.Error("wrong response body", respBody)
	}
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:          txnName,
		IsWeb:         true,
		UnknownCaller: true,
	})
}

func TestRouterParty(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	router := iris.New()
	router.Use(Middleware(app.Application))
	group := router.Party("/group")
	group.Get("/hello", hello)

	if err := router.Build(); err != nil {
		t.Fatal(err)
	}

	txnName := "GET " + pkg + ".hello"

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/group/hello", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "hello response" {
		t.Error("wrong response body", respBody)
	}
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:          txnName,
		IsWeb:         true,
		UnknownCaller: true,
	})
}

func TestAnonymousHandler(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	router := iris.New()
	router.Use(Middleware(app.Application))
	router.Get("/anon", func(ctx iris.Context) {
		ctx.WriteString("anonymous function handler")
	})

	if err := router.Build(); err != nil {
		t.Fatal(err)
	}

	txnName := "GET " + pkg + ".TestAnonymousHandler"

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/anon", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "anonymous function handler" {
		t.Error("wrong response body", respBody)
	}
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:          txnName,
		IsWeb:         true,
		UnknownCaller: true,
	})
}

func multipleStatusCode(ctx iris.Context) {
	ctx.StatusCode(200)
	ctx.StatusCode(500)
	ctx.WriteString("multipleStatusCode")
}

func TestMultipleStatusCode(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	router := iris.New()
	router.Use(Middleware(app.Application))
	router.Get("/header", multipleStatusCode)

	if err := router.Build(); err != nil {
		t.Fatal(err)
	}

	txnName := "GET " + pkg + ".multipleStatusCode"

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/header", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "multipleStatusCode" {
		t.Error("wrong response body", respBody)
	}
	if response.Code != 500 {
		t.Error("wrong response code", response.Code)
	}
	// Error metrics test the 500 response code capture.
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:          txnName,
		IsWeb:         true,
		NumErrors:     1,
		UnknownCaller: true,
		ErrorByCaller: true,
	})
}

func accessTransactionIrisContext(ctx iris.Context) {
	txn := Transaction(ctx)
	txn.NoticeError(errors.New("problem"))
	ctx.WriteString("accessTransactionIrisContext")
}

func TestContextTransaction(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	router := iris.New()
	router.Use(Middleware(app.Application))
	router.Get("/txn", accessTransactionIrisContext)

	if err := router.Build(); err != nil {
		t.Fatal(err)
	}

	txnName := "GET " + pkg + ".accessTransactionIrisContext"

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/txn", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "accessTransactionIrisContext" {
		t.Error("wrong response body", respBody)
	}
	if response.Code != 200 {
		t.Error("wrong response code", response.Code)
	}
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:          txnName,
		IsWeb:         true,
		NumErrors:     1,
		UnknownCaller: true,
		ErrorByCaller: true,
	})
}

func TestNilApp(t *testing.T) {
	var app *newrelic.Application
	router := iris.New()
	router.Use(Middleware(app))
	router.Get("/hello", hello)

	if err := router.Build(); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/hello", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "hello response" {
		t.Error("wrong response body", respBody)
	}
}

func errorStatus(ctx iris.Context) {
	ctx.StatusCode(500)
	// ctx.ContentType("text/plain; charset=utf-8")
	// ctx.WriteString("an error happened")
	ctx.Text("an error happened")
}

func TestStatusCodes(t *testing.T) {
	// Test that we are correctly able to collect status code.
	expectCode := 500
	app := integrationsupport.NewBasicTestApp()
	router := iris.New()
	router.Use(Middleware(app.Application))
	router.Get("/err", errorStatus)

	if err := router.Build(); err != nil {
		t.Fatal(err)
	}

	txnName := "WebTransaction/Go/GET " + pkg + ".errorStatus"

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/err", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "an error happened" {
		t.Error("wrong response body", respBody)
	}
	if response.Code != 500 {
		t.Error("wrong response code", response.Code)
	}
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             txnName,
			"nr.apdexPerfZone": internal.MatchAnything,
			"sampled":          false,
			// Note: "*" is a wildcard value
			"guid":     "*",
			"traceId":  "*",
			"priority": "*",
		},
		UserAttributes: map[string]interface{}{},
		AgentAttributes: map[string]interface{}{
			"httpResponseCode":             expectCode,
			"http.statusCode":              expectCode,
			"request.method":               "GET",
			"request.uri":                  "/err",
			"response.headers.contentType": "text/plain; charset=utf-8",
		},
	}})
}

func noBody(ctx iris.Context) {
	ctx.StatusCode(500)
}

func TestNoResponseBody(t *testing.T) {
	// Test that when no response body is sent (i.e. ctx.Write is never
	// called) that we still capture status code.
	expectCode := 500
	app := integrationsupport.NewBasicTestApp()
	router := iris.New().Configure(iris.WithoutAutoFireStatusCode) /* do not write internal server error automatically */
	router.Use(Middleware(app.Application))
	router.Get("/nobody", noBody)

	if err := router.Build(); err != nil {
		t.Fatal(err)
	}

	txnName := "WebTransaction/Go/GET " + pkg + ".noBody"

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/nobody", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "" {
		t.Error("wrong response body", respBody)
	}
	if response.Code != 500 {
		t.Error("wrong response code", response.Code)
	}
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             txnName,
			"nr.apdexPerfZone": internal.MatchAnything,
			"sampled":          false,
			// Note: "*" is a wildcard value
			"guid":     "*",
			"traceId":  "*",
			"priority": "*",
		},

		UserAttributes: map[string]interface{}{},
		AgentAttributes: map[string]interface{}{
			"httpResponseCode": expectCode,
			"http.statusCode":  expectCode,
			"request.method":   "GET",
			"request.uri":      "/nobody",
		},
	}})
}

func TestRouteWithParams(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	router := iris.New()
	router.Use(Middleware(app.Application))
	router.Get("/hello/:name/*action", hello)

	if err := router.Build(); err != nil {
		t.Fatal(err)
	}

	txnName := "GET " + pkg + ".hello"

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/hello/world/fun", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "hello response" {
		t.Error("wrong response body", respBody)
	}
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:          txnName,
		IsWeb:         true,
		UnknownCaller: true,
	})
}
