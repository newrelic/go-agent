package nrslog

import (
	"context"
	"io"
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

// TextHandler creates a wrapped Slog TextHandler, enabling it to both automatically capture logs
// and to enrich logs locally depending on your logs in context configuration in your New Relic
// application.
func TextHandler(app *newrelic.Application, w io.Writer, opts *slog.HandlerOptions) NRHandler {
	nrWriter := NewWriter(w, app)
	textHandler := slog.NewTextHandler(nrWriter, opts)
	wrappedHandler := WrapHandler(app, textHandler)
	wrappedHandler.addWriter(&nrWriter)
	return wrappedHandler
}

// JSONHandler creates a wrapped Slog JSONHandler, enabling it to both automatically capture logs
// and to enrich logs locally depending on your logs in context configuration in your New Relic
// application.
func JSONHandler(app *newrelic.Application, w io.Writer, opts *slog.HandlerOptions) NRHandler {
	nrWriter := NewWriter(w, app)
	jsonHandler := slog.NewJSONHandler(nrWriter, opts)
	wrappedHandler := WrapHandler(app, jsonHandler)
	wrappedHandler.addWriter(&nrWriter)
	return wrappedHandler
}

// WithTransaction creates a new Slog Logger object to be used for logging within a given transaction.
// Calling this function with a logger having underlying TransactionFromContextHandler handler is a no-op.
func WithTransaction(txn *newrelic.Transaction, logger *slog.Logger) *slog.Logger {
	if txn == nil || logger == nil {
		return logger
	}

	h := logger.Handler()
	switch nrHandler := h.(type) {
	case NRHandler:
		txnHandler := nrHandler.WithTransaction(txn)
		return slog.New(txnHandler)
	default:
		return logger
	}
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
func WrapHandler(app *newrelic.Application, handler slog.Handler) NRHandler {
	return NRHandler{
		handler: handler,
		app:     app,
	}
}

// addWriter is an internal helper function to append an io.Writer to the NRHandler object
func (h *NRHandler) addWriter(w *LogWriter) {
	h.w = w
}

// WithTransaction returns a new handler that is configured to capture log data
// and attribute it to a specific transaction.
func (h *NRHandler) WithTransaction(txn *newrelic.Transaction) NRHandler {
	handler := NRHandler{
		handler: h.handler,
		app:     h.app,
		txn:     txn,
	}

	if h.w != nil {
		writer := h.w.WithTransaction(txn)
		handler.addWriter(&writer)
	}

	return handler
}

// Enabled reports whether the handler handles records at the given level.
// The handler ignores records whose level is lower.
// It is called early, before any arguments are processed,
// to save effort if the log event should be discarded.
// If called from a Logger method, the first argument is the context
// passed to that method, or context.Background() if nil was passed
// or the method does not take a context.
// The context is passed so Enabled can use its values
// to make a decision.
func (h NRHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	return h.handler.Enabled(ctx, lvl)
}

// Handle handles the Record.
// It will only be called when Enabled returns true.
// The Context argument is as for Enabled.
// It is present solely to provide Handlers access to the context's values.
// Canceling the context should not affect record processing.
// (Among other things, log messages may be necessary to debug a
// cancellation-related problem.)
//
// Handle methods that produce output should observe the following rules:
//   - If r.Time is the zero time, ignore the time.
//   - If r.PC is zero, ignore it.
//   - Attr's values should be resolved.
//   - If an Attr's key and value are both the zero value, ignore the Attr.
//     This can be tested with attr.Equal(Attr{}).
//   - If a group's key is empty, inline the group's Attrs.
//   - If a group has no Attrs (even if it has a non-empty key),
//     ignore it.
func (h NRHandler) Handle(ctx context.Context, record slog.Record) error {
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
	if h.txn != nil {
		h.txn.RecordLog(data)
	} else {
		h.app.RecordLog(data)
	}

	return h.handler.Handle(ctx, record)
}

// WithAttrs returns a new Handler whose attributes consist of
// both the receiver's attributes and the arguments.
// The Handler owns the slice: it may retain, modify or discard it.
func (h NRHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handler := h.handler.WithAttrs(attrs)
	return NRHandler{
		handler: handler,
		app:     h.app,
		txn:     h.txn,
	}

}

// WithGroup returns a new Handler with the given group appended to
// the receiver's existing groups.
// The keys of all subsequent attributes, whether added by With or in a
// Record, should be qualified by the sequence of group names.
//
// How this qualification happens is up to the Handler, so long as
// this Handler's attribute keys differ from those of another Handler
// with a different sequence of group names.
//
// A Handler should treat WithGroup as starting a Group of Attrs that ends
// at the end of the log event. That is,
//
//	logger.WithGroup("s").LogAttrs(level, msg, slog.Int("a", 1), slog.Int("b", 2))
//
// should behave like
//
//	logger.LogAttrs(level, msg, slog.Group("s", slog.Int("a", 1), slog.Int("b", 2)))
//
// If the name is empty, WithGroup returns the receiver.
func (h NRHandler) WithGroup(name string) slog.Handler {
	handler := h.handler.WithGroup(name)
	return NRHandler{
		handler: handler,
		app:     h.app,
		txn:     h.txn,
	}
}

// NRHandler is an Slog handler that includes logic to implement New Relic Logs in Context.
// New Relic transaction value is taken from context. It cannot be set directly.
// This serves as a quality of life improvement for cases where slog.Default global instance is
// referenced, allowing to use slog methods directly and maintaining New Relic instrumentation.
type TransactionFromContextHandler struct {
	NRHandler
}

// WithTransactionFromContext creates a wrapped NRHandler, enabling it to automatically reference New Relic
// transaction from context.
func WithTransactionFromContext(handler NRHandler) TransactionFromContextHandler {
	return TransactionFromContextHandler{handler}
}

// Handle handles the Record.
// It will only be called when Enabled returns true.
// The Context argument is as for Enabled and NewRelic transaction.
// Canceling the context should not affect record processing.
// (Among other things, log messages may be necessary to debug a
// cancellation-related problem.)
//
// Handle methods that produce output should observe the following rules:
//   - If r.Time is the zero time, ignore the time.
//   - If r.PC is zero, ignore it.
//   - Attr's values should be resolved.
//   - If an Attr's key and value are both the zero value, ignore the Attr.
//     This can be tested with attr.Equal(Attr{}).
//   - If a group's key is empty, inline the group's Attrs.
//   - If a group has no Attrs (even if it has a non-empty key),
//     ignore it.
func (h TransactionFromContextHandler) Handle(ctx context.Context, record slog.Record) error {
	if txn := newrelic.FromContext(ctx); txn != nil {
		return h.NRHandler.WithTransaction(txn).Handle(ctx, record)
	}

	return h.NRHandler.Handle(ctx, record)
}
