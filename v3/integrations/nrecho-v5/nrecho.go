// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrecho instruments applications using
// https://github.com/labstack/echo v5.
//
// Use this package to instrument inbound requests handled by an echo.Echo
// instance.
//
//	e := echo.New()
//	// Add the nrecho middleware before other middlewares or routes:
//	e.Use(nrecho.Middleware(app))
//
// Example: https://github.com/newrelic/go-agent/tree/master/v3/integrations/nrecho-v5/example/main.go
package nrecho

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/newrelic/go-agent/v3/internal"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)

func init() { internal.TrackUsage("integration", "framework", "echo") }

// FromContext returns the Transaction from the context if present, and nil
// otherwise.
func FromContext(c *echo.Context) *newrelic.Transaction {
	return newrelic.FromContext(c.Request().Context())
}

// echo v5 does not export a constant for the built-in 405 handler's route name
// the way it does echo.NotFoundRouteName, so match the literal value.
const methodNotAllowedRouteName = "echo_route_method_not_allowed_name"

func transactionName(c *echo.Context) (name string, path string) {
	ri := c.RouteInfo()

	if ri.Name == echo.NotFoundRouteName {
		return "NotFoundHandler", ""
	}
	if ri.Name == methodNotAllowedRouteName {
		return "MethodNotAllowedHandler", ""
	}

	return c.Request().Method + " " + ri.Path, ri.Path
}

// Skipper defines a function to skip middleware. Returning true skips processing
// the middleware.
type Skipper func(c *echo.Context) bool

// Config defines the config for the middleware.
type Config struct {
	// App contains newrelic application.
	App *newrelic.Application

	// Skipper defines a function to skip middleware.
	Skipper Skipper
}

type ConfigOption func(*Config)

func WithSkipper(skipper Skipper) ConfigOption {
	return func(cfg *Config) { cfg.Skipper = skipper }
}

// Middleware creates Echo middleware with provided config that
// instruments requests.
//
//	e := echo.New()
//	// Add the nrecho middleware before other middlewares or routes:
//	e.Use(nrecho.Middleware(app))
func Middleware(app *newrelic.Application, opts ...ConfigOption) echo.MiddlewareFunc {
	if app == nil {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return next
		}
	}

	config := Config{
		App: app,
	}

	for _, opt := range opts {
		opt(&config)
	}

	if config.Skipper == nil {
		// set default skipper
		config.Skipper = func(*echo.Context) bool {
			return false
		}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) (err error) {
			if config.Skipper(c) {
				return next(c)
			}

			// echo v5 Context.Response() returns http.ResponseWriter; unwrap to
			// the *echo.Response to reach its writer and Committed flag.
			echoResp, unwrapErr := echo.UnwrapResponse(c.Response())
			if unwrapErr != nil {
				return next(c)
			}

			rw := echoResp.ResponseWriter
			tname, path := transactionName(c)
			txn := config.App.StartTransaction(tname)
			defer txn.End()
			if newrelic.IsSecurityAgentPresent() {
				txn.SetCsecAttributes(newrelic.AttributeCsecRoute, path)
			}
			txn.SetWebRequestHTTP(c.Request())

			echoResp.ResponseWriter = txn.SetWebResponse(rw)

			// Add txn to c.Request().Context()
			c.SetRequest(c.Request().WithContext(newrelic.NewContext(c.Request().Context(), txn)))

			err = next(c)

			// Record the response code. The response headers are not captured
			// in this case because they are set after this middleware returns.
			// Designed to mimic the logic in echo's default HTTP error handler.
			if nil != err && !echoResp.Committed {

				echoResp.ResponseWriter = rw

				// echo v5 errors expose their status via HTTPStatusCoder rather
				// than a *echo.HTTPError type assertion.
				code := http.StatusInternalServerError
				if sc, ok := err.(interface{ StatusCode() int }); ok {
					code = sc.StatusCode()
				}
				txn.SetWebResponse(nil).WriteHeader(code)

				if newrelic.IsSecurityAgentPresent() {
					newrelic.GetSecurityAgentInterface().SendEvent("RESPONSE_HEADER", c.Response().Header(), txn.GetLinkingMetadata().TraceID)
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
		// echo v5 removed Echo.Routes(); reach routes via Router().Routes().
		for _, r := range engine.Router().Routes() {
			newrelic.GetSecurityAgentInterface().SendEvent("API_END_POINTS", r.Path, r.Method, r.Name)
		}
	}
}
