package nrgin

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/internal"
)

var (
	pkg = "github.com/newrelic/go-agent/_integrations/nrgin/v1"
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

func hello(c *gin.Context) {
	c.Writer.WriteString("hello response")
}

func TestBasicRoute(t *testing.T) {
	app := testApp(t)
	router := gin.Default()
	router.Use(Middleware(app))
	router.GET("/hello", hello)

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/hello", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "hello response" {
		t.Error("wrong response body", respBody)
	}
	app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction/Go/" + pkg + ".hello", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/" + pkg + ".hello", Scope: "", Forced: false, Data: nil},
	})
}

func TestRouterGroup(t *testing.T) {
	app := testApp(t)
	router := gin.Default()
	router.Use(Middleware(app))
	group := router.Group("/group")
	group.GET("/hello", hello)

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/group/hello", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "hello response" {
		t.Error("wrong response body", respBody)
	}
	app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction/Go/" + pkg + ".hello", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/" + pkg + ".hello", Scope: "", Forced: false, Data: nil},
	})
}

func TestAnonymousHandler(t *testing.T) {
	app := testApp(t)
	router := gin.Default()
	router.Use(Middleware(app))
	router.GET("/anon", func(c *gin.Context) {
		c.Writer.WriteString("anonymous function handler")
	})

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/anon", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "anonymous function handler" {
		t.Error("wrong response body", respBody)
	}
	app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction/Go/" + pkg + ".TestAnonymousHandler.func1", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/" + pkg + ".TestAnonymousHandler.func1", Scope: "", Forced: false, Data: nil},
	})
}

func multipleWriteHeader(c *gin.Context) {
	// Unlike http.ResponseWriter, gin.ResponseWriter does not immediately
	// write the first WriteHeader.  Instead, it gets buffered until the
	// first Write call.
	c.Writer.WriteHeader(200)
	c.Writer.WriteHeader(500)
	c.Writer.WriteString("multipleWriteHeader")
}

func TestMultipleWriteHeader(t *testing.T) {
	app := testApp(t)
	router := gin.Default()
	router.Use(Middleware(app))
	router.GET("/header", multipleWriteHeader)

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/header", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "multipleWriteHeader" {
		t.Error("wrong response body", respBody)
	}
	if response.Code != 500 {
		t.Error("wrong response code", response.Code)
	}
	app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction/Go/" + pkg + ".multipleWriteHeader", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/" + pkg + ".multipleWriteHeader", Scope: "", Forced: false, Data: nil},
		// Error metrics test the 500 response code capture.
		{Name: "Errors/all", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Errors/allWeb", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Errors/WebTransaction/Go/" + pkg + ".multipleWriteHeader", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
	})
}

func accessTransactionGinContext(c *gin.Context) {
	if txn := Transaction(c); nil != txn {
		txn.NoticeError(errors.New("problem"))
	}
	c.Writer.WriteString("accessTransactionGinContext")
}

func TestContextTransaction(t *testing.T) {
	app := testApp(t)
	router := gin.Default()
	router.Use(Middleware(app))
	router.GET("/txn", accessTransactionGinContext)

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/txn", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "accessTransactionGinContext" {
		t.Error("wrong response body", respBody)
	}
	if response.Code != 200 {
		t.Error("wrong response code", response.Code)
	}
	app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction/Go/" + pkg + ".accessTransactionGinContext", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/" + pkg + ".accessTransactionGinContext", Scope: "", Forced: false, Data: nil},
		{Name: "Errors/all", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Errors/allWeb", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Errors/WebTransaction/Go/" + pkg + ".accessTransactionGinContext", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
	})
}

func TestNilApp(t *testing.T) {
	var app newrelic.Application
	router := gin.Default()
	router.Use(Middleware(app))
	router.GET("/hello", hello)

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/hello", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "hello response" {
		t.Error("wrong response body", respBody)
	}
}
