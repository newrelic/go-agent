// Package nrb3 supports adding B3 headers to outgoing requests.
//
// When using the New Relic Go Agent, use this package if you want to add B3
// headers ("X-B3-TraceId", etc., see
// https://github.com/openzipkin/b3-propagation) to outgoing requests.
//
// Distributed tracing must be enabled
// (https://docs.newrelic.com/docs/understand-dependencies/distributed-tracing/enable-configure/enable-distributed-tracing)
// for B3 headers to be added properly.
//
// This example demonstrates how to create a Zipkin reporter using the standard
// Zipkin http reporter
// (https://godoc.org/github.com/openzipkin/zipkin-go/reporter/http) to send
// Span data to New Relic.  Follow this example when your application uses
// Zipkin for tracing (instead of the New Relic Go Agent) and you wish to send
// span data to the New Relic backend.  The example assumes you have the
// environment variable NEW_RELIC_API_KEY set to your New Relic Insights Insert
// Key.
//
//	import (
//		zipkin "github.com/openzipkin/zipkin-go"
//		reporterhttp "github.com/openzipkin/zipkin-go/reporter/http"
//	)
//
//	func main() {
//		reporter := reporterhttp.NewReporter(
//			"https://trace-api.newrelic.com/trace/v1",
//			reporterhttp.RequestCallback(func(req *http.Request) {
//				req.Header.Add("X-Insert-Key", os.Getenv("NEW_RELIC_API_KEY"))
//				req.Header.Add("Data-Format", "zipkin")
//				req.Header.Add("Data-Format-Version", "2")
//			}),
//		)
//		defer reporter.Close()
//
//		// use the reporter to create a new tracer
//		zipkin.NewTracer(reporter)
//	}
package nrb3
