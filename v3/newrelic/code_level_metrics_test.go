// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"strings"
	"testing"
	"time"
)

func anotherFunction() {
	time.Sleep(1 * time.Millisecond)
}

func TestCodeLocation(t *testing.T) {
	loc1 := ThisCodeLocation()
	if loc1.LineNo != 17 || loc1.Function != "github.com/newrelic/go-agent/v3/newrelic.TestCodeLocation" || !strings.HasSuffix(loc1.FilePath, "/go-agent/v3/newrelic/code_level_metrics_test.go") {
		t.Errorf("CodeLocation() returned %v", loc1)
	}

	loc2, err := FunctionLocation(anotherFunction)
	if err != nil {
		t.Errorf("FunctionLocation() returned error %v", err)
	}
	if loc2.LineNo != 12 || loc2.Function != "github.com/newrelic/go-agent/v3/newrelic.anotherFunction" || !strings.HasSuffix(loc2.FilePath, "/go-agent/v3/newrelic/code_level_metrics_test.go") {
		t.Errorf("FunctionLocation() returned %v", loc2)
	}
}

func TestBadFunctionLocation(t *testing.T) {
	_, err := FunctionLocation(42)
	if err == nil {
		t.Errorf("Expected error with value 42 to FunctionLocation() but got nil")
	}
}

func TestClosureCLM(t *testing.T) {
	l, err := FunctionLocation(func() {
		anotherFunction()
	})
	if err != nil {
		t.Errorf("FunctionLocation of closure: %v", err)
	}
	if l.LineNo != 39 || l.Function != "github.com/newrelic/go-agent/v3/newrelic.TestClosureCLM.func1" || !strings.HasSuffix(l.FilePath, "/go-agent/v3/newrelic/code_level_metrics_test.go") {
		t.Errorf("closure FunctionLocation() returned %v", l)
	}
}
