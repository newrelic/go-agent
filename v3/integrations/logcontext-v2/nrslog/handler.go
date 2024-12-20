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
	goas    []groupOrAttrs
	goasMap map[string]interface{}
}

// groupOrAttrs is a structure that holds either a group name or a slice of attributes
type groupOrAttrs struct {
	group string      // group name if non-empty
	attrs []slog.Attr // attrs if non-empty
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
		goasMap: make(map[string]interface{}),
	}
}

// WithTransaction returns a new handler that is configured to capture log data
// and attribute it to a specific transaction.
func (h *NRHandler) WithTransaction(txn *newrelic.Transaction) *NRHandler {
	handler := &NRHandler{
		handler: h.handler,
		app:     h.app,
		txn:     txn,
		goasMap: make(map[string]interface{}),
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

	record.Attrs(func(attr slog.Attr) bool {
		// ignore empty attributes
		if !attr.Equal(slog.Attr{}) {
			h.goasMap[attr.Key] = attr.Value.Any()
		}
		return true
	})

	// timestamp must be sent to newrelic
	logTime := record.Time.UnixMilli()
	if record.Time.IsZero() {
		logTime = time.Now().UnixMilli()
	}

	// Add any groups or attributes to the log message
	goas := h.goas
	group := ""
	for _, goa := range goas {
		if goa.group != "" {
			group = goa.group
		} else {
			for _, a := range goa.attrs {
				if group != "" {
					a.Key = group + "." + a.Key
				}
				h.appendAttr(a)
				record.AddAttrs(a)
			}
		}
	}

	// Pass Map[string]interface{} to New Relic here
	data := newrelic.LogData{
		Severity:   record.Level.String(),
		Timestamp:  logTime,
		Message:    record.Message,
		Attributes: h.goasMap,
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

func (h *NRHandler) appendAttr(a slog.Attr) {
	// Resolve the Attr's value before doing anything else.
	a.Value = a.Value.Resolve()
	// Ignore empty Attrs.
	if a.Equal(slog.Attr{}) {
		return
	}
	switch a.Value.Kind() {
	case slog.KindString:
		// Quote string values, to make them easy to parse.
		h.goasMap[a.Key] = a.Value.String()
	case slog.KindTime:
		// Write times in a standard way, without the monotonic time.
		h.goasMap[a.Key] = a.Value.Time().Format(time.RFC3339Nano)
	case slog.KindGroup:
		attrs := a.Value.Group()
		// Ignore empty groups.
		if len(attrs) == 0 {
			return
		}
		groupMap := make(map[string]interface{})
		for _, ga := range attrs {
			groupMap[ga.Key] = ga.Value.Any()
		}
		h.goasMap[a.Key] = groupMap
	default:
		h.goasMap[a.Key] = a.Value.Any()
	}
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

func (h *NRHandler) withGroupOrAttrs(goa groupOrAttrs) *NRHandler {
	// Generate cachedAttributes
	h2 := *h
	h2.goas = make([]groupOrAttrs, len(h.goas)+1)
	copy(h2.goas, h.goas)
	h2.goas[len(h2.goas)-1] = goa
	return &h2
}

// WithAttrs returns a new Handler whose attributes consist of
// both the receiver's attributes and the arguments.
// The Handler owns the slice: it may retain, modify or discard it.
func (h *NRHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	return h.withGroupOrAttrs(groupOrAttrs{attrs: attrs})
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
	fmt.Println("With Group!")
	if name == "" {
		return h
	}
	return h.withGroupOrAttrs(groupOrAttrs{group: name})
}

//WithAttributes which adds to a logger
// SLOG record stores distinct attributes in a map. So we need to get them out of the record
