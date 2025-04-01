// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrgin instruments https://github.com/gofiber/fiber applications.
//
// Use this package to instrument inbound requests handled by a gin.Engine.
// Call nrfiber.Middleware to get a nrfiber.HandlerFunc which can be added to your
// application as a middleware:
//
//	router := nrfiber.New()
//	// Add the nrgin middleware before other middlewares or routes:
//	router.Use(nrfiber.Middleware(app))
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

func init() { internal.TrackUsage("integration", "framework", "fiber", "v1") }

// headerResponseWriter gives the transaction access to response headers and the
// response code.
type headerResponseWriter struct {
	w fiber.Response
}

func (w *headerResponseWriter) Header() http.Header       { return w.Header() }
func (w *headerResponseWriter) Write([]byte) (int, error) { return 0, nil }
func (w *headerResponseWriter) WriteHeader(int)           {}

var _ http.ResponseWriter = &headerResponseWriter{}

// Transaction returns the Transaction from the context if it exists.
func Transaction(ctx context.Context) *newrelic.Transaction {
	if ctx == nil {
		return nil
	}
	if txn, ok := ctx.Value(internal.TransactionContextKey).(*newrelic.Transaction); ok {
		return txn
	}
	if txn, ok := ctx.Value("transaction").(newrelic.Transaction); ok {
		return &txn
	}
	return nil
}

// getTransactionName returns a transaction name based on the request path
func getTransactionName(c *fiber.Ctx) string {
	path := c.Path()
	if path == "" {
		path = "/"
	}
	return string(c.Request().Header.Method()) + " " + path
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
func Middleware(app *newrelic.Application) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// If no New Relic application is configured, do nothing
		if app == nil {
			return c.Next()
		}
		// Create New Relic transaction
		txnName := getTransactionName(c)
		txn := app.StartTransaction(txnName, newrelic.WithFunctionLocation(c.App().Handler()))
		defer txn.End()
		w := &headerResponseWriter{w: *c.Response()}
		if newrelic.IsSecurityAgentPresent() {
			txn.SetCsecAttributes(newrelic.AttributeCsecRoute, c.Request().URI().String())
		}
		// Set web Response
		txn.SetWebResponse(w)
		// Set Web Requests
		txn.SetWebRequestHTTP(convertToHTTPRequest(c))
		// Execute next handlers
		err := c.Next()
		if err != nil {
			txn.NoticeError(err)
		}
		if newrelic.IsSecurityAgentPresent() {
			newrelic.GetSecurityAgentInterface().SendEvent("RESPONSE_HEADER", w.Header(), txn.GetLinkingMetadata().TraceID)
		}
		return err
	}
}
