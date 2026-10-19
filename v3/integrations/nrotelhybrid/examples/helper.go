package examples

import (
	"context"
	"fmt"
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
		newrelic.ConfigOpenTelemetryEnabled(true),
		newrelic.ConfigFromEnvironment(),
	)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	processor := nrotelhybrid.NewHybridProcessor(app)
	if processor == nil {
		stop()
		return nil, nil, nil, nil, fmt.Errorf("otel tracing not enabled")
	}

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
