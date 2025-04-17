// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrfiber instruments https://github.com/gofiber/fiber applications.
//
// Use this package to instrument inbound requests handled by a fiber.App.
// Call nrfiber.Middleware to get a fiber.Handler which can be added to your
// application as a middleware:
//
//	app := fiber.New()
//	// Add the nrfiber middleware before other middlewares or routes:
//	app.Use(nrfiber.Middleware(newrelicApp))
//
// Example: https://github.com/newrelic/go-agent/tree/master/v3/integrations/nrfiber/example/main.go
package nrfiber

import (
	"context"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

func init() {
	internal.TrackUsage("integration", "framework", "fiber", "v1")
}

// fastHeaderResponseWriter is a lightweight wrapper around Fiber's response
// that implements http.ResponseWriter interface
type fastHeaderResponseWriter struct {
	fiberResponse *fiber.Response
	header        http.Header // cached header to avoid repeated conversions
	statusCode    int
}

func newFastHeaderResponseWriter(resp *fiber.Response) *fastHeaderResponseWriter {
	return &fastHeaderResponseWriter{
		fiberResponse: resp,
		header:        make(http.Header),
		statusCode:    resp.StatusCode(),
	}
}

func (w *fastHeaderResponseWriter) Header() http.Header {
	// Return cached headers to avoid repeated conversions
	return w.header
}

func (w *fastHeaderResponseWriter) Write([]byte) (int, error) {
	// This is a no-op as we don't actually write anything here
	return 0, nil
}

func (w *fastHeaderResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.fiberResponse.SetStatusCode(statusCode)
}

// Apply cached headers to the actual Fiber response
func (w *fastHeaderResponseWriter) applyHeaders() {
	for key, values := range w.header {
		for _, value := range values {
			w.fiberResponse.Header.Set(key, value)
		}
	}
}

// Transaction returns the Transaction from the context if it exists.
func Transaction(ctx context.Context) *newrelic.Transaction {
	if ctx == nil {
		return nil
	}
	if txn, ok := ctx.Value(internal.TransactionContextKey).(*newrelic.Transaction); ok {
		return txn
	}
	return nil
}

// FromContext extracts a New Relic transaction from a Fiber context.
func FromContext(c *fiber.Ctx) *newrelic.Transaction {
	return newrelic.FromContext(c.UserContext())
}

// getTransactionName returns a transaction name based on the request path.
func getTransactionName(c *fiber.Ctx) string {
	path := c.Path()
	if path == "" {
		path = "/"
	}

	return string(c.Method()) + " " + path
}

// fastHTTPToRequest efficiently converts FastHTTP request to http.Request
// using fasthttpadaptor which is optimized for this purpose
func fastHTTPToRequest(ctx *fasthttp.RequestCtx) *http.Request {
	req := &http.Request{}
	fasthttpadaptor.ConvertRequest(ctx, req, true)
	return req
}

// Middleware creates a Fiber middleware handler that instruments requests with New Relic.
// It starts a New Relic transaction for each request, sets web request and response details,
// and handles error tracking. If no New Relic application is configured, it passes the request through.
//
//	router := fiber.New()
//	// Add the nrfiber middleware before other middlewares or routes:
//	router.Use(nrfiber.Middleware(app))
func Middleware(app *newrelic.Application) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// If no New Relic application is configured, do nothing
		if app == nil {
			return c.Next()
		}

		// Create New Relic transaction
		txnName := getTransactionName(c)
		txn := app.StartTransaction(txnName)
		defer txn.End()

		// Store transaction in context for retrieval in handlers
		ctx := context.WithValue(c.UserContext(), internal.TransactionContextKey, txn)
		c.SetUserContext(ctx)

		// Create optimized response writer wrapper
		w := newFastHeaderResponseWriter(c.Response())

		// Set security agent attributes if present
		if newrelic.IsSecurityAgentPresent() {
			txn.SetCsecAttributes(newrelic.AttributeCsecRoute, string(c.Request().URI().Path()))
		}

		// Set web response object
		txn.SetWebResponse(w)

		// Use fasthttpadaptor to efficiently convert to http.Request
		httpReq := fastHTTPToRequest(c.Context())
		txn.SetWebRequestHTTP(httpReq)

		// Execute next handlers
		err := c.Next()

		// Apply any headers that were set through the ResponseWriter interface
		w.applyHeaders()

		// Report error if any occurred
		if err != nil {
			txn.NoticeError(err)
		}

		// Update response status code in transaction
		txn.SetWebResponse(w).WriteHeader(c.Response().StatusCode())

		// Send security event if agent is present
		if newrelic.IsSecurityAgentPresent() {
			// Convert fiber response headers to http.Header for security event
			headers := make(http.Header)
			c.Response().Header.VisitAll(func(key, value []byte) {
				headers.Add(string(key), string(value))
			})

			newrelic.GetSecurityAgentInterface().SendEvent(
				"RESPONSE_HEADER",
				headers,
				txn.GetLinkingMetadata().TraceID,
			)
		}

		return err
	}
}

// WrapHandler wraps an existing Fiber handler with New Relic instrumentation
//
// fiberApp := fiber.New()
//
//	wrappedHandler := WrapHandler(app.Application, "/wrapped", func(c *fiber.Ctx) error {
//		 return c.SendString("Wrapped Handler")
//	})
//
// fiberApp.Get("/wrapped", wrappedHandler)
func WrapHandler(app *newrelic.Application, pattern string, handler fiber.Handler) fiber.Handler {
	if app == nil {
		return handler
	}

	return func(c *fiber.Ctx) error {
		// Get transaction from context if middleware is already applied
		if txn := Transaction(c.UserContext()); txn != nil {
			// Update name if this is a more specific handler
			txn.SetName(string(c.Method()) + " " + pattern)
			return handler(c)
		}

		// If no transaction exists, create a new one
		txn := app.StartTransaction(string(c.Method()) + " " + pattern)
		defer txn.End()

		// Store in context
		ctx := context.WithValue(c.UserContext(), internal.TransactionContextKey, txn)
		c.SetUserContext(ctx)

		// Create optimized response writer wrapper
		w := newFastHeaderResponseWriter(c.Response())

		// Set web response object
		txn.SetWebResponse(w)

		// Use fasthttpadaptor to efficiently convert to http.Request
		httpReq := fastHTTPToRequest(c.Context())
		txn.SetWebRequestHTTP(httpReq)

		// Call the handler
		err := handler(c)

		// Apply any headers that were set through the ResponseWriter interface
		w.applyHeaders()

		// Update response status code in transaction
		txn.SetWebResponse(w).WriteHeader(c.Response().StatusCode())

		// Report error if any occurred
		if err != nil {
			txn.NoticeError(err)
		}

		return err
	}
}
