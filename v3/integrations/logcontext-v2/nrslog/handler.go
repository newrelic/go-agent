package nrslog

import (
	"bytes"
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
// The Context argument is as for Enabled.
// It is present solely to provide Handlers access to the context's values.
// Canceling the context should not affect record processing.
// (Among other things, log messages may be necessary to debug a
// cancellation-related problem.)
func (h *NRHandler) Handle(ctx context.Context, record slog.Record) error {
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

	nrTxn := h.txn

	ctxTxn := newrelic.FromContext(ctx)
	if ctxTxn != nil {
		nrTxn = ctxTxn
	}

	var enricherOpts newrelic.EnricherOption
	if nrTxn != nil {
		nrTxn.RecordLog(data)
		enricherOpts = newrelic.FromTxn(nrTxn)
	} else {
		h.app.RecordLog(data)
		enricherOpts = newrelic.FromApp(h.app)
	}

	// add linking metadata as an attribute
	// without disrupting normal usage of the handler
	nrLinking := bytes.NewBuffer([]byte{})
	err := newrelic.EnrichLog(nrLinking, enricherOpts)
	if err == nil {
		record.AddAttrs(slog.String("newrelic", nrLinking.String()))
	}

	err = h.handler.Handle(ctx, record)
	return err
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
