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
func TextHandler(app *newrelic.Application, w io.Writer, opts *slog.HandlerOptions) slog.Handler {
	return WrapHandler(app, slog.NewTextHandler(w, opts))
}

// JSONHandler creates a wrapped Slog JSONHandler, enabling it to both automatically capture logs
// and to enrich logs locally depending on your logs in context configuration in your New Relic
// application.
//
// Deprecated: Use WrapHandler() instead.
func JSONHandler(app *newrelic.Application, w io.Writer, opts *slog.HandlerOptions) slog.Handler {
	return WrapHandler(app, slog.NewJSONHandler(w, opts))
}

// WithTransaction creates a new Slog Logger object to be used for logging within a given
// transaction it its found in a context. Creating a transaction logger can have a performance
// benefit when transactions are long running, and have a high log volume in comparison to
// reading transactions from context on every log message.
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
// benefit when transactions are long running, and have a high log volume in comparison to
// reading transactions from context on every log message.
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
