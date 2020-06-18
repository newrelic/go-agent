// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package internal

import "testing"

func TestRulesCache(t *testing.T) {
	testcases := []struct {
		input  string
		isWeb  bool
		output string
	}{
		{input: "name1", isWeb: true, output: "WebTransaction/Go/name1"},
		{input: "name1", isWeb: false, output: "OtherTransaction/Go/name1"},
		{input: "name2", isWeb: true, output: "WebTransaction/Go/name2"},
		{input: "name3", isWeb: true, output: "WebTransaction/Go/name3"},
		{input: "zap/123/zip", isWeb: false, output: "OtherTransaction/Go/zap/*/zip"},
		{input: "zap/45/zip", isWeb: false, output: "OtherTransaction/Go/zap/*/zip"},
	}

	cache := newRulesCache(len(testcases))
	for _, tc := range testcases {
		// Test that nothing is in the cache before population.
		if out := cache.find(tc.input, tc.isWeb); out != "" {
			t.Error(out, tc.input, tc.isWeb)
		}
	}
	for _, tc := range testcases {
		cache.set(tc.input, tc.isWeb, tc.output)
	}
	for _, tc := range testcases {
		// Test that everything is now in the cache as expected.
		if out := cache.find(tc.input, tc.isWeb); out != tc.output {
			t.Error(out, tc.input, tc.isWeb, tc.output)
		}
	}
}

func TestRulesCacheLimit(t *testing.T) {
	cache := newRulesCache(1)
	cache.set("name1", true, "WebTransaction/Go/name1")
	cache.set("name1", false, "OtherTransaction/Go/name1")
	if out := cache.find("name1", true); out != "WebTransaction/Go/name1" {
		t.Error(out)
	}
	if out := cache.find("name1", false); out != "" {
		t.Error(out)
	}
}

func TestRulesCacheNil(t *testing.T) {
	var cache *rulesCache
	// No panics should happen if the rules cache pointer is nil.
	if out := cache.find("name1", true); "" != out {
		t.Error(out)
	}
	cache.set("name1", false, "OtherTransaction/Go/name1")
}
