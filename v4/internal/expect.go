// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package internal

// Validator is used for testing.
type Validator interface {
	Error(...interface{})
}

// WantMetric is a metric expectation.  If Data is nil, then any data values are
// acceptable.  If Data has len 1, then only the metric count is validated.
type WantMetric struct {
	Name   string
	Scope  string
	Forced interface{} // true, false, or nil
	Data   []float64
}

// WantError is a traced error expectation.
type WantError struct {
	TxnName         string
	Msg             string
	Klass           string
	UserAttributes  map[string]interface{}
	AgentAttributes map[string]interface{}
}

func uniquePointer() *struct{} {
	s := struct{}{}
	return &s
}

var (
	// MatchAnything is for use when matching attributes.
	MatchAnything = uniquePointer()
	// MatchAnyString is a placeholder for matching any string
	MatchAnyString = "xxANY-STRINGxx"
)

// WantEvent is a transaction or error event expectation.
type WantEvent struct {
	Intrinsics      map[string]interface{}
	UserAttributes  map[string]interface{}
	AgentAttributes map[string]interface{}
}

// WantTxnTrace is a transaction trace expectation.
type WantTxnTrace struct {
	// DurationMillis is compared if non-nil.
	DurationMillis  *float64
	MetricName      string
	NumSegments     int
	UserAttributes  map[string]interface{}
	AgentAttributes map[string]interface{}
	Intrinsics      map[string]interface{}
	// If the Root's SegmentName is populated then the segments will be
	// tested, otherwise NumSegments will be tested.
	Root WantTraceSegment
}

// WantTraceSegment is a transaction trace segment expectation.
type WantTraceSegment struct {
	SegmentName string
	// RelativeStartMillis and RelativeStopMillis will be tested if they are
	// provided:  This makes it easy for top level tests which cannot
	// control duration.
	RelativeStartMillis interface{}
	RelativeStopMillis  interface{}
	Attributes          map[string]interface{}
	Children            []WantTraceSegment
}

// WantSlowQuery is a slowQuery expectation.
type WantSlowQuery struct {
	Count        int32
	MetricName   string
	Query        string
	TxnName      string
	TxnURL       string
	DatabaseName string
	Host         string
	PortPathOrID string
	Params       map[string]interface{}
}

// WantTxn provides the expectation parameters to ExpectTxnMetrics.
type WantTxn struct {
	Name      string
	IsWeb     bool
	NumErrors int
}

// Expect exposes methods that allow for testing whether the correct data was
// captured.
type Expect interface {
	ExpectCustomEvents(t Validator, want []WantEvent)
	ExpectErrors(t Validator, want []WantError)
	ExpectErrorEvents(t Validator, want []WantEvent)

	ExpectTxnEvents(t Validator, want []WantEvent)

	ExpectMetrics(t Validator, want []WantMetric)
	ExpectMetricsPresent(t Validator, want []WantMetric)
	ExpectTxnMetrics(t Validator, want WantTxn)

	ExpectTxnTraces(t Validator, want []WantTxnTrace)
	ExpectSlowQueries(t Validator, want []WantSlowQuery)

	ExpectSpanEvents(t Validator, want []WantEvent)
}
