package internal

import (
	"net/http"
	"testing"

	"github.com/newrelic/go-agent/api"
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

func TestHostFromRequestResponse(t *testing.T) {
	req, err := http.NewRequest("GET", "http://example.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	host := hostFromRequestResponse(req, &http.Response{Request: req})
	if host != "example.com" {
		t.Error("normal usage", host)
	}
	host = hostFromRequestResponse(nil, &http.Response{Request: req})
	if host != "example.com" {
		t.Error("missing request", host)
	}
	host = hostFromRequestResponse(req, nil)
	if host != "example.com" {
		t.Error("missing response", host)
	}
	host = hostFromRequestResponse(nil, nil)
	if host != "" {
		t.Error("missing request and response", host)
	}
	req.URL = nil
	host = hostFromRequestResponse(req, nil)
	if host != "" {
		t.Error("missing URL", host)
	}
}
