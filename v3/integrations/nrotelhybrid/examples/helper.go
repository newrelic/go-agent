package examples

import (
	"context"
	"os"
	"os/signal"

	"github.com/newrelic/go-agent/v3/integrations/nrotelhybrid"
	"github.com/newrelic/go-agent/v3/newrelic"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"
)

func NewHybridApp(name string) (*newrelic.Application, context.Context, func(), func(), error) {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName(name),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDistributedTracerEnabled(true),
		newrelic.ConfigDebugLogger(os.Stdout),
		newrelic.ConfigFromEnvironment(),
	)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	processor := nrotelhybrid.NewHybridProcessor(app)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)

	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		stop()
		return nil, nil, nil, nil, err
	}

	tp := trace.NewTracerProvider(trace.WithSyncer(exporter), trace.WithSpanProcessor(processor))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return app, ctx, stop, func() {
		tp.Shutdown(context.Background())
	}, nil
}
