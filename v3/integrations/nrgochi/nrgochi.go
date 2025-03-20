// Package nrgochi instruments https://github.com/go-chi/chi applications.
//
// Use this package to instrument inbound requests handled by a chi.Router.
// Call nrgochi.Middleware to get a chi.Middleware which can be added to your
// application as a middleware:
//
//	router := chi.NewRouter()
//	// Add the nrgochi middleware before other middlewares or routes:
//	router.Use(nrgochi.Middleware(app))
//
// Example: https://github.com/newrelic/go-agent/tree/master/v3/integrations/nrgochi/example/main.go
package nrgochi

import (
	"net/http"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func init() { internal.TrackUsage("integration", "framework", "gochi", "v1") }

// headerResponseWriter gives the transaction access to response headers and the
// response code.
type headerResponseWriter struct{ w http.ResponseWriter }

func (w *headerResponseWriter) Header() http.Header       { return w.w.Header() }
func (w *headerResponseWriter) Write([]byte) (int, error) { return 0, nil }
func (w *headerResponseWriter) WriteHeader(int)           {}

var _ http.ResponseWriter = &headerResponseWriter{}

// replacementResponseWriter mimics the behavior of http.ResponseWriter which
// buffers the response code rather than writing it when
// http.ResponseWriter.WriteHeader is called.
type replacementResponseWriter struct {
	http.ResponseWriter
	replacement http.ResponseWriter
	code        int
	written     bool
}

var _ http.ResponseWriter = &replacementResponseWriter{}

func (w *replacementResponseWriter) flushHeader() {
	if !w.written {
		w.replacement.WriteHeader(w.code)
		w.written = true
	}
}

func (w *replacementResponseWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *replacementResponseWriter) Write(data []byte) (int, error) {
	w.flushHeader()
	if newrelic.IsSecurityAgentPresent() {
		w.replacement.Write(data)
	}
	return w.ResponseWriter.Write(data)
}

func (w *replacementResponseWriter) WriteString(s string) (int, error) {
	w.flushHeader()
	if newrelic.IsSecurityAgentPresent() {
		w.replacement.Write([]byte(s))
	}
	return w.ResponseWriter.Write([]byte(s))
}

// Middleware creates a Chi middleware that instruments requests.
//
//	router := chi.NewRouter()
//	// Add the nrgochi middleware before other middlewares or routes:
//	router.Use(nrgochi.Middleware(app))
func Middleware(app *newrelic.Application) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			traceID := ""
			if app != nil {
				name := r.Method + " " + r.URL.Path

				hdrWriter := &headerResponseWriter{w: w}
				txn := app.StartTransaction(name)
				if newrelic.IsSecurityAgentPresent() {
					txn.SetCsecAttributes(newrelic.AttributeCsecRoute, r.URL.Path)
				}
				txn.SetWebRequestHTTP(r)
				defer txn.End()

				repl := &replacementResponseWriter{
					ResponseWriter: w,
					replacement:    txn.SetWebResponse(hdrWriter),
					code:           http.StatusOK,
				}
				w = repl
				defer repl.flushHeader()

				ctx := newrelic.NewContext(r.Context(), txn)
				r = r.WithContext(ctx)
				traceID = txn.GetLinkingMetadata().TraceID
			}
			next.ServeHTTP(w, r)
			if newrelic.IsSecurityAgentPresent() {
				newrelic.GetSecurityAgentInterface().SendEvent("RESPONSE_HEADER", w.Header(), traceID)
			}
		})
	}
}
