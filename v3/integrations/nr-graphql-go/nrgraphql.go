package nrgraphql

import (
	"context"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/gqlerrors"
	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func init() { internal.TrackUsage("integration", "framework", "graphql-go") }

type ext struct{}

var _ graphql.Extension = new(ext)

// NewExtension TODO
func NewExtension() graphql.Extension {
	return &ext{}
}

// Init is used to help you initialize the extension
func (e *ext) Init(ctx context.Context, _ *graphql.Params) context.Context {
	return ctx
}

// Name returns the name of the extension (make sure it's custom)
func (e *ext) Name() string {
	return "New Relic Extension"
}

// ParseDidStart is being called before starting the parse
func (e *ext) ParseDidStart(ctx context.Context) (context.Context, graphql.ParseFinishFunc) {
	var seg *newrelic.Segment
	if txn := newrelic.FromContext(ctx); txn != nil {
		seg = txn.StartSegment("Parse")
	}

	return ctx, func(error) {
		seg.End()
	}
}

// ValidationDidStart is called just before the validation begins
func (e *ext) ValidationDidStart(ctx context.Context) (context.Context, graphql.ValidationFinishFunc) {
	var seg *newrelic.Segment
	if txn := newrelic.FromContext(ctx); txn != nil {
		seg = txn.StartSegment("Validation")
	}

	return ctx, func(errors []gqlerrors.FormattedError) {
		seg.End()
	}
}

// ExecutionDidStart notifies about the start of the execution
func (e *ext) ExecutionDidStart(ctx context.Context) (context.Context, graphql.ExecutionFinishFunc) {
	var seg *newrelic.Segment
	if txn := newrelic.FromContext(ctx); txn != nil {
		seg = txn.StartSegment("Execution")
	}

	return ctx, func(result *graphql.Result) {
		seg.End()
	}
}

// ResolveFieldDidStart notifies about the start of the resolving of a field
func (e *ext) ResolveFieldDidStart(ctx context.Context, i *graphql.ResolveInfo) (context.Context, graphql.ResolveFieldFinishFunc) {
	var seg *newrelic.Segment
	if txn := newrelic.FromContext(ctx); txn != nil {
		seg = txn.StartSegment("Resolve " + i.FieldName)
	}

	return ctx, func(interface{}, error) {
		seg.End()
	}

}

// HasResult returns if the extension wants to add data to the result
func (e *ext) HasResult() bool {
	return false
}

// GetResult returns the data that the extension wants to add to the result
func (e *ext) GetResult(context.Context) interface{} {
	return nil
}
