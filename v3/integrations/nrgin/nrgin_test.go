package nrgin

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	"github.com/newrelic/go-agent/v3/newrelic"
)

var (
	pkg = "github.com/newrelic/go-agent/v3/integrations/nrgin"
)

func hello(c *gin.Context) {
	c.Writer.WriteString("hello response")
}

func TestBasicRoute(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	router := gin.Default()
	router.Use(Middleware(app.Application))
	router.GET("/hello", hello)

	txnName := "GET " + pkg + ".hello"
	if useFullPathVersion(gin.Version) {
		txnName = "GET /hello"
	}

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/hello", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "hello response" {
		t.Error("wrong response body", respBody)
	}
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:  txnName,
		IsWeb: true,
	})
}

func TestRouterGroup(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	router := gin.Default()
	router.Use(Middleware(app.Application))
	group := router.Group("/group")
	group.GET("/hello", hello)

	txnName := "GET " + pkg + ".hello"
	if useFullPathVersion(gin.Version) {
		txnName = "GET /group/hello"
	}

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/group/hello", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "hello response" {
		t.Error("wrong response body", respBody)
	}
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:  txnName,
		IsWeb: true,
	})
}

func TestAnonymousHandler(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	router := gin.Default()
	router.Use(Middleware(app.Application))
	router.GET("/anon", func(c *gin.Context) {
		c.Writer.WriteString("anonymous function handler")
	})

	txnName := "GET " + pkg + ".TestAnonymousHandler.func1"
	if useFullPathVersion(gin.Version) {
		txnName = "GET /anon"
	}

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/anon", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "anonymous function handler" {
		t.Error("wrong response body", respBody)
	}
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:  txnName,
		IsWeb: true,
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
	app := integrationsupport.NewBasicTestApp()
	router := gin.Default()
	router.Use(Middleware(app.Application))
	router.GET("/header", multipleWriteHeader)

	txnName := "GET " + pkg + ".multipleWriteHeader"
	if useFullPathVersion(gin.Version) {
		txnName = "GET /header"
	}

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
	// Error metrics test the 500 response code capture.
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:      txnName,
		IsWeb:     true,
		NumErrors: 1,
	})
}

func accessTransactionGinContext(c *gin.Context) {
	txn := Transaction(c)
	txn.NoticeError(errors.New("problem"))
	c.Writer.WriteString("accessTransactionGinContext")
}

func TestContextTransaction(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	router := gin.Default()
	router.Use(Middleware(app.Application))
	router.GET("/txn", accessTransactionGinContext)

	txnName := "GET " + pkg + ".accessTransactionGinContext"
	if useFullPathVersion(gin.Version) {
		txnName = "GET /txn"
	}

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
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:      txnName,
		IsWeb:     true,
		NumErrors: 1,
	})
}

func TestNilApp(t *testing.T) {
	var app *newrelic.Application
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

func errorStatus(c *gin.Context) {
	c.String(500, "an error happened")
}

// The Gin.Context.Status method behavior changed with this pull
// request: https://github.com/gin-gonic/gin/pull/1606.  This change
// affects our ability to instrument the response code. In Gin v1.4.0
// and below, we always recorded a 200 status, whereas with newer Gin
// versions we now correctly capture the status.
var statusFixVersion = [...]string{"1", "5"}

// Gin added the FullPath method to the Gin.Context in this version. When
// available, we use this method to set the transaction name.
var fullPathVersion = [...]string{"1", "5"}

func useFullPathVersion(v string) bool {
	return checkVersionIsAtLeast(v, fullPathVersion)
}

func useStatusFixVersion(v string) bool {
	return checkVersionIsAtLeast(v, statusFixVersion)
}

func checkVersionIsAtLeast(checkV string, checkAgainst [2]string) bool {
	parts := strings.Split(strings.TrimPrefix(checkV, "v"), ".")
	if len(parts) < 2 {
		return false
	}
	if parts[0] < checkAgainst[0] {
		return false
	}
	return parts[1] >= checkAgainst[1]
}

func TestStatusCodes(t *testing.T) {
	// Test that we are correctly able to collect status code.
	expectCode := 200
	if useStatusFixVersion(gin.Version) {
		expectCode = 500
	}
	app := integrationsupport.NewBasicTestApp()
	router := gin.Default()
	router.Use(Middleware(app.Application))
	router.GET("/err", errorStatus)

	txnName := "WebTransaction/Go/GET " + pkg + ".errorStatus"
	if useFullPathVersion(gin.Version) {
		txnName = "WebTransaction/Go/GET /err"
	}

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/err", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "an error happened" {
		t.Error("wrong response body", respBody)
	}
	if response.Code != 500 {
		t.Error("wrong response code", response.Code)
	}
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             txnName,
			"nr.apdexPerfZone": internal.MatchAnything,
		},
		UserAttributes: map[string]interface{}{},
		AgentAttributes: map[string]interface{}{
			"httpResponseCode":             expectCode,
			"http.statusCode":              expectCode,
			"request.method":               "GET",
			"request.uri":                  "/err",
			"response.headers.contentType": "text/plain; charset=utf-8",
		},
	}})
}

func noBody(c *gin.Context) {
	c.Status(500)
}

func TestNoResponseBody(t *testing.T) {
	// Test that when no response body is sent (i.e. c.Writer.Write is never
	// called) that we still capture status code.
	expectCode := 200
	if useFullPathVersion(gin.Version) {
		expectCode = 500
	}
	app := integrationsupport.NewBasicTestApp()
	router := gin.Default()
	router.Use(Middleware(app.Application))
	router.GET("/nobody", noBody)

	txnName := "WebTransaction/Go/GET " + pkg + ".noBody"
	if useFullPathVersion(gin.Version) {
		txnName = "WebTransaction/Go/GET /nobody"
	}

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/nobody", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "" {
		t.Error("wrong response body", respBody)
	}
	if response.Code != 500 {
		t.Error("wrong response code", response.Code)
	}
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"name":             txnName,
			"nr.apdexPerfZone": internal.MatchAnything,
		},
		UserAttributes: map[string]interface{}{},
		AgentAttributes: map[string]interface{}{
			"httpResponseCode": expectCode,
			"http.statusCode":  expectCode,
			"request.method":   "GET",
			"request.uri":      "/nobody",
		},
	}})
}

func TestRouteWithParams(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	router := gin.Default()
	router.Use(Middleware(app.Application))
	router.GET("/hello/:name/*action", hello)

	txnName := "GET " + pkg + ".hello"
	if useFullPathVersion(gin.Version) {
		// ensure the transaction is named after the route and not the url
		txnName = "GET /hello/:name/*action"
	}

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/hello/world/fun", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "hello response" {
		t.Error("wrong response body", respBody)
	}
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:  txnName,
		IsWeb: true,
	})
}

func TestMiddlewareOldNaming(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	router := gin.Default()
	router.Use(MiddlewareHandlerTxnNames(app.Application))
	router.GET("/hello", hello)

	txnName := "GET " + pkg + ".hello"

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/hello", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "hello response" {
		t.Error("wrong response body", respBody)
	}
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:  txnName,
		IsWeb: true,
	})
}
