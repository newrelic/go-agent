package nrgraphql

import (
	"context"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/gqlerrors"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func init() { internal.TrackUsage("integration", "framework", "graphql-go") }

// Extension TODO
type Extension struct{}

var _ graphql.Extension = Extension{}

// Init is used to help you initialize the extension
func (e Extension) Init(ctx context.Context, _ *graphql.Params) context.Context {
	return ctx
}

// Name returns the name of the extension (make sure it's custom)
func (e Extension) Name() string {
	return "New Relic Extension"
}

// ParseDidStart is being called before starting the parse
func (e Extension) ParseDidStart(ctx context.Context) (context.Context, graphql.ParseFinishFunc) {
	var seg *newrelic.Segment
	if txn := newrelic.FromContext(ctx); txn != nil {
		seg = txn.StartSegment("Parse")
	}

	return ctx, func(error) {
		seg.End()
	}
}

// ValidationDidStart is called just before the validation begins
func (e Extension) ValidationDidStart(ctx context.Context) (context.Context, graphql.ValidationFinishFunc) {
	var seg *newrelic.Segment
	if txn := newrelic.FromContext(ctx); txn != nil {
		seg = txn.StartSegment("Validation")
	}

	return ctx, func([]gqlerrors.FormattedError) {
		seg.End()
	}
}

// ExecutionDidStart notifies about the start of the execution
func (e Extension) ExecutionDidStart(ctx context.Context) (context.Context, graphql.ExecutionFinishFunc) {
	var seg *newrelic.Segment
	if txn := newrelic.FromContext(ctx); txn != nil {
		seg = txn.StartSegment("Execution")
	}

	return ctx, func(*graphql.Result) {
		seg.End()
	}
}

// ResolveFieldDidStart notifies about the start of the resolving of a field
func (e Extension) ResolveFieldDidStart(ctx context.Context, i *graphql.ResolveInfo) (context.Context, graphql.ResolveFieldFinishFunc) {
	var seg *newrelic.Segment
	if txn := newrelic.FromContext(ctx); txn != nil {
		seg = txn.StartSegment("Resolve " + i.FieldName)
	}

	return ctx, func(interface{}, error) {
		seg.End()
	}
}

// HasResult returns if the extension wants to add data to the result
func (e Extension) HasResult() bool {
	return false
}

// GetResult returns the data that the extension wants to add to the result
func (e Extension) GetResult(context.Context) interface{} {
	return nil
}
