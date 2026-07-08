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
	"sync"

	"github.com/newrelic/go-agent/v3/newrelic"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type nrotelExporter struct {
	Incoming chan sdktrace.ReadOnlySpan
	shutdown chan bool
}

type nrotelProcessor struct {
	original       sdktrace.SpanProcessor
	app            *newrelic.Application
	mu             sync.Mutex
	transactionMap map[trace.TraceID]*newrelic.Transaction
	segmentMap     map[trace.SpanID]*newrelic.Segment
}

func (p *nrotelProcessor) OnStart(_ context.Context, s sdktrace.ReadWriteSpan) {
	p.mu.Lock()
	defer p.mu.Unlock()

	traceID := s.SpanContext().TraceID()

	// Entry span -> New Relic transaction, keyed by trace ID so descendants can
	// find it.
	if isRoot(s.Parent()) {
		txn := p.app.StartTransaction(s.Name())
		// Record the entry span's OTel ID so links targeting the root span can
		// be resolved to the New Relic GUID at harvest.
		txn.SetRootOTelSpanID(s.SpanContext().SpanID().String())
		p.transactionMap[traceID] = txn
		return
	}

	// Child span -> segment on the trace's transaction.
	txn := p.transactionMap[traceID]
	if txn == nil {
		return
	}
	seg := txn.StartSegment(s.Name())
	// Carry the OTel span ID so links targeting this span resolve to its New
	// Relic GUID at harvest.
	seg.OTelSpanID = s.SpanContext().SpanID().String()
	p.segmentMap[s.SpanContext().SpanID()] = seg
}

func (p *nrotelProcessor) OnEnd(s sdktrace.ReadOnlySpan) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Entry span -> end the transaction.
	if isRoot(s.Parent()) {
		traceID := s.SpanContext().TraceID()
		if txn := p.transactionMap[traceID]; txn != nil {
			delete(p.transactionMap, traceID)
			txn.End()
		}
		return
	}

	// Child span -> end the segment.
	spanID := s.SpanContext().SpanID()
	if seg := p.segmentMap[spanID]; seg != nil {
		delete(p.segmentMap, spanID)
		seg.Links = convertLinks(s.Links())
		seg.End()
	}
}

func convertLinks(links []sdktrace.Link) []newrelic.Link {
	var ret []newrelic.Link
	for _, link := range links {
		nrLink := newrelic.Link{
			LinkedSpanId:  link.SpanContext.SpanID().String(), // this is the otel link we need it to be the New Relic Linked Span ID
			LinkedTraceId: link.SpanContext.TraceID().String(),
		}
		ret = append(ret, nrLink)
	}
	return ret
}

func (p *nrotelProcessor) Shutdown(ctx context.Context) error {
	//return p.original.Shutdown(ctx)
	return nil
}

func (p *nrotelProcessor) ForceFlush(ctx context.Context) error {
	//return p.original.ForceFlush(ctx)
	return nil
}

func HybridTracerProvider(app *newrelic.Application) *sdktrace.TracerProvider {
	exporter, err := stdouttrace.New(stdouttrace.WithWriter(os.Stdout), stdouttrace.WithPrettyPrint())
	if err != nil {
		log.Fatalf("failed to create exporter: %v", err)
	}
	proc := &nrotelProcessor{
		app:            app,
		transactionMap: make(map[trace.TraceID]*newrelic.Transaction),
		segmentMap:     make(map[trace.SpanID]*newrelic.Segment),
	}
	// if hybrid agent is configured to send to new relic
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter), sdktrace.WithSpanProcessor(proc))
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
	return nil
}

// isRoot reports whether a span with the given parent span context is an entry
// span for this service, and therefore maps to a New Relic transaction rather
// than a segment. A span is an entry span when it has no parent, or when its
// parent lives in another process (a remote parent), which makes it the root of
// the trace as far as this service is concerned.
//
// Callers must pass the span's parent (s.Parent()), not the span's own context.
func isRoot(parent trace.SpanContext) bool {
	return !parent.IsValid() || parent.IsRemote()
}
