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
func TextHandler(app *newrelic.Application, w io.Writer, opts *slog.HandlerOptions) *NRHandler {
	nrWriter := NewWriter(w, app)
	textHandler := slog.NewTextHandler(nrWriter, opts)
	wrappedHandler := WrapHandler(app, textHandler)
	wrappedHandler.addWriter(&nrWriter)
	return wrappedHandler
}

// JSONHandler creates a wrapped Slog JSONHandler, enabling it to both automatically capture logs
// and to enrich logs locally depending on your logs in context configuration in your New Relic
// application.
func JSONHandler(app *newrelic.Application, w io.Writer, opts *slog.HandlerOptions) *NRHandler {
	nrWriter := NewWriter(w, app)
	jsonHandler := slog.NewJSONHandler(nrWriter, opts)
	wrappedHandler := WrapHandler(app, jsonHandler)
	wrappedHandler.addWriter(&nrWriter)
	return wrappedHandler
}

// WithTransaction creates a new Slog Logger object to be used for logging within a given transaction it its found
// in a context.
// Calling this function with a logger having underlying TransactionFromContextHandler handler is a no-op.
func WithContext(ctx context.Context, logger *slog.Logger) *slog.Logger {
	if ctx == nil {
		return logger
	}

	txn := newrelic.FromContext(ctx)
	return WithTransaction(txn, logger)
}

// WrapHandler returns a new handler that is wrapped with New Relic tools to capture
// log data based on your application's logs in context settings.
func WrapHandler(app *newrelic.Application, handler slog.Handler) *NRHandler {
	return &NRHandler{
		handler: handler,
		app:     app,
	}
}

// WithTransaction creates a new Slog Logger object to be used for logging within a given transaction.
// Calling this function with a logger having underlying TransactionFromContextHandler handler is a no-op.
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
