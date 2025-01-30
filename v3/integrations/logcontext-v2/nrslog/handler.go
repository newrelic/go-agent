package nrslog

import (
	"context"
	"log/slog"

	"github.com/newrelic/go-agent/v3/newrelic"
)

// NRHandler is an Slog handler that includes logic to implement New Relic Logs in Context
type NRHandler struct {
	handler slog.Handler
	w       *LogWriter
	app     *newrelic.Application
	txn     *newrelic.Transaction
}

// addWriter is an internal helper function to append an io.Writer to the NRHandler object
func (h *NRHandler) addWriter(w *LogWriter) {
	h.w = w
}

// WithTransaction returns a new handler that is configured to capture log data
// and attribute it to a specific transaction.
func (h *NRHandler) WithTransaction(txn *newrelic.Transaction) *NRHandler {
	handler := NRHandler{
		handler: h.handler,
		app:     h.app,
		txn:     txn,
	}

	if h.w != nil {
		writer := h.w.WithTransaction(txn)
		handler.addWriter(&writer)
	}

	return &handler
}

// Enabled reports whether the handler handles records at the given level.
// The handler ignores records whose level is lower.
func (h *NRHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	return h.handler.Enabled(ctx, lvl)
}

// Handle handles the Record.
// It will only be called when Enabled returns true.
// The Context argument is as for Enabled.
// It is present solely to provide Handlers access to the context's values.
// Canceling the context should not affect record processing.
// (Among other things, log messages may be necessary to debug a
// cancellation-related problem.)
func (h *NRHandler) Handle(ctx context.Context, record slog.Record) error {
	nrTxn := h.txn
	if txn := newrelic.FromContext(ctx); txn != nil {
		nrTxn = txn
	}

	attrs := map[string]interface{}{}

	record.Attrs(func(attr slog.Attr) bool {
		attrs[attr.Key] = attr.Value.Any()
		return true
	})

	data := newrelic.LogData{
		Severity:   record.Level.String(),
		Timestamp:  record.Time.UnixMilli(),
		Message:    record.Message,
		Attributes: attrs,
	}
	if nrTxn != nil {
		nrTxn.RecordLog(data)
	} else {
		h.app.RecordLog(data)
	}

	return h.handler.Handle(ctx, record)
}

// WithAttrs returns a new Handler whose attributes consist of
// both the receiver's attributes and the arguments.
// The Handler owns the slice: it may retain, modify or discard it.
func (h *NRHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handler := h.handler.WithAttrs(attrs)
	return &NRHandler{
		handler: handler,
		app:     h.app,
		txn:     h.txn,
	}
}

// WithGroup returns a new Handler with the given group appended to
// the receiver's existing groups.
// The keys of all subsequent attributes, whether added by With or in a
// Record, should be qualified by the sequence of group names.
// If the name is empty, WithGroup returns the receiver.
func (h *NRHandler) WithGroup(name string) slog.Handler {
	handler := h.handler.WithGroup(name)
	return &NRHandler{
		handler: handler,
		app:     h.app,
		txn:     h.txn,
	}
}

// WithTransactionFromContext creates a wrapped NRHandler, enabling it to automatically reference New Relic
// transaction from context.
//
// Deprecated: this is a no-op
func WithTransactionFromContext(handler *NRHandler) *NRHandler {
	return handler
}
