// Package nrgorilla introduces to support for the gorilla/mux framework.  See
// examples/_gorilla/main.go for an example.
package nrgorilla

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/internal"
)

func init() { internal.TrackUsage("integration", "framework", "gorilla", "v1") }

type instrumentedHandler struct {
	name string
	app  newrelic.Application
	orig http.Handler
}

func (h instrumentedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	txn := h.app.StartTransaction(h.name, w, r)
	r = r.WithContext(NewContext(r.Context(), txn))
	defer txn.End()

	h.orig.ServeHTTP(txn, r)
}

func instrumentRoute(h http.Handler, app newrelic.Application, name string) http.Handler {
	if _, ok := h.(instrumentedHandler); ok {
		return h
	}
	return instrumentedHandler{
		name: name,
		orig: h,
		app:  app,
	}
}

func routeName(route *mux.Route) string {
	if nil == route {
		return ""
	}
	if n := route.GetName(); n != "" {
		return n
	}
	if n, _ := route.GetPathTemplate(); n != "" {
		return n
	}
	n, _ := route.GetHostTemplate()
	return n
}

// InstrumentRoutes adds instrumentation to a router.  This must be used after
// the routes have been added to the router.
func InstrumentRoutes(r *mux.Router, app newrelic.Application) *mux.Router {
	r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		h := instrumentRoute(route.GetHandler(), app, routeName(route))
		route.Handler(h)
		return nil
	})
	if nil != r.NotFoundHandler {
		r.NotFoundHandler = instrumentRoute(r.NotFoundHandler, app, "NotFoundHandler")
	}
	return r
}

// NewContext returns a new Context that carries the provided transcation.
func NewContext(ctx context.Context, txn newrelic.Transaction) context.Context {
	return context.WithValue(ctx, contextKey, txn)
}

// FromContext returns the Transaction in the context, if any. If there
// isn't a transaction in the context, nil is returned.
func FromContext(ctx context.Context) newrelic.Transaction {
	h, _ := ctx.Value(contextKey).(newrelic.Transaction)
	return h
}

type contextKeyType struct{}

// globally unique, since it is an unexported type
var contextKey = contextKeyType(struct{}{})
