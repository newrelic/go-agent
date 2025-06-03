// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrgin

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/newrelic/go-agent/v3/internal"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
	"github.com/newrelic/go-agent/v3/newrelic/integrationsupport"
)

func accessTransactionContextContext(c *gin.Context) {
	var ctx context.Context = c
	// Transaction is designed to take both a context.Context and a
	// *gin.Context.
	txn := Transaction(ctx)
	txn.NoticeError(errors.New("problem"))
	c.Writer.WriteString("accessTransactionContextContext")
}

func TestContextContextTransaction(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	router := gin.Default()
	router.Use(Middleware(app.Application))
	router.GET("/txn", accessTransactionContextContext)

	txnName := "GET " + pkg + ".accessTransactionContextContext"
	if useFullPathVersion(gin.Version) {
		txnName = "GET /txn"
	}

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
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:          txnName,
		IsWeb:         true,
		NumErrors:     1,
		UnknownCaller: true,
		ErrorByCaller: true,
	})
}

func accessTransactionFromContext(c *gin.Context) {
	// This tests that FromContext will find the transaction added to a
	// *gin.Context and by nrgin.Middleware.
	txn := newrelic.FromContext(c)
	txn.NoticeError(errors.New("problem"))
	c.Writer.WriteString("accessTransactionFromContext")
}

func TestFromContext(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	router := gin.Default()
	router.Use(Middleware(app.Application))
	router.GET("/txn", accessTransactionFromContext)

	txnName := "GET " + pkg + ".accessTransactionFromContext"
	if useFullPathVersion(gin.Version) {
		txnName = "GET /txn"
	}

	response := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/txn", nil)
	if err != nil {
		t.Fatal(err)
	}
	router.ServeHTTP(response, req)
	if respBody := response.Body.String(); respBody != "accessTransactionFromContext" {
		t.Error("wrong response body", respBody)
	}
	if response.Code != 200 {
		t.Error("wrong response code", response.Code)
	}
	app.ExpectTxnMetrics(t, internal.WantTxn{
		Name:          txnName,
		IsWeb:         true,
		NumErrors:     1,
		UnknownCaller: true,
		ErrorByCaller: true,
	})
}

func TestContextWithoutTransaction(t *testing.T) {
	txn := Transaction(context.Background())
	if txn != nil {
		t.Error("didn't expect a transaction", txn)
	}
	ctx := context.WithValue(context.Background(), internal.TransactionContextKey, 123)
	txn = Transaction(ctx)
	if txn != nil {
		t.Error("didn't expect a transaction", txn)
	}
}

func TestNewContextTransaction(t *testing.T) {
	// This tests that nrgin.Transaction will find a transaction added to
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
