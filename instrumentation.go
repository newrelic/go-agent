package newrelic

import "net/http"

// WrapHandle facilitates instrumentation of handlers registered with an
// http.ServeMux.  As an example, to instrument this code:
//
//    http.Handle("/foo", fooHandler)
//
// Perform the following replacement:
//
//    http.Handle(newrelic.WrapHandle(app, "/foo", fooHandler))
//
// The Transaction is passed to the handler in place of the original
// http.ResponseWriter, so it can be accessed in your handler by type assertion.
// For example, to rename the transaction:
//
//	// 'w' is the name of the handler's http.ResponseWriter parameter.
//	if txn, ok := w.(newrelic.Transaction); ok {
//		txn.SetName("other-name")
//	}
//
func WrapHandle(app Application, pattern string, handler http.Handler) (string, http.Handler) {
	return pattern, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		txn := app.StartTransaction(pattern, w, r)
		defer txn.End()

		handler.ServeHTTP(txn, r)
	})
}

// WrapHandleFunc serves the same purpose as WrapHandle for functions registered
// with ServeMux.HandleFunc.
func WrapHandleFunc(app Application, pattern string, handler func(http.ResponseWriter, *http.Request)) (string, func(http.ResponseWriter, *http.Request)) {
	p, h := WrapHandle(app, pattern, http.HandlerFunc(handler))
	return p, func(w http.ResponseWriter, r *http.Request) { h.ServeHTTP(w, r) }
}
