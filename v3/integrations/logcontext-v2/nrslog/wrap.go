package nrslog

import (
	"context"
	"io"
	"log/slog"

	"github.com/newrelic/go-agent/v3/newrelic"
)

// TextHandler creates a wrapped Slog TextHandler, enabling it to both automatically capture logs
// and to enrich logs locally depending on your logs in context configuration in your New Relic
// application.
//
// Deprecated: Use WrapHandler() instead.
func TextHandler(app *newrelic.Application, w io.Writer, opts *slog.HandlerOptions) *NRHandler {
	return WrapHandler(app, slog.NewTextHandler(w, opts))
}

// JSONHandler creates a wrapped Slog JSONHandler, enabling it to both automatically capture logs
// and to enrich logs locally depending on your logs in context configuration in your New Relic
// application.
//
// Deprecated: Use WrapHandler() instead.
func JSONHandler(app *newrelic.Application, w io.Writer, opts *slog.HandlerOptions) *NRHandler {
	return WrapHandler(app, slog.NewJSONHandler(w, opts))
}

// WrapHandler returns a new handler that is wrapped with New Relic tools to capture
// log data based on your application's logs in context settings.
func WrapHandler(app *newrelic.Application, handler slog.Handler) *NRHandler {
	return &NRHandler{
		handler: handler,
		app:     app,
	}
}

// New Returns a new slog.Logger object wrapped with a New Relic handler that controls
// logs in context features.
func New(app *newrelic.Application, handler slog.Handler) *slog.Logger {
	return slog.New(WrapHandler(app, handler))
}

// WithTransaction creates a new Slog Logger object to be used for logging within a given
// transaction it its found in a context. Creating a transaction logger can have a performance
// benefit when transactions are long running, and have a high log volume.
//
// Note: transaction contexts can also be passed to the logger without creating a new
// logger using logger.InfoContext() or similar commands.
func WithContext(ctx context.Context, logger *slog.Logger) *slog.Logger {
	if ctx == nil {
		return logger
	}

	txn := newrelic.FromContext(ctx)
	return WithTransaction(txn, logger)
}

// WithTransaction creates a new Slog Logger object to be used for logging
// within a given transaction. Creating a transaction logger can have a performance
// benefit when transactions are long running, and have a high log volume.
//
// Note: transaction contexts can also be passed to the logger without creating a new
// logger using logger.InfoContext() or similar commands.
func WithTransaction(txn *newrelic.Transaction, logger *slog.Logger) *slog.Logger {
	if txn == nil || logger == nil {
		return logger
	}

	h := logger.Handler()
	switch nrHandler := h.(type) {
	case *NRHandler:
		txnHandler := nrHandler.WithTransaction(txn)
		return slog.New(txnHandler)
	default:
		return logger
	}
}
