// Package nrgorilla instruments https://github.com/gorilla/mux applications.
//
// Use this package to instrument inbound requests handled by a gorilla
// mux.Router.  Call nrgorilla.InstrumentRoutes on your gorilla mux.Router
// after your routes have been added to it.
//
// Example: https://github.com/newrelic/go-agent/tree/master/v3/integrations/nrgorilla/example/main.go
package nrgorilla

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/newrelic/go-agent/v3/internal"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)

func init() { internal.TrackUsage("integration", "framework", "gorilla", "v1") }

type instrumentedHandler struct {
	app  *newrelic.Application
	orig http.Handler
}

func (h instrumentedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := routeName(r)
	txn := h.app.StartTransaction(name)
	txn.SetWebRequestHTTP(r)
	w = txn.SetWebResponse(w)
	defer txn.End()

	r = newrelic.RequestWithTransactionContext(r, txn)

	h.orig.ServeHTTP(w, r)
}

func instrumentRoute(h http.Handler, app *newrelic.Application) http.Handler {
	if _, ok := h.(instrumentedHandler); ok {
		return h
	}
	return instrumentedHandler{
		orig: h,
		app:  app,
	}
}

func routeName(r *http.Request) string {
	route := mux.CurrentRoute(r)
	if nil == route {
		return "NotFoundHandler"
	}
	if n := route.GetName(); n != "" {
		return n
	}
	if n, _ := route.GetPathTemplate(); n != "" {
		return r.Method + " " + n
	}
	n, _ := route.GetHostTemplate()
	return r.Method + " " + n
}

// InstrumentRoutes instruments requests through the provided mux.Router.  Use
// this after the routes have been added to the router.
func InstrumentRoutes(r *mux.Router, app *newrelic.Application) *mux.Router {
	if app != nil {
		r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
			h := instrumentRoute(route.GetHandler(), app)
			route.Handler(h)
			return nil
		})
		if nil != r.NotFoundHandler {
			r.NotFoundHandler = instrumentRoute(r.NotFoundHandler, app)
		}
	}
	return r
}
