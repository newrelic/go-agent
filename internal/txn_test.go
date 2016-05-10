package internal

import (
	"testing"

	"github.com/newrelic/go-sdk/api"
)

func TestResponseCodeIsError(t *testing.T) {
	cfg := api.NewConfig("my app", "0123456789012345678901234567890123456789")

	if is := responseCodeIsError(&cfg, 200); is {
		t.Error(is)
	}
	if is := responseCodeIsError(&cfg, 400); !is {
		t.Error(is)
	}
	if is := responseCodeIsError(&cfg, 404); is {
		t.Error(is)
	}
	if is := responseCodeIsError(&cfg, 503); !is {
		t.Error(is)
	}
}
