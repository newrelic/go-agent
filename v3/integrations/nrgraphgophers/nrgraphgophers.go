// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrgraphgophers instruments https://github.com/graph-gophers/graphql-go
// applications.
//
// This package creates a graphql-go Tracer that adds adds segment
// instrumentation to your graphql request transactions.
package nrgraphgophers

import (
	"context"
	"sync"

	"github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/introspection"
	"github.com/graph-gophers/graphql-go/trace"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func init() { internal.TrackUsage("integration", "framework", "graph-gophers") }

type requestIDContextKeyType struct{}

var (
	requestIDContextKey requestIDContextKeyType = struct{}{}
)

type requestID uint64

type tracer struct {
	sync.Mutex
	counter      requestID
	activeFields map[requestID]int
}

// NewTracer creates a new trace.Tracer that adds segment instrumentation
// to the transaction.
func NewTracer() trace.Tracer {
	return &tracer{
		activeFields: make(map[requestID]int),
	}
}

func (t *tracer) newRequestID() requestID {
	t.Lock()
	defer t.Unlock()

	id := t.counter
	t.counter++
	return id
}

func (t *tracer) removeFields(id requestID) {
	t.Lock()
	defer t.Unlock()

	delete(t.activeFields, id)
}

func (t *tracer) startField(id requestID) (async bool) {
	t.Lock()
	defer t.Unlock()

	numActive := t.activeFields[id]
	t.activeFields[id] = numActive + 1
	return numActive > 0
}

func (t *tracer) stopField(id requestID) {
	t.Lock()
	defer t.Unlock()

	t.activeFields[id] = t.activeFields[id] - 1
}

func (t *tracer) TraceQuery(ctx context.Context, queryString string, operationName string, variables map[string]interface{}, varTypes map[string]*introspection.Type) (context.Context, trace.TraceQueryFinishFunc) {
	txn := newrelic.FromContext(ctx)
	if nil == txn {
		return ctx, func([]*errors.QueryError) {}
	}

	// Since this https://github.com/graph-gophers/graphql-go/pull/374 was
	// merged in Feb 2020, an empty operation name should be impossible.
	// This conditional is left here in case someone is using an early
	// graphql-go version.
	if operationName == "" {
		operationName = "unknown operation"
	}
	segment := txn.StartSegment(operationName)

	id := t.newRequestID()
	ctx = context.WithValue(ctx, requestIDContextKey, id)

	return ctx, func(errs []*errors.QueryError) {
		t.removeFields(id)
		for _, err := range errs {
			txn.NoticeError(err)
		}
		segment.End()
	}
}

func (t *tracer) TraceField(ctx context.Context, label, typeName, fieldName string, trivial bool, args map[string]interface{}) (context.Context, trace.TraceFieldFinishFunc) {
	txn := newrelic.FromContext(ctx)
	if nil == txn {
		return ctx, func(*errors.QueryError) {}
	}
	id, ok := ctx.Value(requestIDContextKey).(requestID)
	if !ok {
		return ctx, func(*errors.QueryError) {}
	}

	async := t.startField(id)
	if async {
		txn = txn.NewGoroutine()
		// Update the context with the async transaction in case it is
		// possible to make segments inside the field handling code.
		ctx = newrelic.NewContext(ctx, txn)
	}

	segment := txn.StartSegment(fieldName)

	return ctx, func(*errors.QueryError) {
		// Notice errors in query finish function to avoid double
		// noticing errors.
		t.stopField(id)
		segment.End()
	}
}
