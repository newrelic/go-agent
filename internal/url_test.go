package internal

import (
	"net/url"
	"strings"
	"testing"

	"github.com/newrelic/go-sdk/internal/crossagent"
)

func TestSafeURL(t *testing.T) {
	var testcases []struct {
		Testname string `json:"testname"`
		Expect   string `json:"expected"`
		Input    string `json:"input"`
	}

	err := crossagent.ReadJSON("url_clean.json", &testcases)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range testcases {
		if strings.Contains(tc.Input, ";") {
			// This test case was over defensive:
			// http://www.ietf.org/rfc/rfc3986.txt
			continue
		}

		// Only use testcases which have a scheme, otherwise the urls
		// may not be valid and may not be correctly handled by
		// url.Parse.
		if strings.HasPrefix(tc.Input, "p:") {
			u, err := url.Parse(tc.Input)
			if nil != err {
				t.Error(tc.Testname, tc.Input, err)
				continue
			}
			out := safeURL(u)
			if out != tc.Expect {
				t.Error(tc.Testname, tc.Input, tc.Expect)
			}
		}
	}
}
