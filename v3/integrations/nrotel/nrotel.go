// Copyright 2025 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrotel is a placeholder for the OpenTelemetry <-> New Relic bridge.
//
// See the example directory for a self-contained program that instruments a Go
// application with the upstream OpenTelemetry SDK and exports traces to New
// Relic over OTLP.
package nrotel

import (
	"context"
	"log"
	"os"

	"github.com/newrelic/go-agent/v3/newrelic"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type nrotelExporter struct {
	Incoming chan sdktrace.ReadOnlySpan
	shutdown chan bool
}
type nrSegment interface {
	End()
}

func HybridTracerProvider() *sdktrace.TracerProvider {
	exporter, err := stdouttrace.New(stdouttrace.WithWriter(os.Stdout), stdouttrace.WithPrettyPrint())
	if err != nil {
		log.Fatalf("failed to create exporter: %v", err)
	}
	expo := &nrotelExporter{}
	// if hybrid agent is configured to send to new relic
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter), sdktrace.WithBatcher(expo))
	return tp
}

func (exp *nrotelExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	for _, s := range spans {
		//exp.Incoming <- s
		err := processSpan(s)
		if err != nil {
			return err
		}
	}
	return nil
}

func (exp *nrotelExporter) Shutdown(ctx context.Context) error {
	//exp.shutdown <- true
	return nil
}

func processSpan(span sdktrace.ReadOnlySpan) error {
	if isRoot(span.Parent()) {
		// turn into transaction with a root span
		return nil
	}
	// otherwise it is a segment so what type of segment is it?
	if err := populateSegments(span.Attributes()); err != nil {
		return err
	}

	return nil
}

func isRoot(parent trace.SpanContext) bool {
	// probably need to change this logic.  What makes it a root?
	//parent.IsValid()?
	if parent.HasSpanID() && parent.HasTraceID() {
		return false
	}
	return true
}

func populateSegments(attrs []attribute.KeyValue) error {
	for _, attr := range attrs {

	}
}

func segmentType(attr attribute.KeyValue) nrSegment {
	// http - spans: http.request.method, server.address, server.port, url.full -> ExternalSegment
	// db - spans: db.system.name -> DatastoreSegment
	// messaging: messaging.operation.name, messaging.system -> MessageProducerSegment
	// everything else is a general segment -> Segment
	switch attr.Key {
	case "http.request.method":
		return &newrelic.ExternalSegment{}
	case "server.address":
		return &newrelic.ExternalSegment{}
	case "server.port":
		return &newrelic.ExternalSegment{}
	case "url.full":
		return &newrelic.ExternalSegment{}
	case "db.system.name":
		return &newrelic.DatastoreSegment{}
	case "messaging.operation.name":
		return &newrelic.MessageProducerSegment{}
	case "messaging.system":
		return &newrelic.MessageProducerSegment{}
	}
	return &newrelic.Segment{}
}
