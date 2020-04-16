// +build go1.9
// This build tag is necessary because Infinite Tracing is only supported for Go version 1.9 and up

package newrelic

import (
	"reflect"
	"testing"
)

func TestValidateTraceObserverURL(t *testing.T) {
	testcases := []struct {
		inputHost string
		inputPort int
		expectErr bool
		expectURL *observerURL
	}{
		{
			inputHost: "",
			expectErr: false,
			expectURL: nil,
		},
		{
			inputHost: "testing.com",
			expectErr: false,
			expectURL: &observerURL{
				host:   "testing.com:443",
				secure: true,
			},
		},
		{
			inputHost: "1.2.3.4",
			expectErr: false,
			expectURL: &observerURL{
				host:   "1.2.3.4:443",
				secure: true,
			},
		},
		{
			inputHost: "testing.com",
			inputPort: 123,
			expectErr: false,
			expectURL: &observerURL{
				host:   "testing.com:123",
				secure: true,
			},
		},
		{
			inputHost: "localhost",
			inputPort: 8080,
			expectErr: false,
			expectURL: &observerURL{
				host:   "localhost:8080",
				secure: false,
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.inputHost, func(t *testing.T) {
			c := defaultConfig()
			c.DistributedTracer.Enabled = true
			c.SpanEvents.Enabled = true
			c.InfiniteTracing.TraceObserver.Host = tc.inputHost
			if tc.inputPort != 0 {
				c.InfiniteTracing.TraceObserver.Port = tc.inputPort
			}
			url, err := c.validateTraceObserverConfig()

			if tc.expectErr && err == nil {
				t.Error("expected error, received nil")
			} else if !tc.expectErr && err != nil {
				t.Errorf("expected no error, but got one: %s", err)
			}

			if !reflect.DeepEqual(url, tc.expectURL) {
				t.Errorf("url is not as expected: actual=%#v expect=%#v", url, tc.expectURL)
			}
		})
	}
}

func Test8TConfig(t *testing.T) {
	testcases := []struct {
		host         string
		spansEnabled bool
		DTEnabled    bool
		validConfig  bool
	}{
		{
			host:         "localhost",
			spansEnabled: true,
			DTEnabled:    true,
			validConfig:  true,
		},
		{
			host:         "localhost",
			spansEnabled: false,
			DTEnabled:    true,
			validConfig:  false,
		},
		{
			host:         "localhost",
			spansEnabled: true,
			DTEnabled:    false,
			validConfig:  false,
		},
		{
			host:         "localhost",
			spansEnabled: false,
			DTEnabled:    false,
			validConfig:  false,
		},
		{
			host:         "",
			spansEnabled: false,
			DTEnabled:    false,
			validConfig:  true,
		},
	}

	for _, test := range testcases {
		cfg := Config{}
		cfg.License = "1234567890123456789012345678901234567890"
		cfg.AppName = "app"
		cfg.InfiniteTracing.TraceObserver.Host = test.host
		cfg.SpanEvents.Enabled = test.spansEnabled
		cfg.DistributedTracer.Enabled = test.DTEnabled

		_, err := newInternalConfig(cfg, func(s string) string { return "" }, []string{})
		if (err == nil) != test.validConfig {
			t.Errorf("Infite Tracing config validation failed: %v", test)
		}

	}
}
