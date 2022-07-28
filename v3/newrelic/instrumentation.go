// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"net/http"
)

// instrumentation.go contains helpers built on the lower level api.

// WrapHandle instruments http.Handler handlers with Transactions.  To
// instrument this code:
//
//    http.Handle("/foo", myHandler)
//
// Perform this replacement:
//
//    http.Handle(newrelic.WrapHandle(app, "/foo", myHandler))
//
// WrapHandle adds the Transaction to the request's context.  Access it using
// FromContext to add attributes, create segments, or notice errors:
//
//	func myHandler(rw ResponseWriter, req *Request) {
//		txn := newrelic.FromContext(req.Context())
//		txn.AddAttribute("customerLevel", "gold")
//		io.WriteString(w, "users page")
//	}
//
// The WrapHandle function is safe to call if app is nil.
//
// WrapHandle accepts zero or more TraceOption functions to allow additional options to be
// manually added to the transaction trace generated, in the same fashion as StartTransaction
// does. For example, this can be used to control code level metrics generated for this transaction.
func WrapHandle(app *Application, pattern string, handler http.Handler, options ...TraceOption) (string, http.Handler) {
	if app == nil {
		return pattern, handler
	}
	// add the wrapped function to the trace options as the source code reference point
	// (to the beginning of the option list, so that the user can override this)
	// If we came here via WrapHandleFunc, that means it will have pushed the handler's
	// code location if possible, so the options will be (this code location), (handler function location),
	// (other options); that results in having a backup in case either failed, with preference
	// given to the handler's function location. In any event, the user may override all of that
	// with a location they choose.
	return pattern, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		options = append([]TraceOption{WithThisCodeLocation()}, options...)
		txn := app.StartTransaction(r.Method+" "+pattern, options...)
		defer txn.End()

		w = txn.SetWebResponse(w)
		txn.SetWebRequestHTTP(r)

		r = RequestWithTransactionContext(r, txn)

		handler.ServeHTTP(w, r)
	})
}

// WrapHandleFunc instruments handler functions using Transactions.  To
// instrument this code:
//
//	http.HandleFunc("/users", func(w http.ResponseWriter, req *http.Request) {
//		io.WriteString(w, "users page")
//	})
//
// Perform this replacement:
//
//	http.HandleFunc(newrelic.WrapHandleFunc(app, "/users", func(w http.ResponseWriter, req *http.Request) {
//		io.WriteString(w, "users page")
//	}))
//
// WrapHandleFunc adds the Transaction to the request's context.  Access it using
// FromContext to add attributes, create segments, or notice errors:
//
//	http.HandleFunc(newrelic.WrapHandleFunc(app, "/users", func(w http.ResponseWriter, req *http.Request) {
//		txn := newrelic.FromContext(req.Context())
//		txn.AddAttribute("customerLevel", "gold")
//		io.WriteString(w, "users page")
//	}))
//
// The WrapHandleFunc function is safe to call if app is nil.
//
// WrapHandleFunc accepts zero or more TraceOption functions to allow additional options to be
// manually added to the transaction trace generated, in the same fashion as StartTransaction
// does. For example, this can be used to control code level metrics generated for this transaction.
func WrapHandleFunc(app *Application, pattern string, handler func(http.ResponseWriter, *http.Request), options ...TraceOption) (string, func(http.ResponseWriter, *http.Request)) {
	// add the wrapped function to the trace options as the source code reference point
	// (to the beginning of the option list, so that the user can override this)
	options = append([]TraceOption{WithFunctionLocation(handler)}, options...)
	p, h := WrapHandle(app, pattern, http.HandlerFunc(handler), options...)
	return p, func(w http.ResponseWriter, r *http.Request) { h.ServeHTTP(w, r) }
}

// NewRoundTripper creates an http.RoundTripper to instrument external requests
// and add distributed tracing headers.  The http.RoundTripper returned creates
// an external segment before delegating to the original http.RoundTripper
// provided (or http.DefaultTransport if none is provided).  The
// http.RoundTripper will look for a Transaction in the request's context
// (using FromContext).
func NewRoundTripper(original http.RoundTripper) http.RoundTripper {
	if nil == original {
		original = http.DefaultTransport
	}
	return roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		// The specification of http.RoundTripper requires that the request is never modified.
		request = cloneRequest(request)
		segment := StartExternalSegment(nil, request)

		response, err := original.RoundTrip(request)

		segment.Response = response
		segment.End()

		return response, err
	})
}

// cloneRequest mimics implementation of
// https://godoc.org/github.com/google/go-github/github#BasicAuthTransport.RoundTrip
func cloneRequest(r *http.Request) *http.Request {
	// shallow copy of the struct
	r2 := new(http.Request)
	*r2 = *r
	// deep copy of the Header
	r2.Header = make(http.Header, len(r.Header))
	for k, s := range r.Header {
		r2.Header[k] = append([]string(nil), s...)
	}
	return r2
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
