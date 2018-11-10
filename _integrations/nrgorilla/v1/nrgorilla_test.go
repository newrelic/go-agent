package nrgorilla

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/internal"
)

func makeHandler(text string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(text))
	})
}

func testApp(t *testing.T) newrelic.Application {
	cfg := newrelic.NewConfig("appname", "0123456789012345678901234567890123456789")
	cfg.Enabled = false
	app, err := newrelic.NewApplication(cfg)
	if nil != err {
		t.Fatal(err)
	}
	internal.HarvestTesting(app, nil)
	return app
}

func TestBasicRoute(t *testing.T) {
	app := testApp(t)
	r := mux.NewRouter()
	r.Handle("/alpha", makeHandler("alpha response"))
	InstrumentRoutes(r, app)
	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/alpha", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "alpha response" {
		t.Error("wrong response body", respBody)
	}
	expect, _ := app.(internal.Expect)
	expect.ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction/Go/alpha", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/alpha", Scope: "", Forced: false, Data: nil},
	})
}

func TestSubrouterRoute(t *testing.T) {
	app := testApp(t)
	r := mux.NewRouter()
	users := r.PathPrefix("/users").Subrouter()
	users.Handle("/add", makeHandler("adding user"))
	InstrumentRoutes(r, app)
	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/users/add", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "adding user" {
		t.Error("wrong response body", respBody)
	}
	expect, _ := app.(internal.Expect)
	expect.ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction/Go/users/add", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/users/add", Scope: "", Forced: false, Data: nil},
	})
}

func TestNamedRoute(t *testing.T) {
	app := testApp(t)
	r := mux.NewRouter()
	r.Handle("/named", makeHandler("named route")).Name("special-name-route")
	InstrumentRoutes(r, app)
	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/named", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "named route" {
		t.Error("wrong response body", respBody)
	}
	expect, _ := app.(internal.Expect)
	expect.ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction/Go/special-name-route", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/special-name-route", Scope: "", Forced: false, Data: nil},
	})
}

func TestRouteNotFound(t *testing.T) {
	app := testApp(t)
	r := mux.NewRouter()
	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte("not found"))
	})
	// Tests that routes do not get double instrumented when
	// InstrumentRoutes is called twice by expecting error metrics with a
	// count of 1.
	InstrumentRoutes(r, app)
	InstrumentRoutes(r, app)
	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/foo", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "not found" {
		t.Error("wrong response body", respBody)
	}
	if response.Code != 500 {
		t.Error("wrong response code", response.Code)
	}
	expect, _ := app.(internal.Expect)
	expect.ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction/Go/NotFoundHandler", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/NotFoundHandler", Scope: "", Forced: false, Data: nil},
		// Error metrics test the 500 response code capture.
		{Name: "Errors/all", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Errors/allWeb", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Errors/WebTransaction/Go/NotFoundHandler", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
	})
}
