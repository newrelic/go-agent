package nrslog

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
)

// NRHandler is an Slog handler that includes logic to implement
// New Relic Logs in Context. Please always create a new handler
// using the Wrap() or WrapHandler() functions to ensure proper
// initialization.
//
// Note: shallow coppies of this handler may not duplicate underlying
// datastructures, and may cause logical errors. Please use the Clone()
// method to create deep coppies, or use the WithTransaction, WithAttrs,
// or WithGroup methods to create new handlers with additional data.
type NRHandler struct {
	*attributeCache
	*configCache
	*linkingCache

	// underlying object pointers
	handler slog.Handler
	app     *newrelic.Application
	txn     *newrelic.Transaction
}

// newHandler is an internal helper function to create a new NRHandler
func newHandler(app *newrelic.Application, handler slog.Handler) *NRHandler {
	return &NRHandler{
		handler:        handler,
		attributeCache: newAttributeCache(),
		configCache:    newConfigCache(),
		linkingCache:   newLinkingCache(),
		app:            app,
	}
}

// WrapHandler returns a new handler that is wrapped with New Relic tools to capture
// log data based on your application's logs in context settings.
//
// Note: This function will silently error, and always return a valid handler
// to avoid service disruptions. If you would prefer to handle errors when
// wrapping your handler, use the Wrap() function instead.
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
		return newHandler(app, handler)
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

	return newHandler(app, handler), nil
}

// New Returns a new slog.Logger object wrapped with a New Relic handler that controls
// logs in context features.
func New(app *newrelic.Application, handler slog.Handler) *slog.Logger {
	return slog.New(WrapHandler(app, handler))
}

// Clone creates a deep copy of the original handler, including a copy of all cached data
// and the underlying handler.
//
// Note: application, transaction, and handler pointers will be coppied, but the underlying
// data will not be duplicated.
func (h *NRHandler) Clone() *NRHandler {
	return &NRHandler{
		handler:        h.handler,
		attributeCache: h.attributeCache.clone(),
		configCache:    h.configCache.clone(),
		linkingCache:   h.linkingCache.clone(),
		app:            h.app,
		txn:            h.txn,
	}
}

// WithTransaction returns a new handler that is configured to capture log data
// and attribute it to a specific transaction.
func (h *NRHandler) WithTransaction(txn *newrelic.Transaction) *NRHandler {
	h2 := h.Clone()
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

	if h.shouldForwardLogs(h.app) {
		attrs := h.copyPreCompiledAttributes() // coppies cached attribute map, todo: optimize to avoid map coppies
		prefix := h.getPrefix()

		record.Attrs(func(attr slog.Attr) bool {
			h.appendAttr(attrs, attr, prefix)
			return true
		})

		data := newrelic.LogData{
			Severity:   record.Level.String(),
			Timestamp:  timestamp,
			Message:    record.Message,
			Attributes: attrs,
		}
		if nrTxn != nil {
			nrTxn.RecordLog(data)
		} else {
			h.app.RecordLog(data)
		}
	}

	// enrich logs
	if h.shouldEnrichLog(h.app) {
		if nrTxn != nil {
			h.enrichRecordTxn(nrTxn, &record)
		} else {
			h.enrichRecord(h.app, &record)
		}
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

	h2 := h.Clone()
	h2.handler = h.handler.WithAttrs(attrs)
	h2.precompileAttributes(attrs)
	return h2
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

	h2 := h.Clone()
	h2.handler = h.handler.WithGroup(name)
	h2.precompileGroup(name)
	return h2
}

// WithTransactionFromContext creates a wrapped NRHandler, enabling it to automatically reference New Relic
// transaction from context.
//
// Deprecated: this is a no-op
func WithTransactionFromContext(handler slog.Handler) slog.Handler {
	return handler
}

const newrelicAttributeKey = "newrelic"

func (h *NRHandler) enrichRecord(app *newrelic.Application, record *slog.Record) {
	str := nrLinkingString(h.getAgentLinkingMetadata(app))
	if str == "" {
		return
	}

	record.AddAttrs(slog.String(newrelicAttributeKey, str))
}

func (h *NRHandler) enrichRecordTxn(txn *newrelic.Transaction, record *slog.Record) {
	str := nrLinkingString(h.getTransactionLinkingMetadata(txn))
	if str == "" {
		return
	}

	record.AddAttrs(slog.String(newrelicAttributeKey, str))
}
