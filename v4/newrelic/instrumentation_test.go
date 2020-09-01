// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"net/http"
	"testing"

	"github.com/newrelic/go-agent/v4/internal"
)

func TestNewRoundTripper(t *testing.T) {
	client := http.Client{
		Transport: NewRoundTripper(roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				Request:    req,
				StatusCode: 418,
			}, nil
		})),
	}

	app := newTestApp(t)
	txn := app.StartTransaction("transaction")
	req, err := http.NewRequest("POST", "http://example.com?hello=world", nil)
	if err != nil {
		t.Fatal(err)
	}
	req = RequestWithTransactionContext(req, txn)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	txn.End()

	expTp := "00-00000000000000020000000000000000-0000000000000003-00"
	if actTp := resp.Request.Header.Get("traceparent"); actTp != expTp {
		t.Errorf("Incorrect traceparent header found:\n\texpect=%s actual=%s",
			expTp, actTp)
	}

	app.ExpectSpanEvents(t, []internal.WantSpan{
		{
			Name:       "http POST example.com",
			SpanID:     "0000000000000003",
			TraceID:    "00000000000000020000000000000000",
			ParentID:   "0000000000000002",
			StatusCode: 3,
			Attributes: map[string]interface{}{
				"http.component":   "http",
				"http.flavor":      "1.1",
				"http.host":        "example.com",
				"http.method":      "POST",
				"http.scheme":      "http",
				"http.status_code": int64(418),
				"http.status_text": "I'm a teapot",
				"http.url":         "http://example.com",
			},
		},
		{
			Name:       "transaction",
			SpanID:     "0000000000000002",
			TraceID:    "00000000000000020000000000000000",
			ParentID:   "0000000000000000",
			Attributes: map[string]interface{}{},
		},
	})
}
