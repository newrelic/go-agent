// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/cat"
	"github.com/newrelic/go-agent/v3/internal/crossagent"
	"github.com/newrelic/go-agent/v3/internal/logger"
)

func trueFunc() bool  { return true }
func falseFunc() bool { return false }

func TestStartEndSegment(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)

	txndata := &txnData{}
	thread := &tracingThread{}
	token := startSegment(txndata, thread, start)
	stop := start.Add(1 * time.Second)
	end, err := endSegment(txndata, thread, token, stop)
	if nil != err {
		t.Error(err)
	}
	if end.exclusive != end.duration {
		t.Error(end.exclusive, end.duration)
	}
	if end.duration != 1*time.Second {
		t.Error(end.duration)
	}
	if end.start.Time != start {
		t.Error(end.start, start)
	}
	if end.stop.Time != stop {
		t.Error(end.stop, stop)
	}
	if 0 != len(txndata.SpanEvents) {
		t.Error(txndata.SpanEvents)
	}
}

func TestMultipleChildren(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &txnData{}
	thread := &tracingThread{}

	t1 := startSegment(txndata, thread, start.Add(1*time.Second))
	t2 := startSegment(txndata, thread, start.Add(2*time.Second))
	end2, err2 := endSegment(txndata, thread, t2, start.Add(3*time.Second))
	t3 := startSegment(txndata, thread, start.Add(4*time.Second))
	end3, err3 := endSegment(txndata, thread, t3, start.Add(5*time.Second))
	end1, err1 := endSegment(txndata, thread, t1, start.Add(6*time.Second))
	t4 := startSegment(txndata, thread, start.Add(7*time.Second))
	end4, err4 := endSegment(txndata, thread, t4, start.Add(8*time.Second))

	if nil != err1 || end1.duration != 5*time.Second || end1.exclusive != 3*time.Second {
		t.Error(end1, err1)
	}
	if nil != err2 || end2.duration != end2.exclusive || end2.duration != time.Second {
		t.Error(end2, err2)
	}
	if nil != err3 || end3.duration != end3.exclusive || end3.duration != time.Second {
		t.Error(end3, err3)
	}
	if nil != err4 || end4.duration != end4.exclusive || end4.duration != time.Second {
		t.Error(end4, err4)
	}
	if thread.TotalTime() != 7*time.Second {
		t.Error(thread.TotalTime())
	}
}

func TestInvalidStart(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &txnData{}
	thread := &tracingThread{}

	end, err := endSegment(txndata, thread, segmentStartTime{}, start.Add(1*time.Second))
	if err != errMalformedSegment {
		t.Error(end, err)
	}
	startSegment(txndata, thread, start.Add(2*time.Second))
	end, err = endSegment(txndata, thread, segmentStartTime{}, start.Add(3*time.Second))
	if err != errMalformedSegment {
		t.Error(end, err)
	}
}

func TestSegmentAlreadyEnded(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &txnData{}
	thread := &tracingThread{}

	t1 := startSegment(txndata, thread, start.Add(1*time.Second))
	end, err := endSegment(txndata, thread, t1, start.Add(2*time.Second))
	if err != nil {
		t.Error(end, err)
	}
	end, err = endSegment(txndata, thread, t1, start.Add(3*time.Second))
	if err != errSegmentOrder {
		t.Error(end, err)
	}
}

func TestSegmentBadStamp(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &txnData{}
	thread := &tracingThread{}

	t1 := startSegment(txndata, thread, start.Add(1*time.Second))
	t1.Stamp++
	end, err := endSegment(txndata, thread, t1, start.Add(2*time.Second))
	if err != errSegmentOrder {
		t.Error(end, err)
	}
}

func TestSegmentBadDepth(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &txnData{}
	thread := &tracingThread{}

	t1 := startSegment(txndata, thread, start.Add(1*time.Second))
	t1.Depth++
	end, err := endSegment(txndata, thread, t1, start.Add(2*time.Second))
	if err != errSegmentOrder {
		t.Error(end, err)
	}
}

func TestSegmentNegativeDepth(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &txnData{}
	thread := &tracingThread{}

	t1 := startSegment(txndata, thread, start.Add(1*time.Second))
	t1.Depth = -1
	end, err := endSegment(txndata, thread, t1, start.Add(2*time.Second))
	if err != errMalformedSegment {
		t.Error(end, err)
	}
}

func TestSegmentOutOfOrder(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &txnData{}
	thread := &tracingThread{}

	t1 := startSegment(txndata, thread, start.Add(1*time.Second))
	t2 := startSegment(txndata, thread, start.Add(2*time.Second))
	t3 := startSegment(txndata, thread, start.Add(3*time.Second))
	end2, err2 := endSegment(txndata, thread, t2, start.Add(4*time.Second))
	end3, err3 := endSegment(txndata, thread, t3, start.Add(5*time.Second))
	t4 := startSegment(txndata, thread, start.Add(6*time.Second))
	end4, err4 := endSegment(txndata, thread, t4, start.Add(7*time.Second))
	end1, err1 := endSegment(txndata, thread, t1, start.Add(8*time.Second))

	if nil != err1 ||
		end1.duration != 7*time.Second ||
		end1.exclusive != 4*time.Second {
		t.Error(end1, err1)
	}
	if nil != err2 || end2.duration != end2.exclusive || end2.duration != 2*time.Second {
		t.Error(end2, err2)
	}
	if err3 != errSegmentOrder {
		t.Error(end3, err3)
	}
	if nil != err4 || end4.duration != end4.exclusive || end4.duration != 1*time.Second {
		t.Error(end4, err4)
	}
}

//                                          |-t3-|    |-t4-|
//                           |-t2-|    |-never-finished----------
//            |-t1-|    |--never-finished------------------------
//       |-------alpha------------------------------------------|
//  0    1    2    3    4    5    6    7    8    9    10   11   12
func TestLostChildren(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &txnData{}
	thread := &tracingThread{}

	alpha := startSegment(txndata, thread, start.Add(1*time.Second))
	t1 := startSegment(txndata, thread, start.Add(2*time.Second))
	endBasicSegment(txndata, thread, t1, start.Add(3*time.Second), "t1")
	startSegment(txndata, thread, start.Add(4*time.Second))
	t2 := startSegment(txndata, thread, start.Add(5*time.Second))
	endBasicSegment(txndata, thread, t2, start.Add(6*time.Second), "t2")
	startSegment(txndata, thread, start.Add(7*time.Second))
	t3 := startSegment(txndata, thread, start.Add(8*time.Second))
	endBasicSegment(txndata, thread, t3, start.Add(9*time.Second), "t3")
	t4 := startSegment(txndata, thread, start.Add(10*time.Second))
	endBasicSegment(txndata, thread, t4, start.Add(11*time.Second), "t4")
	endBasicSegment(txndata, thread, alpha, start.Add(12*time.Second), "alpha")

	metrics := newMetricTable(100, time.Now())
	txndata.FinalName = "WebTransaction/Go/zip"
	txndata.IsWeb = true
	mergeBreakdownMetrics(txndata, metrics)
	expectMetrics(t, metrics, []internal.WantMetric{
		{Name: "Custom/alpha", Scope: "", Forced: false, Data: []float64{1, 11, 7, 11, 11, 121}},
		{Name: "Custom/t1", Scope: "", Forced: false, Data: []float64{1, 1, 1, 1, 1, 1}},
		{Name: "Custom/t2", Scope: "", Forced: false, Data: []float64{1, 1, 1, 1, 1, 1}},
		{Name: "Custom/t3", Scope: "", Forced: false, Data: []float64{1, 1, 1, 1, 1, 1}},
		{Name: "Custom/t4", Scope: "", Forced: false, Data: []float64{1, 1, 1, 1, 1, 1}},
		{Name: "Custom/alpha", Scope: txndata.FinalName, Forced: false, Data: []float64{1, 11, 7, 11, 11, 121}},
		{Name: "Custom/t1", Scope: txndata.FinalName, Forced: false, Data: []float64{1, 1, 1, 1, 1, 1}},
		{Name: "Custom/t2", Scope: txndata.FinalName, Forced: false, Data: []float64{1, 1, 1, 1, 1, 1}},
		{Name: "Custom/t3", Scope: txndata.FinalName, Forced: false, Data: []float64{1, 1, 1, 1, 1, 1}},
		{Name: "Custom/t4", Scope: txndata.FinalName, Forced: false, Data: []float64{1, 1, 1, 1, 1, 1}},
	})
}

//                                          |-t3-|    |-t4-|
//                           |-t2-|    |-never-finished----------
//            |-t1-|    |--never-finished------------------------
//  |-------root-------------------------------------------------
//  0    1    2    3    4    5    6    7    8    9    10   11   12
func TestLostChildrenRoot(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &txnData{}
	thread := &tracingThread{}

	t1 := startSegment(txndata, thread, start.Add(2*time.Second))
	endBasicSegment(txndata, thread, t1, start.Add(3*time.Second), "t1")
	startSegment(txndata, thread, start.Add(4*time.Second))
	t2 := startSegment(txndata, thread, start.Add(5*time.Second))
	endBasicSegment(txndata, thread, t2, start.Add(6*time.Second), "t2")
	startSegment(txndata, thread, start.Add(7*time.Second))
	t3 := startSegment(txndata, thread, start.Add(8*time.Second))
	endBasicSegment(txndata, thread, t3, start.Add(9*time.Second), "t3")
	t4 := startSegment(txndata, thread, start.Add(10*time.Second))
	endBasicSegment(txndata, thread, t4, start.Add(11*time.Second), "t4")

	if thread.TotalTime() != 9*time.Second {
		t.Error(thread.TotalTime())
	}

	metrics := newMetricTable(100, time.Now())
	txndata.FinalName = "WebTransaction/Go/zip"
	txndata.IsWeb = true
	mergeBreakdownMetrics(txndata, metrics)
	expectMetrics(t, metrics, []internal.WantMetric{
		{Name: "Custom/t1", Scope: "", Forced: false, Data: []float64{1, 1, 1, 1, 1, 1}},
		{Name: "Custom/t2", Scope: "", Forced: false, Data: []float64{1, 1, 1, 1, 1, 1}},
		{Name: "Custom/t3", Scope: "", Forced: false, Data: []float64{1, 1, 1, 1, 1, 1}},
		{Name: "Custom/t4", Scope: "", Forced: false, Data: []float64{1, 1, 1, 1, 1, 1}},
		{Name: "Custom/t1", Scope: txndata.FinalName, Forced: false, Data: []float64{1, 1, 1, 1, 1, 1}},
		{Name: "Custom/t2", Scope: txndata.FinalName, Forced: false, Data: []float64{1, 1, 1, 1, 1, 1}},
		{Name: "Custom/t3", Scope: txndata.FinalName, Forced: false, Data: []float64{1, 1, 1, 1, 1, 1}},
		{Name: "Custom/t4", Scope: txndata.FinalName, Forced: false, Data: []float64{1, 1, 1, 1, 1, 1}},
	})
}

func TestNilSpanEvent(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)

	txndata := &txnData{}
	thread := &tracingThread{}
	token := startSegment(txndata, thread, start)
	stop := start.Add(1 * time.Second)
	end, err := endSegment(txndata, thread, token, stop)
	if nil != err {
		t.Error(err)
	}

	// A segment without a SpanId does not create a spanEvent.
	if evt := end.spanEvent(); evt != nil {
		t.Error(evt)
	}
}

func TestDefaultSpanEvent(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)

	txndata := &txnData{}
	thread := &tracingThread{}
	token := startSegment(txndata, thread, start)
	stop := start.Add(1 * time.Second)
	end, err := endSegment(txndata, thread, token, stop)
	if nil != err {
		t.Error(err)
	}
	end.SpanID = "123"
	if evt := end.spanEvent(); evt != nil {
		if evt.GUID != end.SpanID ||
			evt.ParentID != end.ParentID ||
			evt.Timestamp != end.start.Time ||
			evt.Duration != end.duration ||
			evt.IsEntrypoint {
			t.Error(evt)
		}
	}
}

func TestGetRootSpanID(t *testing.T) {
	txndata := &txnData{
		TraceIDGenerator: internal.NewTraceIDGenerator(12345),
	}
	if id := txndata.GetRootSpanID(); id != "1ae969564b34a33e" {
		t.Error(id)
	}
	if id := txndata.GetRootSpanID(); id != "1ae969564b34a33e" {
		t.Error(id)
	}
}

func TestCurrentSpanIdentifier(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &txnData{
		TraceIDGenerator: internal.NewTraceIDGenerator(12345),
	}
	thread := &tracingThread{}
	id := txndata.CurrentSpanIdentifier(thread)
	if id != "1ae969564b34a33e" {
		t.Error(id)
	}

	// After starting and ending a segment, the current span id is still the root.
	t1 := startSegment(txndata, thread, start.Add(1*time.Second))
	_, err1 := endSegment(txndata, thread, t1, start.Add(3*time.Second))
	if nil != err1 {
		t.Error(err1)
	}

	id = txndata.CurrentSpanIdentifier(thread)
	if id != "1ae969564b34a33e" {
		t.Error(id)
	}

	// After starting a new segment, there should be a new current span id.
	startSegment(txndata, thread, start.Add(2*time.Second))
	id2 := txndata.CurrentSpanIdentifier(thread)
	if id2 != "cd1af05fe6923d6d" {
		t.Error(id2)
	}
}

func TestDatastoreSpanAddress(t *testing.T) {
	if s := datastoreSpanAddress("host", "portPathOrID"); s != "host:portPathOrID" {
		t.Error(s)
	}
	if s := datastoreSpanAddress("host", ""); s != "host" {
		t.Error(s)
	}
	if s := datastoreSpanAddress("", ""); s != "" {
		t.Error(s)
	}
}

func TestSegmentBasic(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &txnData{}
	thread := &tracingThread{}

	t1 := startSegment(txndata, thread, start.Add(1*time.Second))
	t2 := startSegment(txndata, thread, start.Add(2*time.Second))
	endBasicSegment(txndata, thread, t2, start.Add(3*time.Second), "t2")
	endBasicSegment(txndata, thread, t1, start.Add(4*time.Second), "t1")
	t3 := startSegment(txndata, thread, start.Add(5*time.Second))
	t4 := startSegment(txndata, thread, start.Add(6*time.Second))
	endBasicSegment(txndata, thread, t3, start.Add(7*time.Second), "t3")
	endBasicSegment(txndata, thread, t4, start.Add(8*time.Second), "out-of-order")
	t5 := startSegment(txndata, thread, start.Add(9*time.Second))
	endBasicSegment(txndata, thread, t5, start.Add(10*time.Second), "t1")

	metrics := newMetricTable(100, time.Now())
	txndata.FinalName = "WebTransaction/Go/zip"
	txndata.IsWeb = true
	mergeBreakdownMetrics(txndata, metrics)
	expectMetrics(t, metrics, []internal.WantMetric{
		{Name: "Custom/t1", Scope: "", Forced: false, Data: []float64{2, 4, 3, 1, 3, 10}},
		{Name: "Custom/t2", Scope: "", Forced: false, Data: []float64{1, 1, 1, 1, 1, 1}},
		{Name: "Custom/t3", Scope: "", Forced: false, Data: []float64{1, 2, 2, 2, 2, 4}},
		{Name: "Custom/t1", Scope: txndata.FinalName, Forced: false, Data: []float64{2, 4, 3, 1, 3, 10}},
		{Name: "Custom/t2", Scope: txndata.FinalName, Forced: false, Data: []float64{1, 1, 1, 1, 1, 1}},
		{Name: "Custom/t3", Scope: txndata.FinalName, Forced: false, Data: []float64{1, 2, 2, 2, 2, 4}},
	})
}

func parseURL(raw string) *url.URL {
	u, _ := url.Parse(raw)
	return u
}

func TestSegmentExternal(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &txnData{}
	thread := &tracingThread{}

	t1 := startSegment(txndata, thread, start.Add(1*time.Second))
	t2 := startSegment(txndata, thread, start.Add(2*time.Second))
	endExternalSegment(endExternalParams{
		TxnData: txndata,
		Thread:  thread,
		Start:   t2,
		Now:     start.Add(3 * time.Second),
		Logger:  logger.ShimLogger{},
	})
	endExternalSegment(endExternalParams{
		TxnData: txndata,
		Thread:  thread,
		Start:   t1,
		Now:     start.Add(4 * time.Second),
		URL:     parseURL("http://f1.com"),
		Host:    "f1",
		Logger:  logger.ShimLogger{},
	})
	t3 := startSegment(txndata, thread, start.Add(5*time.Second))
	endExternalSegment(endExternalParams{
		TxnData: txndata,
		Thread:  thread,
		Start:   t3,
		Now:     start.Add(6 * time.Second),
		URL:     parseURL("http://f1.com"),
		Host:    "f1",
		Logger:  logger.ShimLogger{},
	})
	t4 := startSegment(txndata, thread, start.Add(7*time.Second))
	t4.Stamp++
	endExternalSegment(endExternalParams{
		TxnData: txndata,
		Thread:  thread,
		Start:   t4,
		Now:     start.Add(8 * time.Second),
		URL:     parseURL("http://invalid-token.com"),
		Host:    "invalid-token.com",
		Logger:  logger.ShimLogger{},
	})
	if txndata.externalCallCount != 3 {
		t.Error(txndata.externalCallCount)
	}
	if txndata.externalDuration != 5*time.Second {
		t.Error(txndata.externalDuration)
	}
	metrics := newMetricTable(100, time.Now())
	txndata.FinalName = "WebTransaction/Go/zip"
	txndata.IsWeb = true
	mergeBreakdownMetrics(txndata, metrics)
	expectMetrics(t, metrics, []internal.WantMetric{
		{Name: "External/all", Scope: "", Forced: true, Data: []float64{3, 5, 4, 1, 3, 11}},
		{Name: "External/allWeb", Scope: "", Forced: true, Data: []float64{3, 5, 4, 1, 3, 11}},
		{Name: "External/f1/all", Scope: "", Forced: false, Data: []float64{2, 4, 3, 1, 3, 10}},
		{Name: "External/unknown/all", Scope: "", Forced: false, Data: []float64{1, 1, 1, 1, 1, 1}},
		{Name: "External/f1/http", Scope: txndata.FinalName, Forced: false, Data: []float64{2, 4, 3, 1, 3, 10}},
		{Name: "External/unknown/http", Scope: txndata.FinalName, Forced: false, Data: []float64{1, 1, 1, 1, 1, 1}},
	})

	metrics = newMetricTable(100, time.Now())
	txndata.FinalName = "OtherTransaction/Go/zip"
	txndata.IsWeb = false
	mergeBreakdownMetrics(txndata, metrics)
	expectMetrics(t, metrics, []internal.WantMetric{
		{Name: "External/all", Scope: "", Forced: true, Data: []float64{3, 5, 4, 1, 3, 11}},
		{Name: "External/allOther", Scope: "", Forced: true, Data: []float64{3, 5, 4, 1, 3, 11}},
		{Name: "External/f1/all", Scope: "", Forced: false, Data: []float64{2, 4, 3, 1, 3, 10}},
		{Name: "External/unknown/all", Scope: "", Forced: false, Data: []float64{1, 1, 1, 1, 1, 1}},
		{Name: "External/f1/http", Scope: txndata.FinalName, Forced: false, Data: []float64{2, 4, 3, 1, 3, 10}},
		{Name: "External/unknown/http", Scope: txndata.FinalName, Forced: false, Data: []float64{1, 1, 1, 1, 1, 1}},
	})
}

func TestSegmentDatastore(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &txnData{}
	thread := &tracingThread{}

	t1 := startSegment(txndata, thread, start.Add(1*time.Second))
	t2 := startSegment(txndata, thread, start.Add(2*time.Second))
	endDatastoreSegment(endDatastoreParams{
		TxnData:    txndata,
		Thread:     thread,
		Start:      t2,
		Now:        start.Add(3 * time.Second),
		Product:    "MySQL",
		Operation:  "SELECT",
		Collection: "my_table",
	})
	endDatastoreSegment(endDatastoreParams{
		TxnData:   txndata,
		Thread:    thread,
		Start:     t1,
		Now:       start.Add(4 * time.Second),
		Product:   "MySQL",
		Operation: "SELECT",
		// missing collection
	})
	t3 := startSegment(txndata, thread, start.Add(5*time.Second))
	endDatastoreSegment(endDatastoreParams{
		TxnData:   txndata,
		Thread:    thread,
		Start:     t3,
		Now:       start.Add(6 * time.Second),
		Product:   "MySQL",
		Operation: "SELECT",
		// missing collection
	})
	t4 := startSegment(txndata, thread, start.Add(7*time.Second))
	t4.Stamp++
	endDatastoreSegment(endDatastoreParams{
		TxnData:   txndata,
		Thread:    thread,
		Start:     t4,
		Now:       start.Add(8 * time.Second),
		Product:   "MySQL",
		Operation: "invalid-token",
	})
	t5 := startSegment(txndata, thread, start.Add(9*time.Second))
	endDatastoreSegment(endDatastoreParams{
		TxnData: txndata,
		Thread:  thread,
		Start:   t5,
		Now:     start.Add(10 * time.Second),
		// missing datastore, collection, and operation
	})

	if txndata.datastoreCallCount != 4 {
		t.Error(txndata.datastoreCallCount)
	}
	if txndata.datastoreDuration != 6*time.Second {
		t.Error(txndata.datastoreDuration)
	}
	metrics := newMetricTable(100, time.Now())
	txndata.FinalName = "WebTransaction/Go/zip"
	txndata.IsWeb = true
	mergeBreakdownMetrics(txndata, metrics)
	expectMetrics(t, metrics, []internal.WantMetric{
		{Name: "Datastore/all", Scope: "", Forced: true, Data: []float64{4, 6, 5, 1, 3, 12}},
		{Name: "Datastore/allWeb", Scope: "", Forced: true, Data: []float64{4, 6, 5, 1, 3, 12}},
		{Name: "Datastore/MySQL/all", Scope: "", Forced: true, Data: []float64{3, 5, 4, 1, 3, 11}},
		{Name: "Datastore/MySQL/allWeb", Scope: "", Forced: true, Data: []float64{3, 5, 4, 1, 3, 11}},
		{Name: "Datastore/Unknown/all", Scope: "", Forced: true, Data: []float64{1, 1, 1, 1, 1, 1}},
		{Name: "Datastore/Unknown/allWeb", Scope: "", Forced: true, Data: []float64{1, 1, 1, 1, 1, 1}},
		{Name: "Datastore/operation/MySQL/SELECT", Scope: "", Forced: false, Data: []float64{3, 5, 4, 1, 3, 11}},
		{Name: "Datastore/operation/MySQL/SELECT", Scope: txndata.FinalName, Forced: false, Data: []float64{2, 4, 3, 1, 3, 10}},
		{Name: "Datastore/operation/Unknown/other", Scope: "", Forced: false, Data: []float64{1, 1, 1, 1, 1, 1}},
		{Name: "Datastore/operation/Unknown/other", Scope: txndata.FinalName, Forced: false, Data: []float64{1, 1, 1, 1, 1, 1}},
		{Name: "Datastore/statement/MySQL/my_table/SELECT", Scope: "", Forced: false, Data: []float64{1, 1, 1, 1, 1, 1}},
		{Name: "Datastore/statement/MySQL/my_table/SELECT", Scope: txndata.FinalName, Forced: false, Data: []float64{1, 1, 1, 1, 1, 1}},
	})

	metrics = newMetricTable(100, time.Now())
	txndata.FinalName = "OtherTransaction/Go/zip"
	txndata.IsWeb = false
	mergeBreakdownMetrics(txndata, metrics)
	expectMetrics(t, metrics, []internal.WantMetric{
		{Name: "Datastore/all", Scope: "", Forced: true, Data: []float64{4, 6, 5, 1, 3, 12}},
		{Name: "Datastore/allOther", Scope: "", Forced: true, Data: []float64{4, 6, 5, 1, 3, 12}},
		{Name: "Datastore/MySQL/all", Scope: "", Forced: true, Data: []float64{3, 5, 4, 1, 3, 11}},
		{Name: "Datastore/MySQL/allOther", Scope: "", Forced: true, Data: []float64{3, 5, 4, 1, 3, 11}},
		{Name: "Datastore/Unknown/all", Scope: "", Forced: true, Data: []float64{1, 1, 1, 1, 1, 1}},
		{Name: "Datastore/Unknown/allOther", Scope: "", Forced: true, Data: []float64{1, 1, 1, 1, 1, 1}},
		{Name: "Datastore/operation/MySQL/SELECT", Scope: "", Forced: false, Data: []float64{3, 5, 4, 1, 3, 11}},
		{Name: "Datastore/operation/MySQL/SELECT", Scope: txndata.FinalName, Forced: false, Data: []float64{2, 4, 3, 1, 3, 10}},
		{Name: "Datastore/operation/Unknown/other", Scope: "", Forced: false, Data: []float64{1, 1, 1, 1, 1, 1}},
		{Name: "Datastore/operation/Unknown/other", Scope: txndata.FinalName, Forced: false, Data: []float64{1, 1, 1, 1, 1, 1}},
		{Name: "Datastore/statement/MySQL/my_table/SELECT", Scope: "", Forced: false, Data: []float64{1, 1, 1, 1, 1, 1}},
		{Name: "Datastore/statement/MySQL/my_table/SELECT", Scope: txndata.FinalName, Forced: false, Data: []float64{1, 1, 1, 1, 1, 1}},
	})
}

func TestDatastoreInstancesCrossAgent(t *testing.T) {
	var testcases []struct {
		Name           string `json:"name"`
		SystemHostname string `json:"system_hostname"`
		DBHostname     string `json:"db_hostname"`
		Product        string `json:"product"`
		Port           int    `json:"port"`
		Socket         string `json:"unix_socket"`
		DatabasePath   string `json:"database_path"`
		ExpectedMetric string `json:"expected_instance_metric"`
	}

	err := crossagent.ReadJSON("datastores/datastore_instances.json", &testcases)
	if err != nil {
		t.Fatal(err)
	}

	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)

	for _, tc := range testcases {
		portPathOrID := ""
		if 0 != tc.Port {
			portPathOrID = strconv.Itoa(tc.Port)
		} else if "" != tc.Socket {
			portPathOrID = tc.Socket
		} else if "" != tc.DatabasePath {
			portPathOrID = tc.DatabasePath
			// These tests makes weird assumptions.
			tc.DBHostname = "localhost"
		}

		txndata := &txnData{}
		thread := &tracingThread{}

		host := "this-hostname"
		s := startSegment(txndata, thread, start)
		endDatastoreSegment(endDatastoreParams{
			Thread:       thread,
			TxnData:      txndata,
			Start:        s,
			Now:          start.Add(1 * time.Second),
			Product:      tc.Product,
			Operation:    "SELECT",
			Collection:   "my_table",
			PortPathOrID: portPathOrID,
			Host:         tc.DBHostname,
			ThisHost:     host,
		})

		expect := strings.Replace(tc.ExpectedMetric,
			tc.SystemHostname, host, -1)

		metrics := newMetricTable(100, time.Now())
		txndata.FinalName = "OtherTransaction/Go/zip"
		txndata.IsWeb = false
		mergeBreakdownMetrics(txndata, metrics)
		data := []float64{1, 1, 1, 1, 1, 1}
		expectMetrics(extendValidator(t, tc.Name), metrics, []internal.WantMetric{
			{Name: "Datastore/all", Scope: "", Forced: true, Data: data},
			{Name: "Datastore/allOther", Scope: "", Forced: true, Data: data},
			{Name: "Datastore/" + tc.Product + "/all", Scope: "", Forced: true, Data: data},
			{Name: "Datastore/" + tc.Product + "/allOther", Scope: "", Forced: true, Data: data},
			{Name: "Datastore/operation/" + tc.Product + "/SELECT", Scope: "", Forced: false, Data: data},
			{Name: "Datastore/statement/" + tc.Product + "/my_table/SELECT", Scope: "", Forced: false, Data: data},
			{Name: "Datastore/statement/" + tc.Product + "/my_table/SELECT", Scope: txndata.FinalName, Forced: false, Data: data},
			{Name: expect, Scope: "", Forced: false, Data: data},
		})
	}
}

func TestGenericSpanEventCreation(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &txnData{
		TraceIDGenerator:        internal.NewTraceIDGenerator(12345),
		ShouldCollectSpanEvents: trueFunc,
		ShouldCreateSpanGUID:    trueFunc,
	}
	thread := &tracingThread{}

	t1 := startSegment(txndata, thread, start.Add(1*time.Second))
	endBasicSegment(txndata, thread, t1, start.Add(3*time.Second), "t1")

	// Since a basic segment has just ended, there should be exactly one generic span event in txndata.SpanEvents[]
	if 1 != len(txndata.SpanEvents) {
		t.Error(txndata.SpanEvents)
	}
	if txndata.SpanEvents[0].Category != spanCategoryGeneric {
		t.Error(txndata.SpanEvents[0].Category)
	}
}

func TestSpanEventNotCollected(t *testing.T) {
	// Test the situation where ShouldCollectSpanEvents is populated but returns
	// false.
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &txnData{
		TraceIDGenerator:        internal.NewTraceIDGenerator(12345),
		ShouldCollectSpanEvents: falseFunc,
		ShouldCreateSpanGUID:    falseFunc,
	}
	thread := &tracingThread{}

	t1 := startSegment(txndata, thread, start.Add(1*time.Second))
	endBasicSegment(txndata, thread, t1, start.Add(3*time.Second), "t1")

	if 0 != len(txndata.SpanEvents) {
		t.Error(txndata.SpanEvents)
	}
}

func TestDatastoreSpanEventCreation(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &txnData{
		TraceIDGenerator: internal.NewTraceIDGenerator(12345),
	}
	thread := &tracingThread{}

	// Enable that which is necessary to generate span events when segments are ended.
	txndata.ShouldCollectSpanEvents = trueFunc
	txndata.ShouldCreateSpanGUID = trueFunc

	t1 := startSegment(txndata, thread, start.Add(1*time.Second))
	endDatastoreSegment(endDatastoreParams{
		TxnData:    txndata,
		Thread:     thread,
		Start:      t1,
		Now:        start.Add(3 * time.Second),
		Product:    "MySQL",
		Operation:  "SELECT",
		Collection: "my_table",
	})

	// Since a datastore segment has just ended, there should be exactly one datastore span event in txndata.SpanEvents[]
	if 1 != len(txndata.SpanEvents) {
		t.Error(txndata.SpanEvents)
	}
	if txndata.SpanEvents[0].Category != spanCategoryDatastore {
		t.Error(txndata.SpanEvents[0].Category)
	}
}

func TestHTTPSpanEventCreation(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &txnData{
		TraceIDGenerator: internal.NewTraceIDGenerator(12345),
	}
	thread := &tracingThread{}

	// Enable that which is necessary to generate span events when segments are ended.
	txndata.ShouldCollectSpanEvents = trueFunc
	txndata.ShouldCreateSpanGUID = trueFunc

	t1 := startSegment(txndata, thread, start.Add(1*time.Second))
	endExternalSegment(endExternalParams{
		TxnData: txndata,
		Thread:  thread,
		Start:   t1,
		Now:     start.Add(3 * time.Second),
		URL:     nil,
		Logger:  logger.ShimLogger{},
	})

	// Since an external segment has just ended, there should be exactly one HTTP span event in txndata.SpanEvents[]
	if 1 != len(txndata.SpanEvents) {
		t.Error(txndata.SpanEvents)
	}
	if txndata.SpanEvents[0].Category != spanCategoryHTTP {
		t.Error(txndata.SpanEvents[0].Category)
	}
}

func TestExternalSegmentCAT(t *testing.T) {
	// Test that when the reading the response CAT headers fails, an external
	// segment is still created.
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &txnData{
		TraceIDGenerator: internal.NewTraceIDGenerator(12345),
	}
	txndata.CrossProcess.Enabled = true
	thread := &tracingThread{}

	resp := &http.Response{Header: http.Header{}}
	resp.Header.Add(cat.NewRelicAppDataName, "bad header value")

	t1 := startSegment(txndata, thread, start.Add(1*time.Second))
	err := endExternalSegment(endExternalParams{
		TxnData: txndata,
		Thread:  thread,
		Start:   t1,
		Now:     start.Add(4 * time.Second),
		URL:     parseURL("http://f1.com"),
		Logger:  logger.ShimLogger{},
	})

	if nil != err {
		t.Error("endExternalSegment returned an err:", err)
	}
	if txndata.externalCallCount != 1 {
		t.Error(txndata.externalCallCount)
	}
	if txndata.externalDuration != 3*time.Second {
		t.Error(txndata.externalDuration)
	}

	metrics := newMetricTable(100, time.Now())
	txndata.FinalName = "OtherTransaction/Go/zip"
	txndata.IsWeb = false
	mergeBreakdownMetrics(txndata, metrics)
	expectMetrics(t, metrics, []internal.WantMetric{
		{Name: "External/all", Scope: "", Forced: true, Data: []float64{1, 3, 3, 3, 3, 9}},
		{Name: "External/allOther", Scope: "", Forced: true, Data: []float64{1, 3, 3, 3, 3, 9}},
		{Name: "External/f1.com/all", Scope: "", Forced: false, Data: []float64{1, 3, 3, 3, 3, 9}},
		{Name: "External/f1.com/http", Scope: txndata.FinalName, Forced: false, Data: []float64{1, 3, 3, 3, 3, 9}},
	})
}

func TestEndMessageSegment(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &txnData{
		TraceIDGenerator: internal.NewTraceIDGenerator(12345),
	}
	txndata.CrossProcess.Enabled = true
	thread := &tracingThread{}

	seg1 := startSegment(txndata, thread, start.Add(1*time.Second))
	seg2 := startSegment(txndata, thread, start.Add(2*time.Second))
	endMessageSegment(endMessageParams{
		TxnData:         txndata,
		Thread:          thread,
		Start:           seg1,
		Now:             start.Add(3 * time.Second),
		Logger:          nil,
		DestinationName: "MyTopic",
		Library:         "Kafka",
		DestinationType: "Topic",
	})
	endMessageSegment(endMessageParams{
		TxnData:         txndata,
		Thread:          thread,
		Start:           seg2,
		Now:             start.Add(4 * time.Second),
		Logger:          nil,
		DestinationName: "MyOtherTopic",
		Library:         "Kafka",
		DestinationType: "Topic",
	})

	metrics := newMetricTable(100, time.Now())
	txndata.FinalName = "WebTransaction/Go/zip"
	txndata.IsWeb = true
	mergeBreakdownMetrics(txndata, metrics)
	expectMetrics(t, metrics, []internal.WantMetric{
		{Name: "MessageBroker/Kafka/Topic/Produce/Named/MyTopic", Scope: "WebTransaction/Go/zip", Forced: false, Data: []float64{1, 2, 2, 2, 2, 4}},
		{Name: "MessageBroker/Kafka/Topic/Produce/Named/MyTopic", Scope: "", Forced: false, Data: []float64{1, 2, 2, 2, 2, 4}},
	})
}

func TestBetterCAT_SetTraceAndTxnIDs(t *testing.T) {
	cases := map[string]string{
		"12345678901234567890123456789012": "1234567890123456",
		"12345678901234567890":             "1234567890123456",
		"1234567890123456":                 "1234567890123456",
		"":                                 "",
		"123456":                           "123456",
	}
	for k, v := range cases {
		bc := betterCAT{}
		bc.SetTraceAndTxnIDs(k)
		if bc.TxnID != v {
			t.Errorf("Unexpected txn ID - for key %s got %s, but expected %s", k, bc.TxnID, v)
		}
	}
}
