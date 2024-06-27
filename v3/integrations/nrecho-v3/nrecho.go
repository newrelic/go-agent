// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrecho instruments applications using
// https://github.com/labstack/echo v3.
//
// Use this package to instrument inbound requests handled by an echo.Echo
// instance.
//
//	e := echo.New()
//	// Add the nrecho middleware before other middlewares or routes:
//	e.Use(nrecho.Middleware(app))
//
// Example: https://github.com/newrelic/go-agent/tree/master/v3/integrations/nrecho-v3/example/main.go
package nrecho

import (
	"net/http"
	"reflect"

	"github.com/labstack/echo"
	"github.com/newrelic/go-agent/v3/internal"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)

func init() { internal.TrackUsage("integration", "framework", "echo") }

// FromContext returns the Transaction from the context if present, and nil
// otherwise.
func FromContext(c echo.Context) *newrelic.Transaction {
	return newrelic.FromContext(c.Request().Context())
}

func handlerPointer(handler echo.HandlerFunc) uintptr {
	return reflect.ValueOf(handler).Pointer()
}

func handlerName(router interface{}) string {
	val := reflect.ValueOf(router)
	if val.Kind() == reflect.Ptr { // for echo version v3.2.2+
		val = val.Elem()
	} else {
		val = reflect.ValueOf(&router).Elem().Elem()
	}
	if name := val.FieldByName("Name"); name.IsValid() { // for echo version v3.2.2+
		return name.String()
	} else if handler := val.FieldByName("Handler"); handler.IsValid() {
		return handler.String()
	} else {
		return ""
	}
}

func transactionName(c echo.Context) (string, string) {
	ptr := handlerPointer(c.Handler())
	if ptr == handlerPointer(echo.NotFoundHandler) {
		return "NotFoundHandler", ""
	}
	if ptr == handlerPointer(echo.MethodNotAllowedHandler) {
		return "MethodNotAllowedHandler", ""
	}
	return c.Request().Method + " " + c.Path(), c.Path()
}

// Middleware creates Echo middleware that instruments requests.
//
//	e := echo.New()
//	// Add the nrecho middleware before other middlewares or routes:
//	e.Use(nrecho.Middleware(app))
func Middleware(app *newrelic.Application) func(echo.HandlerFunc) echo.HandlerFunc {
	if nil == app {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return next
		}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			rw := c.Response().Writer
			tName, route := transactionName(c)
			txn := app.StartTransaction(tName)
			defer txn.End()
			txn.SetCsecAttributes(newrelic.AttributeCsecRoute, route)
			txn.SetWebRequestHTTP(c.Request())

			c.Response().Writer = txn.SetWebResponse(rw)

			// Add txn to c.Request().Context()
			c.SetRequest(c.Request().WithContext(newrelic.NewContext(c.Request().Context(), txn)))

			err = next(c)

			// Record the response code. The response headers are not captured
			// in this case because they are set after this middleware returns.
			// Designed to mimic the logic in echo.DefaultHTTPErrorHandler.
			if nil != err && !c.Response().Committed {

				c.Response().Writer = rw

				if httperr, ok := err.(*echo.HTTPError); ok {
					txn.SetWebResponse(nil).WriteHeader(httperr.Code)
				} else {
					txn.SetWebResponse(nil).WriteHeader(http.StatusInternalServerError)
				}
			}

			return
		}
	}
}

// WrapRouter extracts API endpoints from the echo instance passed to it
// which is used to detect application URL mapping(api-endpoints) for provable security.
// In this version of the integration, this wrapper is only necessary if you are using the New Relic security agent integration [https://github.com/newrelic/go-agent/tree/master/v3/integrations/nrsecurityagent],
// but it may be enhanced to provide additional functionality in future releases.
//
//	 e := echo.New()
//	 ....
//	 ....
//	 ....
//
//	nrecho.WrapRouter(e)
func WrapRouter(engine *echo.Echo) {
	if engine != nil && newrelic.IsSecurityAgentPresent() {
		router := engine.Routes()
		for _, r := range router {
			newrelic.GetSecurityAgentInterface().SendEvent("API_END_POINTS", r.Path, r.Method, handlerName(r))
		}
	}
}
