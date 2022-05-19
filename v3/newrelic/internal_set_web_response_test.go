// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/newrelic/go-agent/v3/internal"
)

func TestSetWebResponseNil(t *testing.T) {
	// Test that the methods of the txn.SetWebResponse(nil) return value
	// writer can safely be called.
	app := testApp(nil, ConfigDistributedTracerEnabled(false), t)
	txn := app.StartTransaction("hello")
	rw := txn.SetWebResponse(nil)
	rw.WriteHeader(123)
	if hdr := rw.Header(); hdr != nil {
		t.Error(hdr)
	}
	n, err := rw.Write([]byte("should not panic"))
	if err != nil || n != 0 {
		t.Error(err, n)
	}
	txn.End()
	app.ExpectTxnEvents(t, []internal.WantEvent{{
		AgentAttributes: map[string]interface{}{
			"httpResponseCode": 123,
			"http.statusCode":  123,
		},
		Intrinsics: map[string]interface{}{"name": "OtherTransaction/Go/hello"},
	}})
}

func TestSetWebResponseSuccess(t *testing.T) {
	// Test that the return value of txn.SetWebResponse delegates to the
	// input writer.
	app := testApp(nil, ConfigDistributedTracerEnabled(false), t)
	txn := app.StartTransaction("hello")
	w := httptest.NewRecorder()
	rw := txn.SetWebResponse(w)
	rw.WriteHeader(123)
	hdr := rw.Header()
	hdr.Set("zip", "zap")
	body := "should not panic"
	n, err := rw.Write([]byte(body))
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
		AgentAttributes: map[string]interface{}{
			"httpResponseCode": 123,
			"http.statusCode":  123,
		},
		Intrinsics: map[string]interface{}{"name": "OtherTransaction/Go/hello"},
	}})
}

type writerWithFlush struct{}

func (w writerWithFlush) Header() http.Header       { return nil }
func (w writerWithFlush) WriteHeader(int)           {}
func (w writerWithFlush) Write([]byte) (int, error) { return 0, nil }
func (w writerWithFlush) Flush()                    {}

func TestSetWebResponseTxnUpgraded(t *testing.T) {
	// Test that the writer returned by SetWebResponse has the optional
	// methods of the input writer.
	app := testApp(nil, ConfigDistributedTracerEnabled(false), t)
	txn := app.StartTransaction("hello")
	rw := txn.SetWebResponse(writerWithFlush{})
	if _, ok := rw.(http.Flusher); !ok {
		t.Error("should have Flusher now")
	}
}
