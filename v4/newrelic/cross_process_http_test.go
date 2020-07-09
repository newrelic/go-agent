// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/newrelic/go-agent/v3/internal/cat"
)

func TestTxnCrossProcessInitFromHTTPRequest(t *testing.T) {
	txp := &txnCrossProcess{}
	txp.Init(true, false, replyAccountOne)
	if txp.IsInbound() {
		t.Error("inbound CAT enabled even though there was no request")
	}

	txp = &txnCrossProcess{}
	req, err := http.NewRequest("GET", "http://foo.bar/", nil)
	if err != nil {
		t.Fatal(err)
	}
	txp.Init(true, false, replyAccountOne)
	if err := txp.InboundHTTPRequest(req.Header); err != nil {
		t.Errorf("got error while consuming an empty request: %v", err)
	}
	if txp.IsInbound() {
		t.Error("inbound CAT enabled even though there was no metadata in the request")
	}

	txp = &txnCrossProcess{}
	req, err = http.NewRequest("GET", "http://foo.bar/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add(cat.NewRelicIDName, mustObfuscate(`1#1`, "foo"))
	req.Header.Add(cat.NewRelicTxnName, mustObfuscate(`["abcdefgh",false,"12345678","b95be233"]`, "foo"))
	txp.Init(true, false, replyAccountOne)
	if err := txp.InboundHTTPRequest(req.Header); err != nil {
		t.Errorf("got error while consuming an inbound CAT request: %v", err)
	}
	// A second call to InboundHTTPRequest to ensure that it can safely
	// be called multiple times:
	if err := txp.InboundHTTPRequest(req.Header); err != nil {
		t.Errorf("got error while consuming an inbound CAT request: %v", err)
	}
	if !txp.IsInbound() {
		t.Error("inbound CAT disabled even though there was metadata in the request")
	}
	if txp.ClientID != "1#1" {
		t.Errorf("incorrect ClientID: %s", txp.ClientID)
	}
	if txp.ReferringTxnGUID != "abcdefgh" {
		t.Errorf("incorrect ReferringTxnGUID: %s", txp.ReferringTxnGUID)
	}
	if txp.TripID != "12345678" {
		t.Errorf("incorrect TripID: %s", txp.TripID)
	}
	if txp.ReferringPathHash != "b95be233" {
		t.Errorf("incorrect ReferringPathHash: %s", txp.ReferringPathHash)
	}
}

func TestAppDataToHTTPHeader(t *testing.T) {
	header := appDataToHTTPHeader("")
	if len(header) != 0 {
		t.Errorf("unexpected number of header elements: %d", len(header))
	}

	header = appDataToHTTPHeader("foo")
	if len(header) != 1 {
		t.Errorf("unexpected number of header elements: %d", len(header))
	}
	if actual := header.Get(cat.NewRelicAppDataName); actual != "foo" {
		t.Errorf("unexpected header value: %s", actual)
	}
}

func TestHTTPHeaderToAppData(t *testing.T) {
	if appData := httpHeaderToAppData(nil); appData != "" {
		t.Errorf("unexpected app data: %s", appData)
	}

	header := http.Header{}
	if appData := httpHeaderToAppData(header); appData != "" {
		t.Errorf("unexpected app data: %s", appData)
	}

	header.Add("X-Foo", "bar")
	if appData := httpHeaderToAppData(header); appData != "" {
		t.Errorf("unexpected app data: %s", appData)
	}

	header.Add(cat.NewRelicAppDataName, "foo")
	if appData := httpHeaderToAppData(header); appData != "foo" {
		t.Errorf("unexpected app data: %s", appData)
	}
}

func TestHTTPHeaderToMetadata(t *testing.T) {
	if metadata := httpHeaderToMetadata(nil); !reflect.DeepEqual(metadata, crossProcessMetadata{}) {
		t.Errorf("unexpected metadata: %v", metadata)
	}

	header := http.Header{}
	if metadata := httpHeaderToMetadata(header); !reflect.DeepEqual(metadata, crossProcessMetadata{}) {
		t.Errorf("unexpected metadata: %v", metadata)
	}

	header.Add("X-Foo", "bar")
	if metadata := httpHeaderToMetadata(header); !reflect.DeepEqual(metadata, crossProcessMetadata{}) {
		t.Errorf("unexpected metadata: %v", metadata)
	}

	header.Add(cat.NewRelicIDName, "id")
	if metadata := httpHeaderToMetadata(header); !reflect.DeepEqual(metadata, crossProcessMetadata{
		ID: "id",
	}) {
		t.Errorf("unexpected metadata: %v", metadata)
	}

	header.Add(cat.NewRelicTxnName, "txn")
	if metadata := httpHeaderToMetadata(header); !reflect.DeepEqual(metadata, crossProcessMetadata{
		ID:      "id",
		TxnData: "txn",
	}) {
		t.Errorf("unexpected metadata: %v", metadata)
	}

	header.Add(cat.NewRelicSyntheticsName, "synth")
	if metadata := httpHeaderToMetadata(header); !reflect.DeepEqual(metadata, crossProcessMetadata{
		ID:         "id",
		TxnData:    "txn",
		Synthetics: "synth",
	}) {
		t.Errorf("unexpected metadata: %v", metadata)
	}
}

func TestMetadataToHTTPHeader(t *testing.T) {
	metadata := crossProcessMetadata{}

	header := metadataToHTTPHeader(metadata)
	if len(header) != 0 {
		t.Errorf("unexpected number of header elements: %d", len(header))
	}

	metadata.ID = "id"
	header = metadataToHTTPHeader(metadata)
	if len(header) != 1 {
		t.Errorf("unexpected number of header elements: %d", len(header))
	}
	if actual := header.Get(cat.NewRelicIDName); actual != "id" {
		t.Errorf("unexpected header value: %s", actual)
	}

	metadata.TxnData = "txn"
	header = metadataToHTTPHeader(metadata)
	if len(header) != 2 {
		t.Errorf("unexpected number of header elements: %d", len(header))
	}
	if actual := header.Get(cat.NewRelicIDName); actual != "id" {
		t.Errorf("unexpected header value: %s", actual)
	}
	if actual := header.Get(cat.NewRelicTxnName); actual != "txn" {
		t.Errorf("unexpected header value: %s", actual)
	}

	metadata.Synthetics = "synth"
	header = metadataToHTTPHeader(metadata)
	if len(header) != 3 {
		t.Errorf("unexpected number of header elements: %d", len(header))
	}
	if actual := header.Get(cat.NewRelicIDName); actual != "id" {
		t.Errorf("unexpected header value: %s", actual)
	}
	if actual := header.Get(cat.NewRelicTxnName); actual != "txn" {
		t.Errorf("unexpected header value: %s", actual)
	}
	if actual := header.Get(cat.NewRelicSyntheticsName); actual != "synth" {
		t.Errorf("unexpected header value: %s", actual)
	}
}
