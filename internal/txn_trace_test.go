// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/newrelic/go-agent/internal/cat"
	"github.com/newrelic/go-agent/internal/logger"
)

func TestTxnTrace(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &TxnData{}
	thread := &Thread{}
	txndata.TxnTrace.Enabled = true
	txndata.TxnTrace.StackTraceThreshold = 1 * time.Hour
	txndata.TxnTrace.SegmentThreshold = 0

	t1 := StartSegment(txndata, thread, start.Add(1*time.Second))
	t2 := StartSegment(txndata, thread, start.Add(2*time.Second))
	qParams, err := vetQueryParameters(map[string]interface{}{"zip": 1})
	if nil != err {
		t.Error("error creating query params", err)
	}
	EndDatastoreSegment(EndDatastoreParams{
		TxnData:            txndata,
		Thread:             thread,
		Start:              t2,
		Now:                start.Add(3 * time.Second),
		Product:            "MySQL",
		Operation:          "SELECT",
		Collection:         "my_table",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
		QueryParameters:    qParams,
		Database:           "my_db",
		Host:               "db-server-1",
		PortPathOrID:       "3306",
	})
	t3 := StartSegment(txndata, thread, start.Add(4*time.Second))
	EndExternalSegment(EndExternalParams{
		TxnData: txndata,
		Thread:  thread,
		Start:   t3,
		Now:     start.Add(5 * time.Second),
		URL:     parseURL("http://example.com/zip/zap?secret=shhh"),
		Logger:  logger.ShimLogger{},
	})
	EndBasicSegment(txndata, thread, t1, start.Add(6*time.Second), "t1")
	t4 := StartSegment(txndata, thread, start.Add(7*time.Second))
	t5 := StartSegment(txndata, thread, start.Add(8*time.Second))
	t6 := StartSegment(txndata, thread, start.Add(9*time.Second))
	EndBasicSegment(txndata, thread, t6, start.Add(10*time.Second), "t6")
	EndBasicSegment(txndata, thread, t5, start.Add(11*time.Second), "t5")
	t7 := StartSegment(txndata, thread, start.Add(12*time.Second))
	EndDatastoreSegment(EndDatastoreParams{
		TxnData:   txndata,
		Thread:    thread,
		Start:     t7,
		Now:       start.Add(13 * time.Second),
		Product:   "MySQL",
		Operation: "SELECT",
		// no collection
	})
	t8 := StartSegment(txndata, thread, start.Add(14*time.Second))
	EndExternalSegment(EndExternalParams{
		TxnData: txndata,
		Thread:  thread,
		Start:   t8,
		Now:     start.Add(15 * time.Second),
		URL:     nil,
		Logger:  logger.ShimLogger{},
	})
	EndBasicSegment(txndata, thread, t4, start.Add(16*time.Second), "t4")

	t9 := StartSegment(txndata, thread, start.Add(17*time.Second))
	EndMessageSegment(EndMessageParams{
		TxnData:         txndata,
		Thread:          thread,
		Start:           t9,
		Now:             start.Add(18 * time.Second),
		Logger:          nil,
		DestinationName: "MyTopic",
		Library:         "Kafka",
		DestinationType: "Topic",
	})

	acfg := CreateAttributeConfig(sampleAttributeConfigInput, true)
	attr := NewAttributes(acfg)
	attr.Agent.Add(attributeRequestURI, "/url", nil)
	AddUserAttribute(attr, "zap", 123, DestAll)

	ht := newHarvestTraces()
	ht.regular.addTxnTrace(&HarvestTrace{
		TxnEvent: TxnEvent{
			Start:     start,
			Duration:  20 * time.Second,
			TotalTime: 30 * time.Second,
			FinalName: "WebTransaction/Go/hello",
			Attrs:     attr,
			BetterCAT: BetterCAT{
				Enabled:  true,
				ID:       "txn-id",
				Priority: 0.5,
			},
		},
		Trace: txndata.TxnTrace,
	})

	ExpectTxnTraces(t, ht, []WantTxnTrace{{
		MetricName:      "WebTransaction/Go/hello",
		UserAttributes:  map[string]interface{}{"zap": 123},
		AgentAttributes: map[string]interface{}{"request.uri": "/url"},
		Intrinsics: map[string]interface{}{
			"guid":      "txn-id",
			"traceId":   "txn-id",
			"priority":  0.500000,
			"sampled":   false,
			"totalTime": 30,
		},
		Root: WantTraceSegment{
			SegmentName:         "ROOT",
			RelativeStartMillis: 0,
			RelativeStopMillis:  20000,
			Attributes:          map[string]interface{}{},
			Children: []WantTraceSegment{{
				SegmentName:         "WebTransaction/Go/hello",
				RelativeStartMillis: 0,
				RelativeStopMillis:  20000,
				Attributes:          map[string]interface{}{"exclusive_duration_millis": 20000},
				Children: []WantTraceSegment{
					{
						SegmentName:         "Custom/t1",
						RelativeStartMillis: 1000,
						RelativeStopMillis:  6000,
						Attributes:          map[string]interface{}{},
						Children: []WantTraceSegment{
							{
								SegmentName:         "Datastore/statement/MySQL/my_table/SELECT",
								RelativeStartMillis: 2000,
								RelativeStopMillis:  3000,
								Attributes: map[string]interface{}{
									"db.instance":      "my_db",
									"peer.hostname":    "db-server-1",
									"peer.address":     "db-server-1:3306",
									"db.statement":     "INSERT INTO users (name, age) VALUES ($1, $2)",
									"query_parameters": "map[zip:1]",
								},
								Children: []WantTraceSegment{},
							},
							{
								SegmentName:         "External/example.com/http",
								RelativeStartMillis: 4000,
								RelativeStopMillis:  5000,
								Attributes: map[string]interface{}{
									"http.url": "http://example.com/zip/zap",
								},
								Children: []WantTraceSegment{},
							},
						},
					},
					{
						SegmentName:         "Custom/t4",
						RelativeStartMillis: 7000,
						RelativeStopMillis:  16000,
						Attributes:          map[string]interface{}{},
						Children: []WantTraceSegment{
							{
								SegmentName:         "Custom/t5",
								RelativeStartMillis: 8000,
								RelativeStopMillis:  11000,
								Attributes:          map[string]interface{}{},
								Children: []WantTraceSegment{
									{
										SegmentName:         "Custom/t6",
										RelativeStartMillis: 9000,
										RelativeStopMillis:  10000,
										Attributes:          map[string]interface{}{},
										Children:            []WantTraceSegment{},
									},
								},
							},
							{
								SegmentName:         "Datastore/operation/MySQL/SELECT",
								RelativeStartMillis: 12000,
								RelativeStopMillis:  13000,
								Attributes: map[string]interface{}{
									"db.statement": "'SELECT' on 'unknown' using 'MySQL'",
								},
								Children: []WantTraceSegment{},
							},
							{
								SegmentName:         "External/unknown/http",
								RelativeStartMillis: 14000,
								RelativeStopMillis:  15000,
								Attributes:          map[string]interface{}{},
								Children:            []WantTraceSegment{},
							},
						},
					},
					{
						SegmentName:         "MessageBroker/Kafka/Topic/Produce/Named/MyTopic",
						RelativeStartMillis: 17000,
						RelativeStopMillis:  18000,
						Attributes:          map[string]interface{}{},
						Children:            []WantTraceSegment{},
					},
				},
			}},
		},
	}})
}

func TestTxnTraceNoNodes(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &TxnData{}
	txndata.TxnTrace.Enabled = true
	txndata.TxnTrace.StackTraceThreshold = 1 * time.Hour
	txndata.TxnTrace.SegmentThreshold = 0

	ht := newHarvestTraces()
	ht.regular.addTxnTrace(&HarvestTrace{
		TxnEvent: TxnEvent{
			Start:     start,
			Duration:  20 * time.Second,
			TotalTime: 30 * time.Second,
			FinalName: "WebTransaction/Go/hello",
			Attrs:     nil,
			BetterCAT: BetterCAT{
				Enabled:  true,
				ID:       "txn-id",
				Priority: 0.5,
			},
		},
		Trace: txndata.TxnTrace,
	})

	ExpectTxnTraces(t, ht, []WantTxnTrace{{
		MetricName:      "WebTransaction/Go/hello",
		UserAttributes:  map[string]interface{}{},
		AgentAttributes: map[string]interface{}{},
		Intrinsics: map[string]interface{}{
			"guid":      "txn-id",
			"traceId":   "txn-id",
			"priority":  0.500000,
			"sampled":   false,
			"totalTime": 30,
		},
		Root: WantTraceSegment{
			SegmentName:         "ROOT",
			RelativeStartMillis: 0,
			RelativeStopMillis:  20000,
			Attributes:          map[string]interface{}{},
			Children: []WantTraceSegment{{
				SegmentName:         "WebTransaction/Go/hello",
				RelativeStartMillis: 0,
				RelativeStopMillis:  20000,
				Attributes:          map[string]interface{}{"exclusive_duration_millis": 20000},
				Children:            []WantTraceSegment{},
			}},
		},
	}})
}

func TestTxnTraceAsync(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &TxnData{
		TraceIDGenerator: NewTraceIDGenerator(12345),
	}
	thread1 := &Thread{}
	txndata.TxnTrace.Enabled = true
	txndata.TxnTrace.StackTraceThreshold = 1 * time.Hour
	txndata.TxnTrace.SegmentThreshold = 0
	txndata.BetterCAT.Sampled = true
	txndata.SpanEventsEnabled = true
	txndata.LazilyCalculateSampled = func() bool { return true }

	t1s1 := StartSegment(txndata, thread1, start.Add(1*time.Second))
	t1s2 := StartSegment(txndata, thread1, start.Add(2*time.Second))
	thread2 := NewThread(txndata)
	t2s1 := StartSegment(txndata, thread2, start.Add(3*time.Second))
	EndBasicSegment(txndata, thread1, t1s2, start.Add(4*time.Second), "thread1.segment2")
	EndBasicSegment(txndata, thread2, t2s1, start.Add(5*time.Second), "thread2.segment1")
	thread3 := NewThread(txndata)
	t3s1 := StartSegment(txndata, thread3, start.Add(6*time.Second))
	t3s2 := StartSegment(txndata, thread3, start.Add(7*time.Second))
	EndBasicSegment(txndata, thread1, t1s1, start.Add(8*time.Second), "thread1.segment1")
	EndBasicSegment(txndata, thread3, t3s2, start.Add(9*time.Second), "thread3.segment2")
	EndBasicSegment(txndata, thread3, t3s1, start.Add(10*time.Second), "thread3.segment1")

	if tt := thread1.TotalTime(); tt != 7*time.Second {
		t.Error(tt)
	}
	if tt := thread2.TotalTime(); tt != 2*time.Second {
		t.Error(tt)
	}
	if tt := thread3.TotalTime(); tt != 4*time.Second {
		t.Error(tt)
	}

	if len(txndata.spanEvents) != 5 {
		t.Fatal(txndata.spanEvents)
	}
	for _, e := range txndata.spanEvents {
		if e.GUID == "" || e.ParentID == "" {
			t.Error(e.GUID, e.ParentID)
		}
	}
	spanEventT1S2 := txndata.spanEvents[0]
	spanEventT2S1 := txndata.spanEvents[1]
	spanEventT1S1 := txndata.spanEvents[2]
	spanEventT3S2 := txndata.spanEvents[3]
	spanEventT3S1 := txndata.spanEvents[4]

	if txndata.rootSpanID == "" {
		t.Error(txndata.rootSpanID)
	}
	if spanEventT1S1.ParentID != txndata.rootSpanID {
		t.Error(spanEventT1S1.ParentID, txndata.rootSpanID)
	}
	if spanEventT1S2.ParentID != spanEventT1S1.GUID {
		t.Error(spanEventT1S2.ParentID, spanEventT1S1.GUID)
	}
	if spanEventT2S1.ParentID != txndata.rootSpanID {
		t.Error(spanEventT2S1.ParentID, txndata.rootSpanID)
	}
	if spanEventT3S1.ParentID != txndata.rootSpanID {
		t.Error(spanEventT3S1.ParentID, txndata.rootSpanID)
	}
	if spanEventT3S2.ParentID != spanEventT3S1.GUID {
		t.Error(spanEventT3S2.ParentID, spanEventT3S1.GUID)
	}

	ht := newHarvestTraces()
	ht.regular.addTxnTrace(&HarvestTrace{
		TxnEvent: TxnEvent{
			Start:     start,
			Duration:  20 * time.Second,
			TotalTime: 30 * time.Second,
			FinalName: "WebTransaction/Go/hello",
			Attrs:     nil,
			BetterCAT: BetterCAT{
				Enabled:  true,
				ID:       "txn-id",
				Priority: 0.5,
			},
		},
		Trace: txndata.TxnTrace,
	})

	ExpectTxnTraces(t, ht, []WantTxnTrace{{
		MetricName:      "WebTransaction/Go/hello",
		UserAttributes:  map[string]interface{}{},
		AgentAttributes: map[string]interface{}{},
		Intrinsics: map[string]interface{}{
			"totalTime": 30,
			"guid":      "txn-id",
			"traceId":   "txn-id",
			"priority":  0.500000,
			"sampled":   false,
		},
		Root: WantTraceSegment{
			SegmentName:         "ROOT",
			RelativeStartMillis: 0,
			RelativeStopMillis:  20000,
			Attributes:          map[string]interface{}{},
			Children: []WantTraceSegment{{
				SegmentName:         "WebTransaction/Go/hello",
				RelativeStartMillis: 0,
				RelativeStopMillis:  20000,
				Attributes:          map[string]interface{}{"exclusive_duration_millis": 20000},
				Children: []WantTraceSegment{
					{
						SegmentName:         "Custom/thread1.segment1",
						RelativeStartMillis: 1000,
						RelativeStopMillis:  8000,
						Attributes:          map[string]interface{}{},
						Children: []WantTraceSegment{
							{
								SegmentName:         "Custom/thread1.segment2",
								RelativeStartMillis: 2000,
								RelativeStopMillis:  4000,
								Attributes:          map[string]interface{}{},
								Children:            []WantTraceSegment{},
							},
						},
					},
					{
						SegmentName:         "Custom/thread2.segment1",
						RelativeStartMillis: 3000,
						RelativeStopMillis:  5000,
						Attributes:          map[string]interface{}{},
						Children:            []WantTraceSegment{},
					},
					{
						SegmentName:         "Custom/thread3.segment1",
						RelativeStartMillis: 6000,
						RelativeStopMillis:  10000,
						Attributes:          map[string]interface{}{},
						Children: []WantTraceSegment{
							{
								SegmentName:         "Custom/thread3.segment2",
								RelativeStartMillis: 7000,
								RelativeStopMillis:  9000,
								Attributes:          map[string]interface{}{},
								Children:            []WantTraceSegment{},
							},
						},
					},
				},
			}},
		},
	}})
}

func TestTxnTraceOldCAT(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &TxnData{}
	thread := &Thread{}
	txndata.TxnTrace.Enabled = true
	txndata.TxnTrace.StackTraceThreshold = 1 * time.Hour
	txndata.TxnTrace.SegmentThreshold = 0

	txndata.CrossProcess.Init(true, false, replyAccountOne)
	txndata.CrossProcess.GUID = "0123456789"
	appData, err := txndata.CrossProcess.CreateAppData("WebTransaction/Go/otherService", 2*time.Second, 3*time.Second, 123)
	if nil != err {
		t.Fatal(err)
	}
	resp := &http.Response{
		Header: AppDataToHTTPHeader(appData),
	}
	t3 := StartSegment(txndata, thread, start.Add(4*time.Second))
	EndExternalSegment(EndExternalParams{
		TxnData:  txndata,
		Thread:   thread,
		Start:    t3,
		Now:      start.Add(5 * time.Second),
		URL:      parseURL("http://example.com/zip/zap?secret=shhh"),
		Response: resp,
		Logger:   logger.ShimLogger{},
	})

	acfg := CreateAttributeConfig(sampleAttributeConfigInput, true)
	attr := NewAttributes(acfg)
	attr.Agent.Add(attributeRequestURI, "/url", nil)
	AddUserAttribute(attr, "zap", 123, DestAll)

	ht := newHarvestTraces()
	ht.regular.addTxnTrace(&HarvestTrace{
		TxnEvent: TxnEvent{
			Start:     start,
			Duration:  20 * time.Second,
			TotalTime: 30 * time.Second,
			FinalName: "WebTransaction/Go/hello",
			Attrs:     attr,
		},
		Trace: txndata.TxnTrace,
	})

	ExpectTxnTraces(t, ht, []WantTxnTrace{{
		MetricName:      "WebTransaction/Go/hello",
		UserAttributes:  map[string]interface{}{"zap": 123},
		AgentAttributes: map[string]interface{}{"request.uri": "/url"},
		Intrinsics:      map[string]interface{}{"totalTime": 30},
		Root: WantTraceSegment{
			SegmentName:         "ROOT",
			RelativeStartMillis: 0,
			RelativeStopMillis:  20000,
			Attributes:          map[string]interface{}{},
			Children: []WantTraceSegment{{
				SegmentName:         "WebTransaction/Go/hello",
				RelativeStartMillis: 0,
				RelativeStopMillis:  20000,
				Attributes:          map[string]interface{}{"exclusive_duration_millis": 20000},
				Children: []WantTraceSegment{
					{
						SegmentName:         "ExternalTransaction/example.com/1#1/WebTransaction/Go/otherService",
						RelativeStartMillis: 4000,
						RelativeStopMillis:  5000,
						Attributes: map[string]interface{}{
							"http.url":         "http://example.com/zip/zap",
							"transaction_guid": "0123456789",
						},
						Children: []WantTraceSegment{},
					},
				},
			}},
		},
	}})
}

func TestTxnTraceExcludeURI(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &TxnData{}
	tr.TxnTrace.Enabled = true
	tr.TxnTrace.StackTraceThreshold = 1 * time.Hour
	tr.TxnTrace.SegmentThreshold = 0

	c := sampleAttributeConfigInput
	c.TransactionTracer.Exclude = []string{"request.uri"}
	acfg := CreateAttributeConfig(c, true)
	attr := NewAttributes(acfg)
	attr.Agent.Add(attributeRequestURI, "/url", nil)

	ht := newHarvestTraces()
	ht.regular.addTxnTrace(&HarvestTrace{
		TxnEvent: TxnEvent{
			Start:     start,
			Duration:  20 * time.Second,
			FinalName: "WebTransaction/Go/hello",
			Attrs:     attr,
			BetterCAT: BetterCAT{
				Enabled:  true,
				ID:       "txn-id",
				Priority: 0.5,
			},
		},
		Trace: tr.TxnTrace,
	})

	ExpectTxnTraces(t, ht, []WantTxnTrace{{
		MetricName:      "WebTransaction/Go/hello",
		UserAttributes:  map[string]interface{}{},
		AgentAttributes: map[string]interface{}{},
		Intrinsics: map[string]interface{}{
			"totalTime": 0,
			"guid":      "txn-id",
			"traceId":   "txn-id",
			"priority":  0.500000,
			"sampled":   false,
		},
		Root: WantTraceSegment{
			SegmentName:         "ROOT",
			RelativeStartMillis: 0,
			RelativeStopMillis:  20000,
			Attributes:          map[string]interface{}{},
			Children: []WantTraceSegment{{
				SegmentName:         "WebTransaction/Go/hello",
				RelativeStartMillis: 0,
				RelativeStopMillis:  20000,
				Attributes:          map[string]interface{}{"exclusive_duration_millis": 20000},
				Children:            []WantTraceSegment{},
			}},
		},
	}})
}

func TestTxnTraceNoSegmentsNoAttributes(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &TxnData{}
	txndata.TxnTrace.Enabled = true
	txndata.TxnTrace.StackTraceThreshold = 1 * time.Hour
	txndata.TxnTrace.SegmentThreshold = 0

	acfg := CreateAttributeConfig(sampleAttributeConfigInput, true)
	attr := NewAttributes(acfg)

	ht := newHarvestTraces()
	ht.regular.addTxnTrace(&HarvestTrace{
		TxnEvent: TxnEvent{
			Start:     start,
			Duration:  20 * time.Second,
			TotalTime: 30 * time.Second,
			FinalName: "WebTransaction/Go/hello",
			Attrs:     attr,
			BetterCAT: BetterCAT{
				Enabled:  true,
				ID:       "txn-id",
				Priority: 0.5,
			},
		},
		Trace: txndata.TxnTrace,
	})

	ExpectTxnTraces(t, ht, []WantTxnTrace{{
		MetricName:      "WebTransaction/Go/hello",
		UserAttributes:  map[string]interface{}{},
		AgentAttributes: map[string]interface{}{},
		Intrinsics: map[string]interface{}{
			"totalTime": 30,
			"guid":      "txn-id",
			"traceId":   "txn-id",
			"priority":  0.500000,
			"sampled":   false,
		},
		Root: WantTraceSegment{
			SegmentName:         "ROOT",
			RelativeStartMillis: 0,
			RelativeStopMillis:  20000,
			Attributes:          map[string]interface{}{},
			Children: []WantTraceSegment{{
				SegmentName:         "WebTransaction/Go/hello",
				RelativeStartMillis: 0,
				RelativeStopMillis:  20000,
				Attributes:          map[string]interface{}{"exclusive_duration_millis": 20000},
				Children:            []WantTraceSegment{},
			}},
		},
	}})
}

func TestTxnTraceSlowestNodesSaved(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &TxnData{}
	thread := &Thread{}
	txndata.TxnTrace.Enabled = true
	txndata.TxnTrace.StackTraceThreshold = 1 * time.Hour
	txndata.TxnTrace.SegmentThreshold = 0
	txndata.TxnTrace.maxNodes = 5

	durations := []int{5, 4, 6, 3, 7, 2, 8, 1, 9}
	now := start
	for _, d := range durations {
		s := StartSegment(txndata, thread, now)
		now = now.Add(time.Duration(d) * time.Second)
		EndBasicSegment(txndata, thread, s, now, strconv.Itoa(d))
	}

	acfg := CreateAttributeConfig(sampleAttributeConfigInput, true)
	attr := NewAttributes(acfg)
	attr.Agent.Add(attributeRequestURI, "/url", nil)

	ht := newHarvestTraces()
	ht.regular.addTxnTrace(&HarvestTrace{
		TxnEvent: TxnEvent{
			Start:     start,
			Duration:  123 * time.Second,
			TotalTime: 200 * time.Second,
			FinalName: "WebTransaction/Go/hello",
			Attrs:     attr,
			BetterCAT: BetterCAT{
				Enabled:  true,
				ID:       "txn-id",
				Priority: 0.5,
			},
		},
		Trace: txndata.TxnTrace,
	})

	ExpectTxnTraces(t, ht, []WantTxnTrace{{
		MetricName:      "WebTransaction/Go/hello",
		UserAttributes:  map[string]interface{}{},
		AgentAttributes: map[string]interface{}{"request.uri": "/url"},
		Intrinsics: map[string]interface{}{
			"totalTime": 200,
			"guid":      "txn-id",
			"traceId":   "txn-id",
			"priority":  0.500000,
			"sampled":   false,
		},
		Root: WantTraceSegment{
			SegmentName:         "ROOT",
			RelativeStartMillis: 0,
			RelativeStopMillis:  123000,
			Attributes:          map[string]interface{}{},
			Children: []WantTraceSegment{{
				SegmentName:         "WebTransaction/Go/hello",
				RelativeStartMillis: 0,
				RelativeStopMillis:  123000,
				Attributes:          map[string]interface{}{"exclusive_duration_millis": 123000},
				Children: []WantTraceSegment{
					{
						SegmentName:         "Custom/5",
						RelativeStartMillis: 0,
						RelativeStopMillis:  5000,
						Attributes:          map[string]interface{}{},
						Children:            []WantTraceSegment{},
					},
					{
						SegmentName:         "Custom/6",
						RelativeStartMillis: 9000,
						RelativeStopMillis:  15000,
						Attributes:          map[string]interface{}{},
						Children:            []WantTraceSegment{},
					},
					{
						SegmentName:         "Custom/7",
						RelativeStartMillis: 18000,
						RelativeStopMillis:  25000,
						Attributes:          map[string]interface{}{},
						Children:            []WantTraceSegment{},
					},
					{
						SegmentName:         "Custom/8",
						RelativeStartMillis: 27000,
						RelativeStopMillis:  35000,
						Attributes:          map[string]interface{}{},
						Children:            []WantTraceSegment{},
					},
					{
						SegmentName:         "Custom/9",
						RelativeStartMillis: 36000,
						RelativeStopMillis:  45000,
						Attributes:          map[string]interface{}{},
						Children:            []WantTraceSegment{},
					},
				},
			}},
		},
	}})
}

func TestTxnTraceSegmentThreshold(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &TxnData{}
	thread := &Thread{}
	txndata.TxnTrace.Enabled = true
	txndata.TxnTrace.StackTraceThreshold = 1 * time.Hour
	txndata.TxnTrace.SegmentThreshold = 7 * time.Second
	txndata.TxnTrace.maxNodes = 5

	durations := []int{5, 4, 6, 3, 7, 2, 8, 1, 9}
	now := start
	for _, d := range durations {
		s := StartSegment(txndata, thread, now)
		now = now.Add(time.Duration(d) * time.Second)
		EndBasicSegment(txndata, thread, s, now, strconv.Itoa(d))
	}

	acfg := CreateAttributeConfig(sampleAttributeConfigInput, true)
	attr := NewAttributes(acfg)
	attr.Agent.Add(attributeRequestURI, "/url", nil)

	ht := newHarvestTraces()
	ht.regular.addTxnTrace(&HarvestTrace{
		TxnEvent: TxnEvent{
			Start:     start,
			Duration:  123 * time.Second,
			TotalTime: 200 * time.Second,
			FinalName: "WebTransaction/Go/hello",
			Attrs:     attr,
			BetterCAT: BetterCAT{
				Enabled:  true,
				ID:       "txn-id",
				Priority: 0.5,
			},
		},
		Trace: txndata.TxnTrace,
	})

	ExpectTxnTraces(t, ht, []WantTxnTrace{{
		MetricName:      "WebTransaction/Go/hello",
		UserAttributes:  map[string]interface{}{},
		AgentAttributes: map[string]interface{}{"request.uri": "/url"},
		Intrinsics: map[string]interface{}{
			"totalTime": 200,
			"guid":      "txn-id",
			"traceId":   "txn-id",
			"priority":  0.500000,
			"sampled":   false,
		},
		Root: WantTraceSegment{
			SegmentName:         "ROOT",
			RelativeStartMillis: 0,
			RelativeStopMillis:  123000,
			Attributes:          map[string]interface{}{},
			Children: []WantTraceSegment{{
				SegmentName:         "WebTransaction/Go/hello",
				RelativeStartMillis: 0,
				RelativeStopMillis:  123000,
				Attributes:          map[string]interface{}{"exclusive_duration_millis": 123000},
				Children: []WantTraceSegment{
					{
						SegmentName:         "Custom/7",
						RelativeStartMillis: 18000,
						RelativeStopMillis:  25000,
						Attributes:          map[string]interface{}{},
						Children:            []WantTraceSegment{},
					},
					{
						SegmentName:         "Custom/8",
						RelativeStartMillis: 27000,
						RelativeStopMillis:  35000,
						Attributes:          map[string]interface{}{},
						Children:            []WantTraceSegment{},
					},
					{
						SegmentName:         "Custom/9",
						RelativeStartMillis: 36000,
						RelativeStopMillis:  45000,
						Attributes:          map[string]interface{}{},
						Children:            []WantTraceSegment{},
					},
				},
			}},
		},
	}})
}

func TestEmptyHarvestTraces(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	ht := newHarvestTraces()
	js, err := ht.Data("12345", start)
	if nil != err || nil != js {
		t.Error(string(js), err)
	}
}

func TestLongestTraceSaved(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &TxnData{}
	txndata.TxnTrace.Enabled = true

	acfg := CreateAttributeConfig(sampleAttributeConfigInput, true)
	attr := NewAttributes(acfg)
	attr.Agent.Add(attributeRequestURI, "/url", nil)
	ht := newHarvestTraces()

	ht.Witness(HarvestTrace{
		TxnEvent: TxnEvent{
			Start:     start,
			Duration:  3 * time.Second,
			TotalTime: 4 * time.Second,
			FinalName: "WebTransaction/Go/3",
			Attrs:     attr,
			BetterCAT: BetterCAT{
				Enabled:  true,
				ID:       "txn-id-3",
				Priority: 0.5,
			},
		},
		Trace: txndata.TxnTrace,
	})
	ht.Witness(HarvestTrace{
		TxnEvent: TxnEvent{
			Start:     start,
			Duration:  5 * time.Second,
			TotalTime: 6 * time.Second,
			FinalName: "WebTransaction/Go/5",
			Attrs:     attr,
			BetterCAT: BetterCAT{
				Enabled:  true,
				ID:       "txn-id-5",
				Priority: 0.5,
			},
		},
		Trace: txndata.TxnTrace,
	})
	ht.Witness(HarvestTrace{
		TxnEvent: TxnEvent{
			Start:     start,
			Duration:  4 * time.Second,
			TotalTime: 7 * time.Second,
			FinalName: "WebTransaction/Go/4",
			Attrs:     attr,
			BetterCAT: BetterCAT{
				Enabled:  true,
				ID:       "txn-id-4",
				Priority: 0.5,
			},
		},
		Trace: txndata.TxnTrace,
	})

	ExpectTxnTraces(t, ht, []WantTxnTrace{{
		MetricName:      "WebTransaction/Go/5",
		UserAttributes:  map[string]interface{}{},
		AgentAttributes: map[string]interface{}{"request.uri": "/url"},
		Intrinsics: map[string]interface{}{
			"totalTime": 6,
			"guid":      "txn-id-5",
			"traceId":   "txn-id-5",
			"priority":  0.500000,
			"sampled":   false,
		},
		Root: WantTraceSegment{
			SegmentName:         "ROOT",
			RelativeStartMillis: 0,
			RelativeStopMillis:  5000,
			Attributes:          map[string]interface{}{},
			Children: []WantTraceSegment{{
				SegmentName:         "WebTransaction/Go/5",
				RelativeStartMillis: 0,
				RelativeStopMillis:  5000,
				Attributes:          map[string]interface{}{"exclusive_duration_millis": 5000},
				Children:            []WantTraceSegment{},
			}},
		},
	}})
}

func TestTxnTraceStackTraceThreshold(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &TxnData{}
	thread := &Thread{}
	txndata.TxnTrace.Enabled = true
	txndata.TxnTrace.StackTraceThreshold = 2 * time.Second
	txndata.TxnTrace.SegmentThreshold = 0
	txndata.TxnTrace.maxNodes = 5

	// below stack trace threshold
	t1 := StartSegment(txndata, thread, start.Add(1*time.Second))
	EndBasicSegment(txndata, thread, t1, start.Add(2*time.Second), "t1")

	// not above stack trace threshold w/out params
	t2 := StartSegment(txndata, thread, start.Add(2*time.Second))
	EndBasicSegment(txndata, thread, t2, start.Add(4*time.Second), "t2")

	// node above stack trace threshold w/ params
	t3 := StartSegment(txndata, thread, start.Add(4*time.Second))
	EndExternalSegment(EndExternalParams{
		TxnData: txndata,
		Thread:  thread,
		Start:   t3,
		Now:     start.Add(6 * time.Second),
		URL:     parseURL("http://example.com/zip/zap?secret=shhh"),
		Logger:  logger.ShimLogger{},
	})

	ht := newHarvestTraces()
	ht.Witness(HarvestTrace{
		TxnEvent: TxnEvent{
			Start:     start,
			Duration:  3 * time.Second,
			TotalTime: 4 * time.Second,
			FinalName: "WebTransaction/Go/3",
		},
		Trace: txndata.TxnTrace,
	})

	ExpectTxnTraces(t, ht, []WantTxnTrace{
		{
			MetricName:      "WebTransaction/Go/3",
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{},
			Intrinsics:      map[string]interface{}{"totalTime": 4},
			Root: WantTraceSegment{
				SegmentName:         "ROOT",
				RelativeStartMillis: 0,
				RelativeStopMillis:  3000,
				Attributes:          map[string]interface{}{},
				Children: []WantTraceSegment{{
					SegmentName:         "WebTransaction/Go/3",
					RelativeStartMillis: 0,
					RelativeStopMillis:  3000,
					Attributes:          map[string]interface{}{"exclusive_duration_millis": 3000},
					Children: []WantTraceSegment{
						{
							SegmentName:         "Custom/t1",
							RelativeStartMillis: 1000,
							RelativeStopMillis:  2000,
							Attributes:          map[string]interface{}{},
							Children:            []WantTraceSegment{},
						},
						{
							SegmentName:         "Custom/t2",
							RelativeStartMillis: 2000,
							RelativeStopMillis:  4000,
							Attributes:          map[string]interface{}{"backtrace": MatchAnything},
							Children:            []WantTraceSegment{},
						},
						{
							SegmentName:         "External/example.com/http",
							RelativeStartMillis: 4000,
							RelativeStopMillis:  6000,
							Attributes: map[string]interface{}{
								"backtrace": MatchAnything,
								"http.url":  "http://example.com/zip/zap",
							},
							Children: []WantTraceSegment{},
						},
					},
				}},
			},
		},
	})
}

func TestTxnTraceSynthetics(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &TxnData{}
	txndata.TxnTrace.Enabled = true

	acfg := CreateAttributeConfig(sampleAttributeConfigInput, true)
	attr := NewAttributes(acfg)
	attr.Agent.Add(attributeRequestURI, "/url", nil)
	ht := newHarvestTraces()

	ht.Witness(HarvestTrace{
		TxnEvent: TxnEvent{
			Start:     start,
			Duration:  3 * time.Second,
			TotalTime: 4 * time.Second,
			FinalName: "WebTransaction/Go/3",
			Attrs:     attr,
			CrossProcess: TxnCrossProcess{
				Type: txnCrossProcessSynthetics,
				Synthetics: &cat.SyntheticsHeader{
					ResourceID: "resource",
				},
			},
		},
		Trace: txndata.TxnTrace,
	})
	ht.Witness(HarvestTrace{
		TxnEvent: TxnEvent{
			Start:     start,
			Duration:  5 * time.Second,
			TotalTime: 6 * time.Second,
			FinalName: "WebTransaction/Go/5",
			Attrs:     attr,
			CrossProcess: TxnCrossProcess{
				Type: txnCrossProcessSynthetics,
				Synthetics: &cat.SyntheticsHeader{
					ResourceID: "resource",
				},
			},
		},
		Trace: txndata.TxnTrace,
	})
	ht.Witness(HarvestTrace{
		TxnEvent: TxnEvent{
			Start:     start,
			Duration:  4 * time.Second,
			TotalTime: 5 * time.Second,
			FinalName: "WebTransaction/Go/4",
			Attrs:     attr,
			CrossProcess: TxnCrossProcess{
				Type: txnCrossProcessSynthetics,
				Synthetics: &cat.SyntheticsHeader{
					ResourceID: "resource",
				},
			},
		},
		Trace: txndata.TxnTrace,
	})

	ExpectTxnTraces(t, ht, []WantTxnTrace{
		{
			MetricName:      "WebTransaction/Go/3",
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{"request.uri": "/url"},
			Intrinsics: map[string]interface{}{
				"totalTime":              4,
				"synthetics_resource_id": "resource",
			},
			Root: WantTraceSegment{
				SegmentName:         "ROOT",
				RelativeStartMillis: 0,
				RelativeStopMillis:  3000,
				Attributes:          map[string]interface{}{},
				Children: []WantTraceSegment{{
					SegmentName:         "WebTransaction/Go/3",
					RelativeStartMillis: 0,
					RelativeStopMillis:  3000,
					Attributes:          map[string]interface{}{"exclusive_duration_millis": 3000},
					Children:            []WantTraceSegment{},
				}},
			},
		},
		{
			MetricName:      "WebTransaction/Go/5",
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{"request.uri": "/url"},
			Intrinsics: map[string]interface{}{
				"totalTime":              6,
				"synthetics_resource_id": "resource",
			},
			Root: WantTraceSegment{
				SegmentName:         "ROOT",
				RelativeStartMillis: 0,
				RelativeStopMillis:  5000,
				Attributes:          map[string]interface{}{},
				Children: []WantTraceSegment{{
					SegmentName:         "WebTransaction/Go/5",
					RelativeStartMillis: 0,
					RelativeStopMillis:  5000,
					Attributes:          map[string]interface{}{"exclusive_duration_millis": 5000},
					Children:            []WantTraceSegment{},
				}},
			},
		},
		{
			MetricName:      "WebTransaction/Go/4",
			UserAttributes:  map[string]interface{}{},
			AgentAttributes: map[string]interface{}{"request.uri": "/url"},
			Intrinsics: map[string]interface{}{
				"totalTime":              5,
				"synthetics_resource_id": "resource",
			},
			Root: WantTraceSegment{
				SegmentName:         "ROOT",
				RelativeStartMillis: 0,
				RelativeStopMillis:  4000,
				Attributes:          map[string]interface{}{},
				Children: []WantTraceSegment{{
					SegmentName:         "WebTransaction/Go/4",
					RelativeStartMillis: 0,
					RelativeStopMillis:  4000,
					Attributes:          map[string]interface{}{"exclusive_duration_millis": 4000},
					Children:            []WantTraceSegment{},
				}},
			},
		},
	})
}

func TestTraceJSON(t *testing.T) {
	// Have one test compare exact JSON to ensure that all misc fields (such
	// as the trailing `null,false,null,""`) are what we expect.
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &TxnData{}
	txndata.TxnTrace.Enabled = true
	ht := newHarvestTraces()
	ht.Witness(HarvestTrace{
		TxnEvent: TxnEvent{
			Start:     start,
			Duration:  3 * time.Second,
			TotalTime: 4 * time.Second,
			FinalName: "WebTransaction/Go/trace",
			Attrs:     nil,
		},
		Trace: txndata.TxnTrace,
	})

	expect := `[
   "12345",
   [
      [
         1417136460000000,
         3000,
         "WebTransaction/Go/trace",
         null,
         [0,{},{},
            [
               0,
               3000,
               "ROOT",
               {},
               [[0,3000,"WebTransaction/Go/trace",{"exclusive_duration_millis":3000},[]]]
            ],
            {
               "agentAttributes":{},
               "userAttributes":{},
               "intrinsics":{"totalTime":4}
            }
         ],"",null,false,null,""
      ]
   ]
]`

	js, err := ht.Data("12345", start)
	if nil != err {
		t.Fatal(err)
	}
	testExpectedJSON(t, expect, string(js))
}

func TestTraceCatGUID(t *testing.T) {
	// Test catGUID is properly set in outbound json when CAT is enabled
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &TxnData{}
	txndata.TxnTrace.Enabled = true
	ht := newHarvestTraces()
	ht.Witness(HarvestTrace{
		TxnEvent: TxnEvent{
			Start:     start,
			Duration:  3 * time.Second,
			TotalTime: 4 * time.Second,
			FinalName: "WebTransaction/Go/trace",
			Attrs:     nil,
			CrossProcess: TxnCrossProcess{
				Type: 1,
				GUID: "this is guid",
			},
		},
		Trace: txndata.TxnTrace,
	})

	expect := `[
   "12345",
   [
      [
         1417136460000000,
         3000,
         "WebTransaction/Go/trace",
         null,
         [0,{},{},
            [
               0,
               3000,
               "ROOT",
               {},
               [[0,3000,"WebTransaction/Go/trace",{"exclusive_duration_millis":3000},[]]]
            ],
            {
               "agentAttributes":{},
               "userAttributes":{},
               "intrinsics":{"totalTime":4}
            }
         ],"this is guid",null,false,null,""
      ]
   ]
]`

	js, err := ht.Data("12345", start)
	if nil != err {
		t.Fatal(err)
	}
	testExpectedJSON(t, expect, string(js))
}

func TestTraceDistributedTracingGUID(t *testing.T) {
	// Test catGUID is properly set in outbound json when DT is enabled
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &TxnData{}
	txndata.TxnTrace.Enabled = true
	ht := newHarvestTraces()
	ht.Witness(HarvestTrace{
		TxnEvent: TxnEvent{
			Start:     start,
			Duration:  3 * time.Second,
			TotalTime: 4 * time.Second,
			FinalName: "WebTransaction/Go/trace",
			Attrs:     nil,
			BetterCAT: BetterCAT{
				Enabled: true,
				ID:      "this is guid",
			},
		},
		Trace: txndata.TxnTrace,
	})

	expect := `[
   "12345",
   [
      [
         1417136460000000,
         3000,
         "WebTransaction/Go/trace",
         null,
         [0,{},{},
            [
               0,
               3000,
               "ROOT",
               {},
               [[0,3000,"WebTransaction/Go/trace",{"exclusive_duration_millis":3000},[]]]
            ],
            {
               "agentAttributes":{},
               "userAttributes":{},
               "intrinsics":{
				   "totalTime":4,
				   "guid":"this is guid",
				   "traceId":"this is guid",
				   "priority":0.000000,
				   "sampled":false
			   }
            }
         ],"this is guid",null,false,null,""
      ]
   ]
]`

	js, err := ht.Data("12345", start)
	if nil != err {
		t.Fatal(err)
	}
	testExpectedJSON(t, expect, string(js))
}

func BenchmarkWitnessNode(b *testing.B) {
	trace := &TxnTrace{
		Enabled:             true,
		SegmentThreshold:    0,             // save all segments
		StackTraceThreshold: 1 * time.Hour, // no stack traces
		maxNodes:            100 * 1000,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		end := segmentEnd{
			duration:  time.Duration(RandUint32()) * time.Millisecond,
			exclusive: 0,
		}
		trace.witnessNode(end, "myNode", nil, "")
	}
}
