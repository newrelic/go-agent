// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrhttprouter instruments https://github.com/julienschmidt/httprouter
// applications.
//
// Use this package to instrument inbound requests handled by a
// httprouter.Router. Use an *nrhttprouter.Router in place of your
// *httprouter.Router.  Example:
//
//	package main
//
//	import (
//		"fmt"
//		"net/http"
//		"os"
//
//		"github.com/julienschmidt/httprouter"
//		newrelic "github.com/newrelic/go-agent/v3/newrelic"
//		"github.com/newrelic/go-agent/v3/integrations/nrhttprouter"
//	)
//
//	func main() {
//		cfg := newrelic.NewConfig("httprouter App", os.Getenv("NEW_RELIC_LICENSE_KEY"))
//		app, _ := newrelic.NewApplication(cfg)
//
//		// Create the Router replacement:
//		router := nrhttprouter.New(app)
//
//		router.GET("/", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
//			w.Write([]byte("welcome\n"))
//		})
//		router.GET("/hello/:name", (w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
//			w.Write([]byte(fmt.Sprintf("hello %s\n", ps.ByName("name"))))
//		})
//		http.ListenAndServe(":8000", router)
//	}
//
// Runnable example: https://github.com/newrelic/go-agent/tree/master/v3/integrations/nrhttprouter/example/main.go
package nrhttprouter

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/newrelic/go-agent/v3/internal"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)

func init() { internal.TrackUsage("integration", "framework", "httprouter") }

// Router should be used in place of httprouter.Router.  Create it using
// New().
type Router struct {
	*httprouter.Router

	application *newrelic.Application
}

// New creates a new Router to be used in place of httprouter.Router.
func New(app *newrelic.Application) *Router {
	return &Router{
		Router:      httprouter.New(),
		application: app,
	}
}

func txnName(method, path string) string {
	return method + " " + path
}

func (r *Router) handle(method string, path string, original httprouter.Handle) {
	handle := original
	if nil != r.application {
		handle = func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
			txn := r.application.StartTransaction(txnName(method, path))
			txn.SetWebRequestHTTP(req)
			w = txn.SetWebResponse(w)
			defer txn.End()

			req = newrelic.RequestWithTransactionContext(req, txn)

			original(w, req, ps)
		}
	}
	r.Router.Handle(method, path, handle)
	if newrelic.IsSecurityAgentPresent() {
		newrelic.GetSecurityAgentInterface().SendEvent("API_END_POINTS", path, method, internal.HandlerName(original))
	}
}

// DELETE replaces httprouter.Router.DELETE.
func (r *Router) DELETE(path string, h httprouter.Handle) {
	r.handle(http.MethodDelete, path, h)
}

// GET replaces httprouter.Router.GET.
func (r *Router) GET(path string, h httprouter.Handle) {
	r.handle(http.MethodGet, path, h)
}

// HEAD replaces httprouter.Router.HEAD.
func (r *Router) HEAD(path string, h httprouter.Handle) {
	r.handle(http.MethodHead, path, h)
}

// OPTIONS replaces httprouter.Router.OPTIONS.
func (r *Router) OPTIONS(path string, h httprouter.Handle) {
	r.handle(http.MethodOptions, path, h)
}

// PATCH replaces httprouter.Router.PATCH.
func (r *Router) PATCH(path string, h httprouter.Handle) {
	r.handle(http.MethodPatch, path, h)
}

// POST replaces httprouter.Router.POST.
func (r *Router) POST(path string, h httprouter.Handle) {
	r.handle(http.MethodPost, path, h)
}

// PUT replaces httprouter.Router.PUT.
func (r *Router) PUT(path string, h httprouter.Handle) {
	r.handle(http.MethodPut, path, h)
}

// Handle replaces httprouter.Router.Handle.
func (r *Router) Handle(method, path string, h httprouter.Handle) {
	r.handle(method, path, h)
}

// Handler replaces httprouter.Router.Handler.
func (r *Router) Handler(method, path string, handler http.Handler) {
	_, h := newrelic.WrapHandle(r.application, path, handler)
	r.Router.Handler(method, path, h)
}

// HandlerFunc replaces httprouter.Router.HandlerFunc.
func (r *Router) HandlerFunc(method, path string, handler http.HandlerFunc) {
	r.Handler(method, path, handler)
}

// ServeHTTP replaces httprouter.Router.ServeHTTP.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if nil != r.application {
		h, _, _ := r.Router.Lookup(req.Method, req.URL.Path)
		if nil == h {
			txn := r.application.StartTransaction("NotFound")
			defer txn.End()

			req = newrelic.RequestWithTransactionContext(req, txn)

			txn.SetWebRequestHTTP(req)
			w = txn.SetWebResponse(w)
		}
	}

	r.Router.ServeHTTP(w, req)
}
