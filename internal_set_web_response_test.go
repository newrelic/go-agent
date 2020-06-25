// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/newrelic/go-agent/internal"
)

func TestTransactionStartedWithoutResponse(t *testing.T) {
	// Test that the http.ResponseWriter methods of the transaction can
	// safely be called if a ResponseWriter is not provided.
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	txn.WriteHeader(123)
	if hdr := txn.Header(); hdr != nil {
		t.Error(hdr)
	}
	n, err := txn.Write([]byte("should not panic"))
	if err != nil || n != 0 {
		t.Error(err, n)
	}
	txn.End()
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		AgentAttributes: map[string]interface{}{"httpResponseCode": 123},
		Intrinsics:      map[string]interface{}{"name": "OtherTransaction/Go/hello"},
	}})
}

func TestSetWebResponseNil(t *testing.T) {
	// Test that the http.ResponseWriter methods of the transaction can
	// safely be called if txn.SetWebResponse(nil) has been called.
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	txn = txn.SetWebResponse(nil)
	txn.WriteHeader(123)
	if hdr := txn.Header(); hdr != nil {
		t.Error(hdr)
	}
	n, err := txn.Write([]byte("should not panic"))
	if err != nil || n != 0 {
		t.Error(err, n)
	}
	txn.End()
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		AgentAttributes: map[string]interface{}{"httpResponseCode": 123},
		Intrinsics:      map[string]interface{}{"name": "OtherTransaction/Go/hello"},
	}})
}

func TestSetWebResponseSuccess(t *testing.T) {
	// Test that the http.ResponseWriter methods of the transaction use the
	// writer set by SetWebResponse.
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	w := httptest.NewRecorder()
	txn = txn.SetWebResponse(w)
	txn.WriteHeader(123)
	hdr := txn.Header()
	hdr.Set("zip", "zap")
	body := "should not panic"
	n, err := txn.Write([]byte(body))
	if err != nil || n != len(body) {
		t.Error(err, n)
	}
	txn.End()
	if w.Code != 123 {
		t.Error(w.Code)
	}
	if w.HeaderMap.Get("zip") != "zap" {
		t.Error(w.HeaderMap)
	}
	if w.Body.String() != body {
		t.Error(w.Body.String())
	}
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		AgentAttributes: map[string]interface{}{"httpResponseCode": 123},
		Intrinsics:      map[string]interface{}{"name": "OtherTransaction/Go/hello"},
	}})
}

type writerWithFlush struct{}

func (w writerWithFlush) Header() http.Header       { return nil }
func (w writerWithFlush) WriteHeader(int)           {}
func (w writerWithFlush) Write([]byte) (int, error) { return 0, nil }
func (w writerWithFlush) Flush()                    {}

func TestSetWebResponseTxnUpgraded(t *testing.T) {
	// Test that the using Transaction reference returned by SetWebResponse
	// properly has the optional methods that the ResponseWriter does.
	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	if _, ok := txn.(http.Flusher); ok {
		t.Error("should not have Flusher")
	}
	txn = txn.SetWebResponse(writerWithFlush{})
	if _, ok := txn.(http.Flusher); !ok {
		t.Error("should have Flusher now")
	}
}
