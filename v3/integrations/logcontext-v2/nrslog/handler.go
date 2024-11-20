package nrslog

import (
	"bytes"
	"context"
	"fmt"
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

// WithTransactionFromContext creates a wrapped NRHandler, enabling it to automatically reference New Relic
//
// Warning: This function is deprecated and will be removed in a future release.
func WithTransactionFromContext(handler *NRHandler) *NRHandler {
	return handler
}

// WrapHandler returns a new handler that is wrapped with New Relic tools to capture
// log data based on your application's logs in context settings.
func WrapHandler(app *newrelic.Application, handler slog.Handler) *NRHandler {
	return &NRHandler{
		handler: handler,
		app:     app,
	}
}

// WithTransaction returns a new handler that is configured to capture log data
// and attribute it to a specific transaction.
func (h *NRHandler) WithTransaction(txn *newrelic.Transaction) *NRHandler {
	handler := &NRHandler{
		handler: h.handler,
		app:     h.app,
		txn:     txn,
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
//
// Handle methods that produce output should observe the following rules:
//   - If r.Time is the zero time, time will not be added to your log print, but a timestamp will be sent to newrelic.
//   - Attr's values should be resolved.
//   - If an Attr's key and value are both the zero value, ignore the Attr.
//     This can be tested with attr.Equal(Attr{}).
//   - If a group's key is empty, inline the group's Attrs.
//   - If a group has no Attrs (even if it has a non-empty key),
//     ignore it.
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
	logTime := record.Time.UnixMilli()
	if record.Time.IsZero() {
		logTime = time.Now().UnixMilli()
	}

	data := newrelic.LogData{
		Severity:   record.Level.String(),
		Timestamp:  logTime,
		Message:    record.Message,
		Attributes: attrs,
	}

	// attempt to get the transaction from the context
	txn := newrelic.FromContext(ctx)
	if txn == nil {
		txn = h.txn
	}

	if txn != nil {
		txn.RecordLog(data)
	} else {
		h.app.RecordLog(data)
	}

	var err error

	// enrich log with newrelic metadata
	// this will always return a valid log message even if an error occurs
	enrichedRecord, enrichErr := enrichLog(record.Message, h.app, txn)
	record.Message = enrichedRecord
	if enrichErr != nil {
		err = fmt.Errorf("failed to enrich logs with New Relic metadata: %v", enrichErr)
	}
	handleErr := h.handler.Handle(ctx, record)
	if handleErr != nil {
		if err != nil {
			err = fmt.Errorf("%w; %w", err, handleErr)
		} else {
			err = handleErr
		}
	}

	return err
}

// enrich log always returns a valid log message even if an error occurs
func enrichLog(record string, app *newrelic.Application, txn *newrelic.Transaction) (string, error) {
	var buf *bytes.Buffer
	var err error

	if txn != nil {
		buf = bytes.NewBuffer([]byte(record))
		err = newrelic.EnrichLog(buf, newrelic.FromTxn(txn))
	} else if app != nil {
		buf = bytes.NewBuffer([]byte(record))
		err = newrelic.EnrichLog(buf, newrelic.FromApp(app))
	} else {
		return record, nil
	}

	return buf.String(), err
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
func (h *NRHandler) WithGroup(name string) slog.Handler {
	handler := h.handler.WithGroup(name)
	return &NRHandler{
		handler: handler,
		app:     h.app,
		txn:     h.txn,
	}
}
