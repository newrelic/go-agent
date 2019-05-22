package newrelic

import "net/http"

// instrumentation.go contains helpers built on the lower level api.

// WrapHandle instruments http.Handler handlers with transactions.  To
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
//		if txn := newrelic.FromContext(req.Context()); nil != txn {
//			txn.AddAttribute("customerLevel", "gold")
//		}
//	}
//
// This function is safe to call if 'app' is nil.
func WrapHandle(app Application, pattern string, handler http.Handler) (string, http.Handler) {
	if app == nil {
		return pattern, handler
	}
	return pattern, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		txn := app.StartTransaction(pattern, w, r)
		defer txn.End()

		r = RequestWithTransactionContext(r, txn)

		handler.ServeHTTP(txn, r)
	})
}

// WrapHandleFunc serves the same purpose as WrapHandle for functions registered
// with ServeMux.HandleFunc.
func WrapHandleFunc(app Application, pattern string, handler func(http.ResponseWriter, *http.Request)) (string, func(http.ResponseWriter, *http.Request)) {
	p, h := WrapHandle(app, pattern, http.HandlerFunc(handler))
	return p, func(w http.ResponseWriter, r *http.Request) { h.ServeHTTP(w, r) }
}

// NewRoundTripper creates an http.RoundTripper to instrument external requests
// without using StartExternalSegment.  The RoundTripper returned creates an
// external segment before delegating to the original RoundTripper provided (or
// http.DefaultTransport if none is provided).  If the Transaction parameter is
// nil then the RoundTripper will look for a Transaction in the request's
// context (using FromContext).  Using a nil Transaction is STRONGLY recommended
// because it allows the same RoundTripper (and client) to be reused for
// multiple transactions.
func NewRoundTripper(txn Transaction, original http.RoundTripper) http.RoundTripper {
	return roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		segment := StartExternalSegment(txn, request)

		if nil == original {
			original = http.DefaultTransport
		}
		response, err := original.RoundTrip(request)

		segment.Response = response
		segment.End()

		return response, err
	})
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
