package nrgochi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
)

func hello(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello"))
}

func TestBasicRoute(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	router := chi.NewRouter()
	router.Use(Middleware(app.Application))
	router.Get("/hello", hello)

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/hello", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "hello" {
		t.Error("wrong response body", respBody)
	}
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:          "GET /hello",
		IsWeb:         true,
		UnknownCaller: true,
	})
}

func TestAnonymousFunctions(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	router := chi.NewRouter()
	router.Use(Middleware(app.Application))
	router.Get("/helloAnon", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello anon"))
	})

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/helloAnon", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "hello anon" {
		t.Error("wrong response body", respBody)
	}
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:          "GET /helloAnon",
		IsWeb:         true,
		UnknownCaller: true,
	})

}
