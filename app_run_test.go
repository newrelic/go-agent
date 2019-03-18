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
