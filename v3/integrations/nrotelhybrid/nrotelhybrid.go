package nrotelhybrid

import (
	"context"

	"go.opentelemetry.io/otel/sdk/trace"
)

type nrotelhybridProcessor struct {
}

func (p *nrotelhybridProcessor) OnStart(ctx context.Context, s trace.ReadWriteSpan) {

}

func (p *nrotelhybridProcessor) OnEnd(s trace.ReadOnlySpan) {

}

func (p *nrotelhybridProcessor) Shutdown(ctx context.Context) error {

}

func (p *nrotelhybridProcessor) ForceFlush(ctx context.Context) error {

}
