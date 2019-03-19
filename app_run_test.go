package newrelic

import (
	"testing"

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
