// +build go1.7

package nrgin

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/newrelic/go-agent/internal"
)

func accessTransactionContextContext(c *gin.Context) {
	var ctx context.Context = c
	// Transaction is designed to take both a context.Context and a
	// *gin.Context.
	if txn := Transaction(ctx); nil != txn {
		txn.NoticeError(errors.New("problem"))
	}
	c.Writer.WriteString("accessTransactionContextContext")
}

func TestContextContextTransaction(t *testing.T) {
	app := testApp(t)
	router := gin.Default()
	router.Use(Middleware(app))
	router.GET("/txn", accessTransactionContextContext)

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/txn", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "accessTransactionContextContext" {
		t.Error("wrong response body", respBody)
	}
	if response.Code != 200 {
		t.Error("wrong response code", response.Code)
	}
	app.(internal.Expect).ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction/Go/" + pkg + ".accessTransactionContextContext", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex", Scope: "", Forced: true, Data: nil},
		{Name: "Apdex/Go/" + pkg + ".accessTransactionContextContext", Scope: "", Forced: false, Data: nil},
		{Name: "Errors/all", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Errors/allWeb", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "Errors/WebTransaction/Go/" + pkg + ".accessTransactionContextContext", Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
	})
}

func TestContextWithoutTransaction(t *testing.T) {
	txn := Transaction(context.Background())
	if txn != nil {
		t.Error("didn't expect a transaction", txn)
	}
	ctx := context.WithValue(context.Background(), ctxKey, 123)
	txn = Transaction(ctx)
	if txn != nil {
		t.Error("didn't expect a transaction", txn)
	}
}
