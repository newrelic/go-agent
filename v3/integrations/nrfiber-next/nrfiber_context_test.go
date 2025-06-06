package nrfiber

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/stretchr/testify/assert"
)

// TestMiddleware_ContextPropagation tests that the middleware correctly propagates the transaction context
func TestMiddleware_ContextPropagation(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	router := fiber.New()
	router.Use(Middleware(app.Application))
	router.Get("/txn", func(c fiber.Ctx) error {
		txn := FromContext(c)
		if txn == nil {
			t.Error("Transaction is nil")
		}
		if txn.Name() != "GET /txn" {
			t.Error("wrong transaction name", txn.Name())
		}
		txn.NoticeError(errors.New("ooops"))
		c.WriteString("accessTransactionFromContext")
		return nil
	})

	req, err := http.NewRequest("GET", "/txn", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := router.Test(req)
	body, _ := io.ReadAll(resp.Body)

	if respBody := string(body); respBody != "accessTransactionFromContext" {
		t.Error("wrong response body", respBody)
	}

	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:          "GET /txn",
		IsWeb:         true,
		NumErrors:     1,
		UnknownCaller: true,
		ErrorByCaller: true,
	})
}

// Test_Transaction tests that the Transaction function retrieves the
// transaction from the context correctly.
// It also tests the behavior when the context is nil or does not contain
// a transaction.
func Test_Transaction(t *testing.T) {
	// Create a mock transaction
	mockTxn := &newrelic.Transaction{}

	// Create a context with the transaction
	ctx := context.WithValue(context.Background(), internal.TransactionContextKey, mockTxn)

	// Retrieve the transaction from the context
	resultTxn := Transaction(ctx)

	// Verify the transaction was correctly retrieved
	assert.Equal(t, mockTxn, resultTxn)

	// Test with nil context
	assert.Nil(t, Transaction(nil))

	// Test with context but no transaction
	emptyCtx := context.Background()
	assert.Nil(t, Transaction(emptyCtx))
}

// TestNewContext_Transaction tests that nrfiber.Transaction will find a transaction
// added to a context using newrelic.NewContext.
func TestNewContext_Transaction(t *testing.T) {
	// This tests that nrfiber.Transaction will find a transaction added to
	// to a context using newrelic.NewContext.
	app := integrationsupport.NewBasicTestApp()
	txn := app.StartTransaction("name")
	ctx := newrelic.NewContext(context.Background(), txn)
	if tx := Transaction(ctx); nil != tx {
		tx.NoticeError(errors.New("problem"))
	}
	txn.End()

	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:          "name",
		IsWeb:         false,
		NumErrors:     1,
		UnknownCaller: true,
		ErrorByCaller: true,
	})
}
