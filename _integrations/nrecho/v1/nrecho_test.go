package nrecho

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/internal"
)

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

	e := echo.New()
	e.Use(Middleware(app))
	e.GET("/hello", func(c echo.Context) error {
		return c.Blob(http.StatusOK, "text/html", []byte("Hello, World!"))
	})

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/hello", nil)
	if err != nil {
		t.Fatal(err)
	}

	e.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "Hello, World!" {
		t.Error("wrong response body", respBody)
	}
	app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
	})
	app.(internal.Expect).ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             "WebTransaction/Go/hello",
			"nr.apdexPerfZone": "S",
		},
		AgentAttributes: map[string]interface{}{
			"httpResponseCode":             "200",
			"request.method":               "GET",
			"response.headers.contentType": "text/html",
		},
		UserAttributes: map[string]interface{}{},
	}})
}

func TestNilApp(t *testing.T) {
	e := echo.New()
	e.Use(Middleware(nil))
	e.GET("/hello", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/hello", nil)
	if err != nil {
		t.Fatal(err)
	}

	e.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "Hello, World!" {
		t.Error("wrong response body", respBody)
	}
}

func TestTransactionContext(t *testing.T) {
	app := testApp(t)

	e := echo.New()
	e.Use(Middleware(app))
	e.GET("/hello", func(c echo.Context) error {
		txn := FromContext(c)
		if nil != txn {
			txn.NoticeError(errors.New("ooops"))
		}
		return c.String(http.StatusOK, "Hello, World!")
	})

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/hello", nil)
	if err != nil {
		t.Fatal(err)
	}

	e.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "Hello, World!" {
		t.Error("wrong response body", respBody)
	}
	app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "Errors/WebTransaction/Go/hello", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Errors/all", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Errors/allWeb", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
	})
}

func TestNoticedErrors(t *testing.T) {
	app := testApp(t)

	e := echo.New()
	e.Use(Middleware(app))
	e.GET("/hello", func(c echo.Context) error {
		return errors.New("ooooooooops")
	})

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/hello", nil)
	if err != nil {
		t.Fatal(err)
	}

	e.ServeHTTP(response, req)
	app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "Errors/WebTransaction/Go/hello", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Errors/all", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Errors/allWeb", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
	})
}

func TestNotFoundHandler(t *testing.T) {
	app := testApp(t)

	e := echo.New()
	e.Use(Middleware(app))

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/hello", nil)
	if err != nil {
		t.Fatal(err)
	}

	e.ServeHTTP(response, req)
	app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/NotFoundHandler", Scope: "", Forced: false, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction/Go/NotFoundHandler", Scope: "", Forced: true, Data: nil},
	})
}

func TestMethodNotAllowedHandler(t *testing.T) {
	app := testApp(t)

	e := echo.New()
	e.Use(Middleware(app))
	e.GET("/hello", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})

	response := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/hello", nil)
	if err != nil {
		t.Fatal(err)
	}

	e.ServeHTTP(response, req)
	app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/MethodNotAllowedHandler", Scope: "", Forced: false, Data: nil},
		{Name: "Errors/WebTransaction/Go/MethodNotAllowedHandler", Scope: "",
			Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Errors/all", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Errors/allWeb", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction/Go/MethodNotAllowedHandler", Scope: "", Forced: true, Data: nil},
	})
}
