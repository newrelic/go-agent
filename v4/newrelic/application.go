// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/newrelic/go-agent/v4/internal/logger"
)

// Application represents your application.  All methods on Application are nil
// safe.  Therefore, a nil Application pointer can be safely used as a mock.
type Application struct {
	tracer      trace.Tracer
	propagators propagation.TextMapPropagator
	logger      Logger
}

// StartTransaction begins a Transaction with the given name.
func (app *Application) StartTransaction(name string) *Transaction {
	if app == nil {
		return nil
	}
	if app.tracer == nil {
		app.logDebug(
			"trying to start a transaction, but the OpenTelemetry.Tracer is not set in the config; aborting",
			nil,
		)

		return nil
	}
	ctx, sp := app.tracer.Start(context.Background(), name,
		trace.WithSpanKind(trace.SpanKindInternal))
	s := &span{
		Span: sp,
		ctx:  ctx,
	}
	return &Transaction{
		rootSpan: s,
		thread: &thread{
			currentSpan: s,
			logger:      app.logger,
		},
		app:  app,
		name: name,
	}
}

// RecordCustomEvent adds a custom event.
//
// eventType must consist of alphanumeric characters, underscores, and
// colons, and must contain fewer than 255 bytes.
//
// Each value in the params map must be a number, string, or boolean.
// Keys must be less than 255 bytes.  The params map may not contain
// more than 64 attributes.  For more information, and a set of
// restricted keywords, see:
// https://docs.newrelic.com/docs/insights/new-relic-insights/adding-querying-data/inserting-custom-events-new-relic-apm-agents
//
// An error is logged if eventType or params is invalid.
func (app *Application) RecordCustomEvent(eventType string, params map[string]interface{}) {
	if app != nil {
		app.logDebug(unimplementedMessage("Application.RecordCustomEvent"), nil)
	}
}

// RecordCustomMetric records a custom metric.  The metric name you
// provide will be prefixed by "Custom/".  Custom metrics are not
// currently supported in serverless mode.
//
// See
// https://docs.newrelic.com/docs/agents/manage-apm-agents/agent-data/collect-custom-metrics
// for more information on custom events.
func (app *Application) RecordCustomMetric(name string, value float64) {
	if app != nil {
		app.logDebug(unimplementedMessage("Application.RecordCustomMetric"), nil)
	}
}

// WaitForConnection blocks until the application is connected, is
// incapable of being connected, or the timeout has been reached.  This
// method is useful for short-lived processes since the application will
// not gather data until it is connected.  nil is returned if the
// application is connected successfully.
//
// If Infinite Tracing is enabled, WaitForConnection will block until a
// connection to the Trace Observer is made, a fatal error is reached, or the
// timeout is hit.
func (app *Application) WaitForConnection(timeout time.Duration) error {
	if app != nil {
		app.logDebug("WaitForConnection is a no-op for this New Relic agent and can be removed", nil)
	}
	return nil
}

// Shutdown flushes data to New Relic's servers and stops all
// agent-related goroutines managing this application.  After Shutdown
// is called, the Application is disabled and will never collect data
// again.  This method blocks until all final data is sent to New Relic
// or the timeout has elapsed.  Increase the timeout and check debug
// logs if you aren't seeing data.
//
// If Infinite Tracing is enabled, Shutdown will block until all queued span
// events have been sent to the Trace Observer or the timeout has been reached.
func (app *Application) Shutdown(timeout time.Duration) {
	if app != nil {
		app.logDebug(unimplementedMessage("Application.Shutdown"), nil)
	}
}

// NewApplication creates an Application and spawns goroutines to manage the
// aggregation and harvesting of data.  On success, a non-nil Application and a
// nil error are returned. On failure, a nil Application and a non-nil error
// are returned. All methods on an Application are nil safe. Therefore, a nil
// Application pointer can be safely used.  Applications do not share global
// state, therefore it is safe to create multiple applications.
//
// The ConfigOption arguments allow for configuration of the Application.  They
// are applied in order from first to last, i.e. latter ConfigOptions may
// overwrite the Config fields already set.
func NewApplication(opts ...ConfigOption) (*Application, error) {
	c := defaultConfig()
	for _, fn := range opts {
		if nil != fn {
			fn(&c)
			if nil != c.Error {
				return nil, c.Error
			}
		}
	}
	tracer := c.OpenTelemetry.Tracer
	if nil == tracer {
		tracer = otel.Tracer("traceName")
	}
	propagators := c.OpenTelemetry.Propagators
	if nil == propagators {
		otel.SetTextMapPropagator(propagation.TraceContext{})
		// propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{})
		// otel.SetTextMapPropagator(propagator)
		// propagators = otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}))
		propagators = otel.GetTextMapPropagator()
	}

	if c.Logger == nil {
		c.Logger = logger.ShimLogger{}
	}
	msg := fmt.Sprintf("Starting New Relic Shim Agent. If everything is configured properly, you should be able"+
		" to see your data in a few minutes by visiting https://one.newrelic.com/launcher/nr1-core.explorer and"+
		" searching for '%s'.", c.AppName)
	c.Logger.Info(msg, nil)

	return &Application{
		tracer:      tracer,
		propagators: propagators,
		logger:      c.Logger,
	}, nil
}

func (app *Application) logInfo(msg string, context map[string]interface{}) {
	if app != nil {
		app.initLogger()
		app.logger.Info(msg, context)
	}
}

func (app *Application) logDebug(msg string, context map[string]interface{}) {
	if app != nil {
		app.initLogger()
		app.logger.Debug(msg, context)
	}
}

func (app *Application) initLogger() {
	if app.logger == nil {
		app.logger = logger.ShimLogger{}
	}
}

func unimplementedMessage(methodName string) string {
	return methodName + " is currently a no-op for this New Relic agent (it may be implemented in a future version)"
}
