package nrotelhybrid

import (
	"context"

	"go.opentelemetry.io/otel/sdk/trace"
)

type nrotelhybridProcessor struct {
}

func (p *nrotelhybridProcessor) OnStart(ctx context.Context, s trace.ReadWriteSpan) {
	// for now pretending like everything is enabled
	// first check for case when span has a remote parent.  In this case, it must create a new transaction

	// check if remote parent
	// should be a valid span context and be marked as remote
	// this begins a transaction
	if s.Parent().IsValid() && s.Parent().IsRemote() {
	}

}

func (p *nrotelhybridProcessor) OnEnd(s trace.ReadOnlySpan) {

}

func (p *nrotelhybridProcessor) Shutdown(ctx context.Context) error {
	return nil
}

func (p *nrotelhybridProcessor) ForceFlush(ctx context.Context) error {
	return nil
}
