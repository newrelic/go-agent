// +build go1.9
// This build tag is necessary because Infinite Tracing is only supported for Go version 1.9 and up

package newrelic

import (
	"reflect"
	"testing"
)

func TestValidateTraceObserverURL(t *testing.T) {
	testcases := []struct {
		inputURL  string
		expectErr bool
		expectURL *observerURL
	}{
		{
			inputURL:  "",
			expectErr: false,
			expectURL: nil,
		},
		{
			inputURL:  "https://testing.com",
			expectErr: false,
			expectURL: &observerURL{
				host:   "testing.com:443",
				secure: true,
			},
		},
		{
			inputURL:  "https://1.2.3.4",
			expectErr: false,
			expectURL: &observerURL{
				host:   "1.2.3.4:443",
				secure: true,
			},
		},
		{
			inputURL:  "https://1.2.3.4:",
			expectErr: false,
			expectURL: &observerURL{
				host:   "1.2.3.4:443",
				secure: true,
			},
		},
		{
			inputURL:  "http://1.2.3.4:",
			expectErr: false,
			expectURL: &observerURL{
				host:   "1.2.3.4:80",
				secure: false,
			},
		},
		{
			inputURL:  "http://testing.com",
			expectErr: false,
			expectURL: &observerURL{
				host:   "testing.com:80",
				secure: false,
			},
		},
		{
			inputURL:  "https://testing.com/",
			expectErr: false,
			expectURL: &observerURL{
				host:   "testing.com:443/",
				secure: true,
			},
		},
		{
			inputURL:  "//not valid url",
			expectErr: true,
			expectURL: nil,
		},
		{
			inputURL:  "this has no host",
			expectErr: true,
			expectURL: nil,
		},
		{
			inputURL:  "https://testing.com/with/path",
			expectErr: false,
			expectURL: &observerURL{
				host:   "testing.com:443/with/path",
				secure: true,
			},
		},
		{
			inputURL:  "https://testing.com?with=queries",
			expectErr: false,
			expectURL: &observerURL{
				host:   "testing.com:443",
				secure: true,
			},
		},
		{
			inputURL:  "https://testing.com:123",
			expectErr: false,
			expectURL: &observerURL{
				host:   "testing.com:123",
				secure: true,
			},
		},
		{
			inputURL:  "testing.com",
			expectErr: true,
			expectURL: nil,
		},
		{
			inputURL:  "testing.com:443",
			expectErr: true,
			expectURL: nil,
		},
		{
			inputURL:  "grpc://testing.com",
			expectErr: true,
			expectURL: nil,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.inputURL, func(t *testing.T) {
			c := defaultConfig()
			c.DistributedTracer.Enabled = true
			c.SpanEvents.Enabled = true
			c.InfiniteTracing.TraceObserverURL = tc.inputURL
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
		URL          string
		spansEnabled bool
		DTEnabled    bool
		validConfig  bool
	}{
		{
			URL:          "http://localhost:8080",
			spansEnabled: true,
			DTEnabled:    true,
			validConfig:  true,
		},
		{
			URL:          "http://localhost:8080",
			spansEnabled: false,
			DTEnabled:    true,
			validConfig:  false,
		},
		{
			URL:          "http://localhost:8080",
			spansEnabled: true,
			DTEnabled:    false,
			validConfig:  false,
		},
		{
			URL:          "http://localhost:8080",
			spansEnabled: false,
			DTEnabled:    false,
			validConfig:  false,
		},
		{
			URL:          "",
			spansEnabled: false,
			DTEnabled:    false,
			validConfig:  true,
		},
	}

	for _, test := range testcases {
		cfg := Config{}
		cfg.License = "1234567890123456789012345678901234567890"
		cfg.AppName = "app"
		cfg.InfiniteTracing.TraceObserverURL = test.URL
		cfg.SpanEvents.Enabled = test.spansEnabled
		cfg.DistributedTracer.Enabled = test.DTEnabled

		_, err := newInternalConfig(cfg, func(s string) string { return "" }, []string{})
		if (err == nil) != test.validConfig {
			t.Errorf("Infite Tracing config validation failed: %v", test)
		}

	}
}
