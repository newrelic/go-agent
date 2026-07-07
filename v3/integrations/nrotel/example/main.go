// Copyright 2025 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// This example instruments a Go application with the upstream OpenTelemetry SDK
// and prints the resulting spans to stdout as JSON. It uses only the
// `go.opentelemetry.io/otel` packages and needs no New Relic account or network.
//
// The comments map each New Relic concept to its OpenTelemetry equivalent:
//
//	New Relic Go agent     OpenTelemetry
//	------------------     -------------
//	Transaction            Root span (a span with no in-process parent)
//	Segment                Child span
//	txn.AddAttribute(...)  span.SetAttributes(...)
//
// Run it with:
//
//	go run ./example
package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/newrelic/go-agent/v3/integrations/nrotel"
	"github.com/newrelic/go-agent/v3/newrelic"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func main() {
	// Export spans to stdout as indented JSON.

	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Hybrid Example"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDistributedTracerEnabled(true),
		newrelic.ConfigDebugLogger(os.Stdout),
	)
	if err != nil {
		log.Fatal(err)
	}
	err = app.WaitForConnection(10 * time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer app.Shutdown(10 * time.Second)

	tp := nrotel.HybridTracerProvider(app)
	defer tp.Shutdown(context.Background())
	otel.SetTracerProvider(tp)
	handleCheckout(context.Background(), "order-12345")
}

// handleCheckout models a New Relic transaction. The span started here is a
// root span: nothing started a span before it, so it sits at the top of the trace.
func handleCheckout(ctx context.Context, orderID string) {
	tracer := otel.Tracer("nrotel-example")

	// Start returns a new context carrying this span. Passing it down is how
	// child spans (segments) find their parent.
	ctx, txn := tracer.Start(ctx, "Checkout")
	defer txn.End()

	// Custom attributes on the root span == txn.AddAttribute(...).
	txn.SetAttributes(
		attribute.String("order.id", orderID),
		attribute.String("customer.tier", "premium"),
	)

	chargePayment(ctx)
	saveOrder(ctx)
}

// chargePayment models a New Relic segment: a child span timing work inside the
// transaction. Using the context from the root span auto-parents it to "Checkout".
func chargePayment(ctx context.Context) {
	_, seg := otel.Tracer("nrotel-example").Start(ctx, "ChargePayment")
	defer seg.End()
	link := trace.Link{
		SpanContext: seg.SpanContext(),
	}
	link.Attributes = append(link.Attributes, attribute.KeyValue{
		Key:   "SpanLinkTest1",
		Value: attribute.Value{},
	})
	seg.AddLink(link)

	seg.SetAttributes(attribute.Float64("payment.amount", 49.99))
	time.Sleep(15 * time.Millisecond) // pretend to call a payment gateway
}

// saveOrder is a second segment, a sibling of ChargePayment under "Checkout".
func saveOrder(ctx context.Context) {
	_, seg := otel.Tracer("nrotel-example").Start(ctx, "SaveOrder")
	defer seg.End()
	link := trace.Link{
		SpanContext: seg.SpanContext(),
	}
	link.Attributes = append(link.Attributes, attribute.KeyValue{
		Key:   "SpanLinkTest2",
		Value: attribute.Value{},
	})
	seg.AddLink(link)
	seg.SetAttributes(attribute.String("db.operation", "INSERT"))
	time.Sleep(10 * time.Millisecond) // pretend to write to a database
}
