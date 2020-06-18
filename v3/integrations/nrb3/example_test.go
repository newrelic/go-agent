// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrb3_test

import (
	"fmt"
	"log"
	"net/http"

	"github.com/newrelic/go-agent/v3/integrations/nrb3"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func currentTxn() *newrelic.Transaction {
	return nil
}

func ExampleNewRoundTripper() {
	// When defining the client, set the Transport to the NewRoundTripper. This
	// will create ExternalSegments and add B3 headers for each request.
	client := &http.Client{
		Transport: nrb3.NewRoundTripper(nil),
	}

	// Distributed Tracing must be enabled for this application.
	txn := currentTxn()

	req, err := http.NewRequest("GET", "http://example.com", nil)
	if nil != err {
		log.Fatalln(err)
	}

	// Be sure to add the transaction to the request context.  This step is
	// required.
	req = newrelic.RequestWithTransactionContext(req, txn)
	resp, err := client.Do(req)
	if nil != err {
		log.Fatalln(err)
	}

	defer resp.Body.Close()
	fmt.Println(resp.StatusCode)
}
