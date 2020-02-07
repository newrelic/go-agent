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
func (Extension) Init(ctx context.Context, _ *graphql.Params) context.Context {
	return ctx
}

// Name returns the name of the extension (make sure it's custom)
func (Extension) Name() string {
	return "New Relic Extension"
}

// ParseDidStart is being called before starting the parse
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

// ValidationDidStart is called just before the validation begins
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

// ExecutionDidStart notifies about the start of the execution
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

// ResolveFieldDidStart notifies about the start of the resolving of a field
func (Extension) ResolveFieldDidStart(ctx context.Context, i *graphql.ResolveInfo) (context.Context, graphql.ResolveFieldFinishFunc) {
	seg := newrelic.FromContext(ctx).StartSegment("Resolve " + i.FieldName)
	return ctx, func(interface{}, error) {
		seg.End()
	}
}

// HasResult returns if the extension wants to add data to the result
func (Extension) HasResult() bool {
	return false
}

// GetResult returns the data that the extension wants to add to the result
func (Extension) GetResult(context.Context) interface{} {
	return nil
}
