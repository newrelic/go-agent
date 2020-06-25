// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrgin instruments https://github.com/gin-gonic/gin applications.
//
// Use this package to instrument inbound requests handled by a gin.Engine.
// Call nrgin.Middleware to get a gin.HandlerFunc which can be added to your
// application as a middleware:
//
//	router := gin.Default()
//	// Add the nrgin middleware before other middlewares or routes:
//	router.Use(nrgin.Middleware(app))
//
// Example: https://github.com/newrelic/go-agent/tree/master/_integrations/nrgin/v1/example/main.go
package nrgin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/internal"
)

func init() { internal.TrackUsage("integration", "framework", "gin", "v1") }

// headerResponseWriter gives the transaction access to response headers and the
// response code.
type headerResponseWriter struct{ w gin.ResponseWriter }

func (w *headerResponseWriter) Header() http.Header       { return w.w.Header() }
func (w *headerResponseWriter) Write([]byte) (int, error) { return 0, nil }
func (w *headerResponseWriter) WriteHeader(int)           {}

var _ http.ResponseWriter = &headerResponseWriter{}

// replacementResponseWriter mimics the behavior of gin.ResponseWriter which
// buffers the response code rather than writing it when
// gin.ResponseWriter.WriteHeader is called.
type replacementResponseWriter struct {
	gin.ResponseWriter
	txn     newrelic.Transaction
	code    int
	written bool
}

var _ gin.ResponseWriter = &replacementResponseWriter{}

func (w *replacementResponseWriter) flushHeader() {
	if !w.written {
		w.txn.WriteHeader(w.code)
		w.written = true
	}
}

func (w *replacementResponseWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *replacementResponseWriter) Write(data []byte) (int, error) {
	w.flushHeader()
	return w.ResponseWriter.Write(data)
}

func (w *replacementResponseWriter) WriteString(s string) (int, error) {
	w.flushHeader()
	return w.ResponseWriter.WriteString(s)
}

func (w *replacementResponseWriter) WriteHeaderNow() {
	w.flushHeader()
	w.ResponseWriter.WriteHeaderNow()
}

// Context avoids making this package 1.7+ specific.
type Context interface {
	Value(key interface{}) interface{}
}

// Transaction returns the transaction stored inside the context, or nil if not
// found.
func Transaction(c Context) newrelic.Transaction {
	if v := c.Value(internal.GinTransactionContextKey); nil != v {
		if txn, ok := v.(newrelic.Transaction); ok {
			return txn
		}
	}
	if v := c.Value(internal.TransactionContextKey); nil != v {
		if txn, ok := v.(newrelic.Transaction); ok {
			return txn
		}
	}
	return nil
}

// Middleware creates a Gin middleware that instruments requests.
//
//	router := gin.Default()
//	// Add the nrgin middleware before other middlewares or routes:
//	router.Use(nrgin.Middleware(app))
//
func Middleware(app newrelic.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		if app != nil {
			name := c.HandlerName()
			w := &headerResponseWriter{w: c.Writer}
			txn := app.StartTransaction(name, w, c.Request)
			defer txn.End()

			repl := &replacementResponseWriter{
				ResponseWriter: c.Writer,
				txn:            txn,
				code:           http.StatusOK,
			}
			c.Writer = repl
			defer repl.flushHeader()

			c.Set(internal.GinTransactionContextKey, txn)
		}
		c.Next()
	}
}
