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
	"net/url"

	"github.com/gofiber/fiber/v2"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func init() {
	internal.TrackUsage("integration", "framework", "fiber", "v2")
}

// headerResponseWriter gives the transaction access to response headers and the
// response code.
type headerResponseWriter struct {
	fiberResponse *fiber.Response
}

func (w *headerResponseWriter) Header() http.Header {
	header := make(http.Header)
	w.fiberResponse.Header.VisitAll(func(key, value []byte) {
		header.Add(string(key), string(value))
	})
	return header
}

func (w *headerResponseWriter) Write([]byte) (int, error) { return 0, nil }

func (w *headerResponseWriter) WriteHeader(statusCode int) {
	w.fiberResponse.SetStatusCode(statusCode)
}

var _ http.ResponseWriter = &headerResponseWriter{}

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

func FromContext(c *fiber.Ctx) *newrelic.Transaction {
	return newrelic.FromContext(c.UserContext())
}

// getTransactionName returns a transaction name based on the request path
func getTransactionName(c *fiber.Ctx) string {
	path := c.Path()
	if path == "" {
		path = "/"
	}

	return string(c.Method()) + " " + path
}

// convertHeaderToHTTP converts Fiber headers to http.Header
func convertHeaderToHTTP(c *fiber.Ctx) http.Header {
	header := make(http.Header)
	c.Request().Header.VisitAll(func(key, value []byte) {
		header.Add(string(key), string(value))
	})
	return header
}

// convertToHTTPRequest converts a Fiber context to http.Request
func convertToHTTPRequest(c *fiber.Ctx) *http.Request {
	// Create a simplified http.Request with essential information
	r := &http.Request{
		Method: string(c.Method()),
		URL: &url.URL{
			Path:     string(c.Path()),
			RawQuery: string(c.Query("")),
		},
		Header: convertHeaderToHTTP(c),
		Host:   string(c.Hostname()),
	}
	return r
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

		// Create response writer wrapper
		w := &headerResponseWriter{fiberResponse: c.Response()}

		// Set security agent attributes if present
		if newrelic.IsSecurityAgentPresent() {
			txn.SetCsecAttributes(newrelic.AttributeCsecRoute, string(c.Request().URI().Path()))
		}

		// Set web response object
		txn.SetWebResponse(w)

		// Set web request details
		txn.SetWebRequestHTTP(convertToHTTPRequest(c))

		// Execute next handlers
		err := c.Next()

		// Report error if any occurred
		if err != nil {
			txn.NoticeError(err)
		}

		// Update response status code in transaction
		txn.SetWebResponse(w).WriteHeader(c.Response().StatusCode())

		// Send security event if agent is present
		if newrelic.IsSecurityAgentPresent() {
			newrelic.GetSecurityAgentInterface().SendEvent(
				"RESPONSE_HEADER",
				w.Header(),
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
// wrappedHandler := WrapHandler(app.Application, "/wrapped", func(c *fiber.Ctx) error {
//	 return c.SendString("Wrapped Handler")
// })
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

		// Create response writer wrapper
		w := &headerResponseWriter{fiberResponse: c.Response()}

		// Set web response object
		txn.SetWebResponse(w)

		// Set web request details
		txn.SetWebRequestHTTP(convertToHTTPRequest(c))

		// Call the handler
		err := handler(c)

		// Update response status code in transaction
		txn.SetWebResponse(w).WriteHeader(c.Response().StatusCode())

		// Report error if any occurred
		if err != nil {
			txn.NoticeError(err)
		}

		return err
	}
}
