// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrgraphqlgo instruments https://github.com/graphql-go/graphql
// applications.
//
// This package creates an Extension that adds segment
// instrumentation for each portion of the GraphQL execution
// (Parse, Validation, Execution, ResolveField) to your GraphQL
// request transactions. Errors in any of these steps will
// be noticed using NoticeError
// (https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Transaction.NoticeError)
//
// Please note that you must also instrument your web request handlers
// and put the transaction into the context object in order to
// utilize this instrumentation. For example, you could use
// newrelic.WrapHandle (https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#WrapHandle)
// or newrelic.WrapHandleFunc (https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#WrapHandleFunc)
// or you could use a New Relic integration for the web framework you are using
// if it is available (for example,
// https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrgorilla)
//
// For a complete example, including instrumenting a graphql-go-handler, see:
// https://github.com/newrelic/go-agent/tree/master/v3/integrations/nrgraphqlgo/example/main.go
package nrgraphqlgo

import (
	"context"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/gqlerrors"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func init() { internal.TrackUsage("integration", "framework", "graphql-go") }

// Extension is an extension that creates segments for New Relic, tracking each
// step of the execution process
type Extension struct{}

var _ graphql.Extension = Extension{}

// Init is used to help you initialize the extension - in this case, a noop
func (Extension) Init(ctx context.Context, _ *graphql.Params) context.Context {
	return ctx
}

// Name returns the name of the extension
func (Extension) Name() string {
	return "New Relic Extension"
}

// ParseDidStart is called before parsing starts
func (Extension) ParseDidStart(ctx context.Context) (context.Context, graphql.ParseFinishFunc) {
	txn := newrelic.FromContext(ctx)
	seg := txn.StartSegment("Parse")
	return ctx, func(err error) {
		if err != nil {
			txn.NoticeError(err)
		}
		seg.End()
	}
}

// ValidationDidStart is called before the validation begins
func (Extension) ValidationDidStart(ctx context.Context) (context.Context, graphql.ValidationFinishFunc) {
	txn := newrelic.FromContext(ctx)
	seg := txn.StartSegment("Validation")
	return ctx, func(errs []gqlerrors.FormattedError) {
		for _, err := range errs {
			txn.NoticeError(err)
		}
		seg.End()
	}
}

// ExecutionDidStart is called before the execution begins
func (Extension) ExecutionDidStart(ctx context.Context) (context.Context, graphql.ExecutionFinishFunc) {
	txn := newrelic.FromContext(ctx)
	seg := txn.StartSegment("Execution")
	return ctx, func(res *graphql.Result) {
		// noticing here also captures those during resolve
		for _, err := range res.Errors {
			txn.NoticeError(err)
		}
		seg.End()
	}
}

// ResolveFieldDidStart is called at the start of the resolving of a field
func (Extension) ResolveFieldDidStart(ctx context.Context, i *graphql.ResolveInfo) (context.Context, graphql.ResolveFieldFinishFunc) {
	seg := newrelic.FromContext(ctx).StartSegment("ResolveField:" + i.FieldName)
	return ctx, func(interface{}, error) {
		seg.End()
	}
}

// HasResult returns true if the extension wants to add data to the result - this extension does not.
func (Extension) HasResult() bool {
	return false
}

// GetResult returns the data that the extension wants to add to the result - in this case, none
func (Extension) GetResult(context.Context) interface{} {
	return nil
}
