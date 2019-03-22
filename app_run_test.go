package newrelic

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/newrelic/go-agent/internal"
)

func TestResponseCodeIsError(t *testing.T) {
	cfg := NewConfig("my app", "0123456789012345678901234567890123456789")
	cfg.ErrorCollector.IgnoreStatusCodes = append(cfg.ErrorCollector.IgnoreStatusCodes, 504)
	run := newAppRun(cfg, internal.ConnectReplyDefaults())

	if is := run.responseCodeIsError(200); is {
		t.Error(is)
	}
	if is := run.responseCodeIsError(400); !is {
		t.Error(is)
	}
	if is := run.responseCodeIsError(404); is {
		t.Error(is)
	}
	if is := run.responseCodeIsError(503); !is {
		t.Error(is)
	}
	if is := run.responseCodeIsError(504); is {
		t.Error(is)
	}
}

func TestCrossAppTracingEnabled(t *testing.T) {
	cfg := NewConfig("my app", "0123456789012345678901234567890123456789")
	run := newAppRun(cfg, internal.ConnectReplyDefaults())

	// CAT should be enabled by default.
	if enabled := run.crossApplicationTracingEnabled(); !enabled {
		t.Error(enabled)
	}

	// DT gets priority over CAT.
	run.Config.DistributedTracer.Enabled = true
	run.Config.CrossApplicationTracer.Enabled = true
	if enabled := run.crossApplicationTracingEnabled(); enabled {
		t.Error(enabled)
	}

	run.Config.DistributedTracer.Enabled = false
	run.Config.CrossApplicationTracer.Enabled = false
	if enabled := run.crossApplicationTracingEnabled(); enabled {
		t.Error(enabled)
	}

	run.Config.DistributedTracer.Enabled = false
	run.Config.CrossApplicationTracer.Enabled = true
	if enabled := run.crossApplicationTracingEnabled(); !enabled {
		t.Error(enabled)
	}
}

func TestTxnTraceThreshold(t *testing.T) {
	// Test that the default txn trace threshold is the failing apdex.
	cfg := NewConfig("my app", "0123456789012345678901234567890123456789")
	run := newAppRun(cfg, internal.ConnectReplyDefaults())
	threshold := run.txnTraceThreshold(1 * time.Second)
	if threshold != 4*time.Second {
		t.Error(threshold)
	}

	// Test that the trace threshold can be assigned to a fixed value.
	cfg = NewConfig("my app", "0123456789012345678901234567890123456789")
	cfg.TransactionTracer.Threshold.IsApdexFailing = false
	cfg.TransactionTracer.Threshold.Duration = 3 * time.Second
	run = newAppRun(cfg, internal.ConnectReplyDefaults())
	threshold = run.txnTraceThreshold(1 * time.Second)
	if threshold != 3*time.Second {
		t.Error(threshold)
	}

	// Test that the trace threshold can be overwritten by server-side-config.
	// with "apdex_f".
	cfg = NewConfig("my app", "0123456789012345678901234567890123456789")
	cfg.TransactionTracer.Threshold.IsApdexFailing = false
	cfg.TransactionTracer.Threshold.Duration = 3 * time.Second
	reply := internal.ConnectReplyDefaults()
	json.Unmarshal([]byte(`{"agent_config":{"transaction_tracer.transaction_threshold":"apdex_f"}}`), &reply)
	run = newAppRun(cfg, reply)
	threshold = run.txnTraceThreshold(1 * time.Second)
	if threshold != 4*time.Second {
		t.Error(threshold)
	}

	// Test that the trace threshold can be overwritten by server-side-config.
	// with a numberic value.
	cfg = NewConfig("my app", "0123456789012345678901234567890123456789")
	reply = internal.ConnectReplyDefaults()
	json.Unmarshal([]byte(`{"agent_config":{"transaction_tracer.transaction_threshold":3}}`), &reply)
	run = newAppRun(cfg, reply)
	threshold = run.txnTraceThreshold(1 * time.Second)
	if threshold != 3*time.Second {
		t.Error(threshold)
	}
}
