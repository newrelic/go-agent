// Copyright 2022 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nriris

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)

func accessTransactionContextContext(ctx iris.Context) {
	var c context.Context = ctx
	// Transaction is designed to take both a context.Context and an
	// iris.Context.
	txn := Transaction(c)
	txn.NoticeError(errors.New("problem"))
	ctx.WriteString("accessTransactionContextContext")
}

func TestContextContextTransaction(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	router := iris.New()
	router.Use(Middleware(app.Application))
	router.Get("/txn", accessTransactionContextContext)

	if err := router.Build(); err != nil {
		t.Fatal(err)
	}

	txnName := "GET " + pkg + ".accessTransactionContextContext"

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

func accessTransactionFromContext(ctx iris.Context) {
	// This tests that FromContext will find the transaction added to a
	// *iris.Context and by nriris.Middleware.
	txn := newrelic.FromContext(ctx)
	txn.NoticeError(errors.New("problem"))
	ctx.WriteString("accessTransactionFromContext")
}

func TestFromContext(t *testing.T) {
	app := integrationsupport.NewBasicTestApp()
	router := iris.New()
	router.Use(Middleware(app.Application))
	router.Get("/txn", accessTransactionFromContext)

	if err := router.Build(); err != nil {
		t.Fatal(err)
	}

	txnName := "GET " + pkg + ".accessTransactionFromContext"

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
	// This tests that nriris.Transaction will find a transaction added to
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
