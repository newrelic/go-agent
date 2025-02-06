package nrslog

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
)

// NRHandler is an Slog handler that includes logic to implement New Relic Logs in Context
type NRHandler struct {
	configCache
	attributeCache

	handler slog.Handler
	app     *newrelic.Application
	txn     *newrelic.Transaction

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
// Note: This function will silently fail, and always return a valid handler
// to avoid service disruptions. If you would prefer to handle when wrapping
// fails, use Wrap() instead.
func WrapHandler(app *newrelic.Application, handler slog.Handler) slog.Handler {
	if app == nil {
		return handler
	}
	if handler == nil {
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

var ErrNilApp = errors.New("New Relic application cannot be nil")
var ErrNilHandler = errors.New("slog handler cannot be nil")
var ErrAlreadyWrapped = errors.New("handler is already wrapped with a New Relic handler")

// WrapHandler returns a new handler that is wrapped with New Relic tools to capture
// log data based on your application's logs in context settings.
func Wrap(app *newrelic.Application, handler slog.Handler) (*NRHandler, error) {
	if app == nil {
		return nil, ErrNilApp
	}
	if handler == nil {
		return nil, ErrNilHandler
	}
	if _, ok := handler.(*NRHandler); ok {
		return nil, ErrAlreadyWrapped
	}

	return &NRHandler{
		handler: handler,
		app:     app,
	}, nil
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
	// exit quickly logs in context is disabled in the agent
	// to preserve resources
	if !h.isEnabled(h.app) {
		return h.handler.Handle(ctx, record)
	}

	// get transaction, preferring transaction from context
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

	var data newrelic.LogData

	if h.shouldForwardLogs(h.app) {
		attrs := h.getPreCompiledAttributes() // coppies cached attribute map, todo: optimize to avoid map
		prefix := h.getPrefix()

		record.Attrs(func(attr slog.Attr) bool {
			h.appendAttr(attrs, attr, prefix)
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
		h.enrichRecordTxn(nrTxn, &record)
	} else {
		if data.Message != "" {
			h.app.RecordLog(data)
		}
		h.enrichRecord(h.app, &record)
	}

	return h.handler.Handle(ctx, record)
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
	newHandler.computePrecompiledAttributes(newHandler.goas)
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
	newHandler.computePrecompiledAttributes(newHandler.goas)
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

const (
	nrlinking = "NR-LINKING"
	key       = "newrelic"
)

func (h *NRHandler) enrichRecord(app *newrelic.Application, record *slog.Record) {
	if !h.shouldEnrichLog(app) {
		return
	}

	str := nrLinkingString(app.GetLinkingMetadata())
	if str == "" {
		return
	}

	record.AddAttrs(slog.String(key, str))
}

func (h *NRHandler) enrichRecordTxn(txn *newrelic.Transaction, record *slog.Record) {
	if !h.shouldEnrichLog(txn.Application()) {
		return
	}

	str := nrLinkingString(txn.GetLinkingMetadata())
	if str == "" {
		return
	}

	record.AddAttrs(slog.String(key, str))
}

// nrLinkingString returns a string that represents the linking metadata
func nrLinkingString(data newrelic.LinkingMetadata) string {
	if data.EntityGUID == "" {
		return ""
	}

	len := 16 + len(data.EntityGUID) + len(data.Hostname) + len(data.TraceID) + len(data.SpanID) + len(data.EntityName)
	str := strings.Builder{}
	str.Grow(len) // only 1 alloc

	str.WriteString(nrlinking)
	str.WriteByte('|')
	str.WriteString(data.EntityGUID)
	str.WriteByte('|')
	str.WriteString(data.Hostname)
	str.WriteByte('|')
	str.WriteString(data.TraceID)
	str.WriteByte('|')
	str.WriteString(data.SpanID)
	str.WriteByte('|')
	str.WriteString(data.EntityName)
	str.WriteByte('|')

	return str.String()
}
