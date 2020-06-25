// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/newrelic/go-agent/v3/internal"
)

var (
	start = time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	end   = time.Date(2014, time.November, 28, 1, 2, 0, 0, time.UTC)
)

func TestEmptyMetrics(t *testing.T) {
	mt := newMetricTable(20, start)
	js, err := mt.CollectorJSON(`12345`, end)
	if nil != err {
		t.Fatal(err)
	}
	if nil != js {
		t.Error(string(js))
	}
}

func isValidJSON(data []byte) error {
	var v interface{}

	return json.Unmarshal(data, &v)
}

func TestMetrics(t *testing.T) {
	mt := newMetricTable(20, start)

	mt.addDuration("one", "", 2*time.Second, 1*time.Second, unforced)
	mt.addDuration("two", "my_scope", 4*time.Second, 2*time.Second, unforced)
	mt.addDuration("one", "my_scope", 2*time.Second, 1*time.Second, unforced)
	mt.addDuration("one", "", 2*time.Second, 1*time.Second, unforced)

	mt.addApdex("apdex satisfied", "", 9*time.Second, apdexSatisfying, unforced)
	mt.addApdex("apdex satisfied", "", 8*time.Second, apdexSatisfying, unforced)
	mt.addApdex("apdex tolerated", "", 7*time.Second, apdexTolerating, unforced)
	mt.addApdex("apdex tolerated", "", 8*time.Second, apdexTolerating, unforced)
	mt.addApdex("apdex failed", "my_scope", 1*time.Second, apdexFailing, unforced)

	mt.addCount("count 123", float64(123), unforced)
	mt.addSingleCount("count 1", unforced)

	expectMetrics(t, mt, []internal.WantMetric{
		{Name: "apdex satisfied", Scope: "", Forced: false, Data: []float64{2, 0, 0, 8, 9, 0}},
		{Name: "apdex tolerated", Scope: "", Forced: false, Data: []float64{0, 2, 0, 7, 8, 0}},
		{Name: "one", Scope: "", Forced: false, Data: []float64{2, 4, 2, 2, 2, 8}},
		{Name: "apdex failed", Scope: "my_scope", Forced: false, Data: []float64{0, 0, 1, 1, 1, 0}},
		{Name: "one", Scope: "my_scope", Forced: false, Data: []float64{1, 2, 1, 2, 2, 4}},
		{Name: "two", Scope: "my_scope", Forced: false, Data: []float64{1, 4, 2, 4, 4, 16}},
		{Name: "count 123", Scope: "", Forced: false, Data: []float64{123, 0, 0, 0, 0, 0}},
		{Name: "count 1", Scope: "", Forced: false, Data: []float64{1, 0, 0, 0, 0, 0}},
	})

	js, err := mt.Data("12345", end)
	if nil != err {
		t.Error(err)
	}
	// The JSON metric order is not deterministic, so we merely test that it
	// is valid JSON.
	if err := isValidJSON(js); nil != err {
		t.Error(err, string(js))
	}
}

func TestApplyRules(t *testing.T) {
	js := `[
	{
		"ignore":false,
		"each_segment":false,
		"terminate_chain":true,
		"replacement":"been_renamed",
		"replace_all":false,
		"match_expression":"one$",
		"eval_order":1
	},
	{
		"ignore":true,
		"each_segment":false,
		"terminate_chain":true,
		"replace_all":false,
		"match_expression":"ignore_me",
		"eval_order":1
	},
	{
		"ignore":false,
		"each_segment":false,
		"terminate_chain":true,
		"replacement":"merge_me",
		"replace_all":false,
		"match_expression":"merge_me[0-9]+$",
		"eval_order":1
	}
	]`
	var rules internal.MetricRules
	err := json.Unmarshal([]byte(js), &rules)
	if nil != err {
		t.Fatal(err)
	}

	mt := newMetricTable(20, start)
	mt.addDuration("one", "", 2*time.Second, 1*time.Second, unforced)
	mt.addDuration("one", "scope1", 2*time.Second, 1*time.Second, unforced)
	mt.addDuration("one", "scope2", 2*time.Second, 1*time.Second, unforced)
	mt.addDuration("ignore_me", "", 2*time.Second, 1*time.Second, unforced)
	mt.addDuration("ignore_me", "scope1", 2*time.Second, 1*time.Second, unforced)
	mt.addDuration("ignore_me", "scope2", 2*time.Second, 1*time.Second, unforced)
	mt.addDuration("merge_me1", "", 2*time.Second, 1*time.Second, unforced)
	mt.addDuration("merge_me2", "", 2*time.Second, 1*time.Second, unforced)

	applied := mt.ApplyRules(rules)
	expectMetrics(t, applied, []internal.WantMetric{
		{Name: "been_renamed", Scope: "", Forced: false, Data: []float64{1, 2, 1, 2, 2, 4}},
		{Name: "been_renamed", Scope: "scope1", Forced: false, Data: []float64{1, 2, 1, 2, 2, 4}},
		{Name: "been_renamed", Scope: "scope2", Forced: false, Data: []float64{1, 2, 1, 2, 2, 4}},
		{Name: "merge_me", Scope: "", Forced: false, Data: []float64{2, 4, 2, 2, 2, 8}},
	})
}

func TestApplyEmptyRules(t *testing.T) {
	js := `[]`
	var rules internal.MetricRules
	err := json.Unmarshal([]byte(js), &rules)
	if nil != err {
		t.Fatal(err)
	}
	mt := newMetricTable(20, start)
	mt.addDuration("one", "", 2*time.Second, 1*time.Second, unforced)
	mt.addDuration("one", "my_scope", 2*time.Second, 1*time.Second, unforced)
	applied := mt.ApplyRules(rules)
	expectMetrics(t, applied, []internal.WantMetric{
		{Name: "one", Scope: "", Forced: false, Data: []float64{1, 2, 1, 2, 2, 4}},
		{Name: "one", Scope: "my_scope", Forced: false, Data: []float64{1, 2, 1, 2, 2, 4}},
	})
}

func TestApplyNilRules(t *testing.T) {
	var rules internal.MetricRules

	mt := newMetricTable(20, start)
	mt.addDuration("one", "", 2*time.Second, 1*time.Second, unforced)
	mt.addDuration("one", "my_scope", 2*time.Second, 1*time.Second, unforced)
	applied := mt.ApplyRules(rules)
	expectMetrics(t, applied, []internal.WantMetric{
		{Name: "one", Scope: "", Forced: false, Data: []float64{1, 2, 1, 2, 2, 4}},
		{Name: "one", Scope: "my_scope", Forced: false, Data: []float64{1, 2, 1, 2, 2, 4}},
	})
}

func TestForced(t *testing.T) {
	mt := newMetricTable(0, start)

	mt.addDuration("unforced", "", 1*time.Second, 1*time.Second, unforced)
	mt.addDuration("forced", "", 2*time.Second, 2*time.Second, forced)

	expectMetrics(t, mt, []internal.WantMetric{
		{Name: "forced", Scope: "", Forced: true, Data: []float64{1, 2, 2, 2, 2, 4}},
		{Name: supportabilityDropped, Scope: "", Forced: true, Data: []float64{1, 0, 0, 0, 0, 0}},
	})

}

func TestMetricsMergeIntoEmpty(t *testing.T) {
	src := newMetricTable(20, start)
	src.addDuration("one", "", 2*time.Second, 1*time.Second, unforced)
	src.addDuration("two", "", 2*time.Second, 1*time.Second, unforced)
	dest := newMetricTable(20, start)
	dest.merge(src, "")

	expectMetrics(t, dest, []internal.WantMetric{
		{Name: "one", Scope: "", Forced: false, Data: []float64{1, 2, 1, 2, 2, 4}},
		{Name: "two", Scope: "", Forced: false, Data: []float64{1, 2, 1, 2, 2, 4}},
	})
}

func TestMetricsMergeFromEmpty(t *testing.T) {
	src := newMetricTable(20, start)
	dest := newMetricTable(20, start)
	dest.addDuration("one", "", 2*time.Second, 1*time.Second, unforced)
	dest.addDuration("two", "", 2*time.Second, 1*time.Second, unforced)
	dest.merge(src, "")

	expectMetrics(t, dest, []internal.WantMetric{
		{Name: "one", Scope: "", Forced: false, Data: []float64{1, 2, 1, 2, 2, 4}},
		{Name: "two", Scope: "", Forced: false, Data: []float64{1, 2, 1, 2, 2, 4}},
	})
}

func TestMetricsMerge(t *testing.T) {
	src := newMetricTable(20, start)
	dest := newMetricTable(20, start)
	dest.addDuration("one", "", 2*time.Second, 1*time.Second, unforced)
	dest.addDuration("two", "", 2*time.Second, 1*time.Second, unforced)
	src.addDuration("two", "", 2*time.Second, 1*time.Second, unforced)
	src.addDuration("three", "", 2*time.Second, 1*time.Second, unforced)

	dest.merge(src, "")

	expectMetrics(t, dest, []internal.WantMetric{
		{Name: "one", Scope: "", Forced: false, Data: []float64{1, 2, 1, 2, 2, 4}},
		{Name: "two", Scope: "", Forced: false, Data: []float64{2, 4, 2, 2, 2, 8}},
		{Name: "three", Scope: "", Forced: false, Data: []float64{1, 2, 1, 2, 2, 4}},
	})
}

func TestMergeFailedSuccess(t *testing.T) {
	src := newMetricTable(20, start)
	dest := newMetricTable(20, end)
	dest.addDuration("one", "", 2*time.Second, 1*time.Second, unforced)
	dest.addDuration("two", "", 2*time.Second, 1*time.Second, unforced)
	src.addDuration("two", "", 2*time.Second, 1*time.Second, unforced)
	src.addDuration("three", "", 2*time.Second, 1*time.Second, unforced)

	if 0 != dest.failedHarvests {
		t.Fatal(dest.failedHarvests)
	}

	dest.mergeFailed(src)

	expectMetrics(t, dest, []internal.WantMetric{
		{Name: "one", Scope: "", Forced: false, Data: []float64{1, 2, 1, 2, 2, 4}},
		{Name: "two", Scope: "", Forced: false, Data: []float64{2, 4, 2, 2, 2, 8}},
		{Name: "three", Scope: "", Forced: false, Data: []float64{1, 2, 1, 2, 2, 4}},
	})
}

func TestMergeFailedLimitReached(t *testing.T) {
	src := newMetricTable(20, start)
	dest := newMetricTable(20, end)
	dest.addDuration("one", "", 2*time.Second, 1*time.Second, unforced)
	dest.addDuration("two", "", 2*time.Second, 1*time.Second, unforced)
	src.addDuration("two", "", 2*time.Second, 1*time.Second, unforced)
	src.addDuration("three", "", 2*time.Second, 1*time.Second, unforced)

	src.failedHarvests = failedMetricAttemptsLimit

	dest.mergeFailed(src)

	expectMetrics(t, dest, []internal.WantMetric{
		{Name: "one", Scope: "", Forced: false, Data: []float64{1, 2, 1, 2, 2, 4}},
		{Name: "two", Scope: "", Forced: false, Data: []float64{1, 2, 1, 2, 2, 4}},
	})
}

func BenchmarkMetricTableCollectorJSON(b *testing.B) {
	mt := newMetricTable(2000, time.Now())
	md := metricData{
		countSatisfied:  1234567812345678.1234567812345678,
		totalTolerated:  1234567812345678.1234567812345678,
		exclusiveFailed: 1234567812345678.1234567812345678,
		min:             1234567812345678.1234567812345678,
		max:             1234567812345678.1234567812345678,
		sumSquares:      1234567812345678.1234567812345678,
	}

	for i := 0; i < 20; i++ {
		scope := fmt.Sprintf("WebTransaction/Uri/myblog2/%d", i)

		for j := 0; j < 20; j++ {
			name := fmt.Sprintf("Datastore/statement/MySQL/City%d/insert", j)
			mt.add(name, "", md, forced)
			mt.add(name, scope, md, forced)

			name = fmt.Sprintf("WebTransaction/Uri/myblog2/newPost_rum_%d.php", j)
			mt.add(name, "", md, forced)
			mt.add(name, scope, md, forced)
		}
	}

	data, err := mt.CollectorJSON("12345", time.Now())
	if nil != err {
		b.Fatal(err)
	}
	if err := isValidJSON(data); nil != err {
		b.Fatal(err, string(data))
	}

	b.ResetTimer()
	b.ReportAllocs()

	id := "12345"
	now := time.Now()
	for i := 0; i < b.N; i++ {
		mt.CollectorJSON(id, now)
	}
}

func BenchmarkAddingSameMetrics(b *testing.B) {
	name := "my_name"
	scope := "my_scope"
	duration := 2 * time.Second
	exclusive := 1 * time.Second

	mt := newMetricTable(2000, time.Now())

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		mt.addDuration(name, scope, duration, exclusive, forced)
		mt.addSingleCount(name, forced)
	}
}

func TestMergedMetricsAreCopied(t *testing.T) {
	src := newMetricTable(20, start)
	dest := newMetricTable(20, start)

	src.addSingleCount("zip", unforced)
	dest.merge(src, "")
	src.addSingleCount("zip", unforced)
	expectMetrics(t, dest, []internal.WantMetric{
		{Name: "zip", Scope: "", Forced: false, Data: []float64{1, 0, 0, 0, 0, 0}},
	})
}

func TestMergedWithScope(t *testing.T) {
	src := newMetricTable(20, start)
	dest := newMetricTable(20, start)

	src.addSingleCount("one", unforced)
	src.addDuration("two", "", 2*time.Second, 1*time.Second, unforced)
	dest.addDuration("two", "my_scope", 2*time.Second, 1*time.Second, unforced)
	dest.merge(src, "my_scope")

	expectMetrics(t, dest, []internal.WantMetric{
		{Name: "one", Scope: "my_scope", Forced: false, Data: []float64{1, 0, 0, 0, 0, 0}},
		{Name: "two", Scope: "my_scope", Forced: false, Data: []float64{2, 4, 2, 2, 2, 8}},
	})
}
