// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package nrb3

import (
	"fmt"
	"log"
	"net/http"
	"os"

	newrelic "github.com/newrelic/go-agent"
	"github.com/openzipkin/zipkin-go"
	reporterhttp "github.com/openzipkin/zipkin-go/reporter/http"
)

func currentTxn() newrelic.Transaction {
	return nil
}

func ExampleNewRoundTripper() {
	// When defining the client, set the Transport to the NewRoundTripper. This
	// will create ExternalSegments and add B3 headers for each request.
	client := &http.Client{
		Transport: NewRoundTripper(nil),
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

// This example demonstrates how to create a Zipkin reporter using the standard
// Zipkin http reporter
// (https://godoc.org/github.com/openzipkin/zipkin-go/reporter/http) to send
// Span data to New Relic.  Follow this example when your application uses
// Zipkin for tracing (instead of the New Relic Go Agent) and you wish to send
// span data to the New Relic backend.  The example assumes you have the
// environment variable NEW_RELIC_API_KEY set to your New Relic Insights Insert
// Key.
func Example_zipkinReporter() {
	// import (
	//    reporterhttp "github.com/openzipkin/zipkin-go/reporter/http"
	// )
	reporter := reporterhttp.NewReporter(
		"https://trace-api.newrelic.com/trace/v1",
		reporterhttp.RequestCallback(func(req *http.Request) {
			req.Header.Add("X-Insert-Key", os.Getenv("NEW_RELIC_API_KEY"))
			req.Header.Add("Data-Format", "zipkin")
			req.Header.Add("Data-Format-Version", "2")
		}),
	)
	defer reporter.Close()

	// use the reporter to create a new tracer
	zipkin.NewTracer(reporter)
}
