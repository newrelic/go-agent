// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func anotherFunction() {
	time.Sleep(1 * time.Millisecond)
}

func TestCodeLocation(t *testing.T) {
	loc1 := ThisCodeLocation()
	if loc1.LineNo != 18 || loc1.Function != "github.com/newrelic/go-agent/v3/newrelic.TestCodeLocation" || !strings.HasSuffix(loc1.FilePath, "/go-agent/v3/newrelic/code_level_metrics_test.go") {
		t.Errorf("CodeLocation() returned %v", loc1)
	}

	loc2, err := FunctionLocation(anotherFunction)
	if err != nil {
		t.Errorf("FunctionLocation() returned error %v", err)
	}
	if loc2.LineNo != 13 || loc2.Function != "github.com/newrelic/go-agent/v3/newrelic.anotherFunction" || !strings.HasSuffix(loc2.FilePath, "/go-agent/v3/newrelic/code_level_metrics_test.go") {
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
	if l.LineNo != 40 || l.Function != "github.com/newrelic/go-agent/v3/newrelic.TestClosureCLM.func1" || !strings.HasSuffix(l.FilePath, "/go-agent/v3/newrelic/code_level_metrics_test.go") {
		t.Errorf("closure FunctionLocation() returned %v", l)
	}
}

func TestBasicCaching(t *testing.T) {
	c := NewCachedCodeLocation()

	l, err := c.FunctionLocation(anotherFunction)
	if err != nil {
		t.Errorf("cached FunctionLocation error %v", err)
	}

	if l.LineNo != 13 || l.Function != "github.com/newrelic/go-agent/v3/newrelic.anotherFunction" || !strings.HasSuffix(l.FilePath, "/go-agent/v3/newrelic/code_level_metrics_test.go") {
		t.Errorf("FunctionLocation() returned %v", l)
	}

	if c.location == nil {
		t.Errorf("FunctionLocation cache location is nil")
	} else if c.location.LineNo != 13 || c.location.Function != "github.com/newrelic/go-agent/v3/newrelic.anotherFunction" || !strings.HasSuffix(c.location.FilePath, "/go-agent/v3/newrelic/code_level_metrics_test.go") {
		t.Errorf("FunctionLocation cache value is wrong %v", *c.location)
	}

	if c.Err() != nil {
		t.Errorf("FunctionLocation cache error %v", c.Err())
	}
}

func TestCachedCodeLocation(t *testing.T) {
	c := NewCachedCodeLocation()
	c2 := NewCachedCodeLocation()

	loc1 := c.ThisCodeLocation()
	if loc1.LineNo != 78 || loc1.Function != "github.com/newrelic/go-agent/v3/newrelic.TestCachedCodeLocation" || !strings.HasSuffix(loc1.FilePath, "/go-agent/v3/newrelic/code_level_metrics_test.go") {
		t.Errorf("CodeLocation() returned %v", loc1)
	}

	// This should give us the previously cached value, not the new
	// function passed. This is actually an example of a user error in the
	// code since they're reusing the cache for one code location on a call
	// to determine the location of an entirely different function. However,
	// since they specified a cache that now has a value cached in it, the defined
	// behavior is to use the cache.
	loc2, err := c.FunctionLocation(anotherFunction)
	if err != nil {
		t.Errorf("FunctionLocation() returned error %v", err)
	}
	if loc2.LineNo != 78 || loc2.Function != "github.com/newrelic/go-agent/v3/newrelic.TestCachedCodeLocation" || !strings.HasSuffix(loc2.FilePath, "/go-agent/v3/newrelic/code_level_metrics_test.go") {
		t.Errorf("FunctionLocation() returned %v", loc2)
	}

	// This is how we should have done it, using a separate cache for each
	// function location we're measuring. This should give us the true location
	loc2, err = c2.FunctionLocation(anotherFunction)
	if err != nil {
		t.Errorf("FunctionLocation() returned error %v", err)
	}
	if loc2.LineNo != 13 || loc2.Function != "github.com/newrelic/go-agent/v3/newrelic.anotherFunction" || !strings.HasSuffix(loc2.FilePath, "/go-agent/v3/newrelic/code_level_metrics_test.go") {
		t.Errorf("FunctionLocation() returned %v", loc2)
	}
}

func TestTraceOptions(t *testing.T) {
	var o traceOptSet
	WithCodeLocation(ThisCodeLocation())(&o)
	WithIgnoredPrefix("foo", "bar")(&o)
	WithPathPrefix("alpha", "beta", "gamma")(&o)
	WithoutCodeLevelMetrics()(&o)
	WithDefaultFunctionLocation(anotherFunction)(&o)

	if o.LocationOverride == nil {
		t.Errorf("failed to set a location")
	} else {
		if o.LocationOverride.LineNo != 110 || o.LocationOverride.Function != "github.com/newrelic/go-agent/v3/newrelic.TestTraceOptions" || !strings.HasSuffix(o.LocationOverride.FilePath, "/go-agent/v3/newrelic/code_level_metrics_test.go") {
			t.Errorf("function location set to %v", *o.LocationOverride)
		}
	}

	if !o.SuppressCLM {
		t.Errorf("asked to suppress CLM but that didn't show up")
	}

	if o.DemandCLM {
		t.Errorf("was not asked to demand CLM but that didn't show up")
	}

	if !reflect.DeepEqual(o.IgnoredPrefixes, []string{"foo", "bar"}) {
		t.Errorf("ignored prefixes wrong: %v", o.IgnoredPrefixes)
	}

	if !reflect.DeepEqual(o.PathPrefixes, []string{"alpha", "beta", "gamma"}) {
		t.Errorf("ignored prefixes wrong: %v", o.PathPrefixes)
	}
}

func TestTraceOptions2(t *testing.T) {
	var o traceOptSet
	WithPathPrefix("alpha")(&o)
	WithDefaultFunctionLocation(anotherFunction)(&o)
	WithCodeLevelMetrics()(&o)

	if o.LocationOverride == nil {
		t.Errorf("failed to set a location")
	} else {
		if o.LocationOverride.LineNo != 13 || o.LocationOverride.Function != "github.com/newrelic/go-agent/v3/newrelic.anotherFunction" || !strings.HasSuffix(o.LocationOverride.FilePath, "/go-agent/v3/newrelic/code_level_metrics_test.go") {
			t.Errorf("function location set to %v", *o.LocationOverride)
		}
	}

	if o.SuppressCLM {
		t.Errorf("was not asked to suppress CLM but that didn't show up")
	}

	if !o.DemandCLM {
		t.Errorf("asked to demand CLM but that didn't show up")
	}

	if o.IgnoredPrefixes != nil {
		t.Errorf("ignored prefixes wrong: %v", o.IgnoredPrefixes)
	}

	if !reflect.DeepEqual(o.PathPrefixes, []string{"alpha"}) {
		t.Errorf("ignored prefixes wrong: %v", o.PathPrefixes)
	}
}

func TestNullCache(t *testing.T) {
	// verify that given a zero-value cache, we still fall back to the non-cached version
	var c CachedCodeLocation

	l, err := c.FunctionLocation(anotherFunction)
	if err != nil {
		t.Errorf("cached FunctionLocation error %v", err)
	}

	if l.LineNo != 13 || l.Function != "github.com/newrelic/go-agent/v3/newrelic.anotherFunction" || !strings.HasSuffix(l.FilePath, "/go-agent/v3/newrelic/code_level_metrics_test.go") {
		t.Errorf("FunctionLocation() returned %v", l)
	}

	if c.location != nil {
		t.Errorf("FunctionLocation cache location is non-nil")
	}

	if c.Err() != nil {
		t.Errorf("FunctionLocation cache error %v", c.Err())
	}

	l = c.ThisCodeLocation()
	if l.LineNo != 193 || !strings.HasSuffix(l.Function, "TestNullCache") {
		t.Errorf("ThisCodeLocation line %v func %v", l.LineNo, l.Function)
	}
}

func skipA(t *testing.T) {
	skipB(t)
}

func skipB(t *testing.T) {
	skipC(t)
}

func skipC(t *testing.T) {
	l := ThisCodeLocation()
	if l.LineNo != 208 || !strings.HasSuffix(l.Function, "skipC") {
		t.Errorf("skipC shows as %v %v", l.LineNo, l.Function)
	}

	l = ThisCodeLocation(1)
	if l.LineNo != 204 || !strings.HasSuffix(l.Function, "skipB") {
		t.Errorf("skipB shows as %v %v", l.LineNo, l.Function)
	}

	l = ThisCodeLocation(2)
	if l.LineNo != 200 || !strings.HasSuffix(l.Function, "skipA") {
		t.Errorf("skipA shows as %v %v", l.LineNo, l.Function)
	}
}

func TestCLMSkip(t *testing.T) {
	skipA(t)
}

func skipACached(t *testing.T) {
	skipBCached(t)
}

func skipBCached(t *testing.T) {
	skipCCached(t)
}

func skipCCached(t *testing.T) {
	l := ThisCodeLocation()
	if l.LineNo != 237 || !strings.HasSuffix(l.Function, "skipCCached") {
		t.Errorf("skipC shows as %v %v", l.LineNo, l.Function)
	}

	l = ThisCodeLocation(1)
	if l.LineNo != 233 || !strings.HasSuffix(l.Function, "skipBCached") {
		t.Errorf("skipB shows as %v %v", l.LineNo, l.Function)
	}

	l = ThisCodeLocation(2)
	if l.LineNo != 229 || !strings.HasSuffix(l.Function, "skipACached") {
		t.Errorf("skipA shows as %v %v", l.LineNo, l.Function)
	}
}

func TestCLMSkipCached(t *testing.T) {
	skipACached(t)
}
