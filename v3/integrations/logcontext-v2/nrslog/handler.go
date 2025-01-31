package nrslog

import (
	"context"
	"log/slog"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
)

// NRHandler is an Slog handler that includes logic to implement New Relic Logs in Context
type NRHandler struct {
	handler slog.Handler
	app     *newrelic.Application
	txn     *newrelic.Transaction
}

// WithTransaction returns a new handler that is configured to capture log data
// and attribute it to a specific transaction.
func (h *NRHandler) WithTransaction(txn *newrelic.Transaction) *NRHandler {
	handler := NRHandler{
		handler: h.handler,
		app:     h.app,
		txn:     txn,
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
func (h *NRHandler) Handle(ctx context.Context, record slog.Record) error {
	nrTxn := h.txn

	ctxTxn := newrelic.FromContext(ctx)
	if ctxTxn != nil {
		nrTxn = ctxTxn
	}

	// if no app or txn, do nothing
	if h.app == nil && nrTxn == nil {
		return h.handler.Handle(ctx, record)
	}

	attrs := map[string]interface{}{}

	record.Attrs(func(attr slog.Attr) bool {
		// ignore empty attributes
		if !attr.Equal(slog.Attr{}) {
			attrs[attr.Key] = attr.Value.Any()
		}
		return true
	})

	// timestamp must be sent to newrelic
	var timestamp int64
	if record.Time.IsZero() {
		timestamp = time.Now().UnixMilli()
	} else {
		timestamp = record.Time.UnixMilli()
	}

	data := newrelic.LogData{
		Severity:   record.Level.String(),
		Timestamp:  timestamp,
		Message:    record.Message,
		Attributes: attrs,
	}

	if nrTxn != nil {
		nrTxn.RecordLog(data)
		enrichRecordTxn(nrTxn, &record)
	} else {
		h.app.RecordLog(data)
		enrichRecord(h.app, &record)
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
