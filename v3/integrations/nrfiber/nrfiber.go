// Package nrfiber implements a middleware for the Fiber web framework.
// This middleware instruments Fiber applications with New Relic.
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

type contextKeyType struct{}

var (
	// TransactionContextKey is the Fiber context key used to store the
	// transaction.  This is exported to allow instrumentation of Fiber
	// applications using a fiber.Handler.
	TransactionContextKey = contextKeyType{}
)

// FromContext returns the Transaction from the context if it exists.
func FromContext(ctx context.Context) *newrelic.Transaction {
	if ctx == nil {
		return nil
	}
	if txn, ok := ctx.Value(TransactionContextKey).(*newrelic.Transaction); ok {
		return txn
	}
	return nil
}

// Middleware creates a Fiber middleware that instruments requests with
// New Relic.
func Middleware(app *newrelic.Application) fiber.Handler {
	return func(c *fiber.Ctx) error {

		// If no New Relic application is configured, do nothing
		if app == nil {
			return c.Next()
		}

		// Add request information to transaction
		webReq := convertToHTTPRequest(c)

		w := &headerResponseWriter{w: *c.Response()}

		// Create New Relic transaction
		txnName := getTransactionName(c)
		txn := app.StartTransaction(txnName)
		if newrelic.IsSecurityAgentPresent() {
			txn.SetCsecAttributes(newrelic.AttributeCsecRoute, c.Request().URI().String())
		}
		defer txn.End()

		// Set web Response
		txn.SetWebResponse(w)

		// Store the transaction in context
		userCtx := context.WithValue(c.UserContext(), TransactionContextKey, txn)
		c.SetUserContext(userCtx)

		// Accept distributed trace payload if present
		txn.AcceptDistributedTraceHeaders(newrelic.TransportHTTP, webReq.Header)
		txn.SetWebRequestHTTP(webReq)

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

// WrapHandler wraps an existing Fiber handler with New Relic instrumentation
func WrapHandler(app *newrelic.Application, pattern string, handler fiber.Handler) fiber.Handler {
	if app == nil {
		return handler
	}

	return func(c *fiber.Ctx) error {
		// Get transaction from context if middleware is already applied
		if txn := FromContext(c.UserContext()); txn != nil {
			txn.SetName("Web" + pattern + "/" + string(c.Method()))
			return handler(c)
		}

		// If no transaction exists, create a new one
		txn := app.StartTransaction("Web" + pattern)
		defer txn.End()

		userCtx := context.WithValue(c.UserContext(), TransactionContextKey, txn)
		c.SetUserContext(userCtx)

		txn.SetWebRequest(newrelic.WebRequest{
			Header: convertHeaderToHTTP(c),
			URL: &url.URL{
				Path:     string(c.Path()),
				RawQuery: string(c.Query("")),
			},
			Method:    string(c.Method()),
			Transport: newrelic.TransportHTTP,
		})

		err := handler(c)

		w := &headerResponseWriter{w: *c.Response()}
		txn.SetWebResponse(w)

		if err != nil {
			txn.NoticeError(err)
		}

		return err
	}
}
