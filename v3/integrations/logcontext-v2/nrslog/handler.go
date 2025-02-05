package nrslog

import (
	"bytes"
	"context"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
)

// NRHandler is an Slog handler that includes logic to implement New Relic Logs in Context
type NRHandler struct {
	handler slog.Handler

	app *newrelic.Application
	txn *newrelic.Transaction

	// group logic
	goas []groupOrAttrs
}

type groupOrAttrs struct {
	group string
	attrs []slog.Attr
}

// WrapHandler returns a new handler that is wrapped with New Relic tools to capture
// log data based on your application's logs in context settings.
//
// Note: if your app is nil, or your handler is already wrapped with a NRHandler, WrapHandler will return
// the handler as is.
//
// TODO: does this need to return an error?
func WrapHandler(app *newrelic.Application, handler slog.Handler) slog.Handler {
	if app == nil {
		return handler
	}
	switch handler.(type) {
	case *NRHandler:
		return handler
	default:
		return &NRHandler{
			handler: handler,
			app:     app,
		}
	}
}

// New Returns a new slog.Logger object wrapped with a New Relic handler that controls
// logs in context features.
func New(app *newrelic.Application, handler slog.Handler) *slog.Logger {
	return slog.New(WrapHandler(app, handler))
}

// clone duplicates the handler, creating a new instance with the same configuration.
// This is a deep copy.
func (h *NRHandler) clone() *NRHandler {
	return &NRHandler{
		handler: h.handler,
		app:     h.app,
		txn:     h.txn,
		goas:    slices.Clone(h.goas),
	}
}

// WithTransaction returns a new handler that is configured to capture log data
// and attribute it to a specific transaction.
func (h *NRHandler) WithTransaction(txn *newrelic.Transaction) *NRHandler {
	h2 := h.clone()
	h2.txn = txn
	return h2
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

	// if no app or txn, invoke underlying handler
	if h.app == nil && nrTxn == nil {
		return h.handler.Handle(ctx, record)
	}

	// timestamp must be sent to newrelic
	var timestamp int64
	if record.Time.IsZero() {
		timestamp = time.Now().UnixMilli()
	} else {
		timestamp = record.Time.UnixMilli()
	}

	goas := h.goas
	if record.NumAttrs() == 0 {
		// If the record has no Attrs, remove groups at the end of the list; they are empty.
		for len(goas) > 0 && goas[len(goas)-1].group != "" {
			goas = goas[:len(goas)-1]
		}
	}

	var data newrelic.LogData

	// TODO: can we cache this?
	if shouldForwardLogs(h.app) {
		// TODO: optimize this to avoid maps, its very expensive
		attrs := map[string]interface{}{}
		groupPrefix := strings.Builder{}

		for _, goa := range goas {
			if goa.group != "" {
				if len(groupPrefix.String()) > 0 {
					groupPrefix.WriteByte('.')
				}
				groupPrefix.WriteString(goa.group)
			} else {
				for _, a := range goa.attrs {
					h.appendAttr(attrs, a, groupPrefix.String())
				}
			}
		}

		record.Attrs(func(attr slog.Attr) bool {
			h.appendAttr(attrs, attr, groupPrefix.String())
			return true
		})

		data = newrelic.LogData{
			Severity:   record.Level.String(),
			Timestamp:  timestamp,
			Message:    record.Message,
			Attributes: attrs,
		}
	}

	if nrTxn != nil {
		if data.Message != "" {
			nrTxn.RecordLog(data)
		}
		enrichRecordTxn(nrTxn, &record)
	} else {
		if data.Message != "" {
			h.app.RecordLog(data)
		}
		enrichRecord(h.app, &record)
	}

	return h.handler.Handle(ctx, record)
}

func (h *NRHandler) appendAttr(nrAttrs map[string]interface{}, a slog.Attr, groupPrefix string) {
	// Resolve the Attr's value before doing anything else.
	a.Value = a.Value.Resolve()
	// Ignore empty Attrs.
	if a.Equal(slog.Attr{}) {
		return
	}

	groupBuffer := bytes.Buffer{}
	groupBuffer.WriteString(groupPrefix)

	if groupBuffer.Len() > 0 {
		groupBuffer.WriteByte('.')
	}
	groupBuffer.WriteString(a.Key)
	key := groupBuffer.String()

	// If the Attr is a group, append its attributes
	if a.Value.Kind() == slog.KindGroup {
		attrs := a.Value.Group()
		// Ignore empty groups.
		if len(attrs) == 0 {
			return
		}

		for _, ga := range attrs {
			h.appendAttr(nrAttrs, ga, key)
		}
		return
	}

	// attr is an attribute
	nrAttrs[key] = a.Value.Any()
}

// WithAttrs returns a new Handler whose attributes consist of
// both the receiver's attributes and the arguments.
//
// This wraps the WithAttrs of the underlying handler, and will not modify the
// attributes slice in any way.
func (h *NRHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	newHandler := h.withGroupOrAttrs(groupOrAttrs{attrs: attrs})
	newHandler.handler = newHandler.handler.WithAttrs(attrs)
	return newHandler
}

// WithGroup returns a new Handler with the given group appended to
// the receiver's existing groups.
// The keys of all subsequent attributes, whether added by With or in a
// Record, should be qualified by the sequence of group names.
// If the name is empty, WithGroup returns the receiver.
func (h *NRHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	newHandler := h.withGroupOrAttrs(groupOrAttrs{group: name})
	newHandler.handler = newHandler.handler.WithGroup(name)
	return newHandler
}

func (h *NRHandler) withGroupOrAttrs(goa groupOrAttrs) *NRHandler {
	h2 := *h
	h2.goas = make([]groupOrAttrs, len(h.goas)+1)
	copy(h2.goas, h.goas)
	h2.goas[len(h2.goas)-1] = goa
	return &h2
}

// WithTransactionFromContext creates a wrapped NRHandler, enabling it to automatically reference New Relic
// transaction from context.
//
// Deprecated: this is a no-op
func WithTransactionFromContext(handler slog.Handler) slog.Handler {
	return handler
}
