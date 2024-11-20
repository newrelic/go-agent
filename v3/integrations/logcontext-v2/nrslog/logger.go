package nrslog

import (
	"context"
	"io"
	"log/slog"

	"github.com/newrelic/go-agent/v3/newrelic"
)

// TextHandler is a wrapper on slog.NewTextHandler that includes New Relic Logs in Context.
// This method has been preserved for backwards compatibility, but is not longer recommended.
// Deprecated: Use WrapHandler instead.
func TextHandler(app *newrelic.Application, w io.Writer, opts *slog.HandlerOptions) *NRHandler {
	return WrapHandler(app, slog.NewTextHandler(w, opts))
}

// TextHandler is a wrapper on slog.NewTextHandler that includes New Relic Logs in Context.
// This method has been preserved for backwards compatibility, but is not longer recommended.
// Deprecated: Use WrapHandler instead.
func JSONHandler(app *newrelic.Application, w io.Writer, opts *slog.HandlerOptions) *NRHandler {
	return WrapHandler(app, slog.NewJSONHandler(w, opts))
}

// WithTransaction creates a new Slog Logger object to be used for logging within a given transaction.
// If no transaction is found, the original logger will be returned.
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

// WithContext creates a new Slog Logger object if a transaction is found in the context.
// The new logger will exclusively log for a given transaction.
// If no transaction is found, the original logger is returned.
func WithContext(ctx context.Context, logger *slog.Logger) *slog.Logger {
	if ctx == nil {
		return logger
	}

	txn := newrelic.FromContext(ctx)
	return WithTransaction(txn, logger)
}
