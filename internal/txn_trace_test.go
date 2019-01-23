package internal

import (
	"strconv"
	"testing"
	"time"

	"github.com/newrelic/go-agent/internal/cat"
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
	EndDatastoreSegment(EndDatastoreParams{
		TxnData:            txndata,
		Thread:             thread,
		Start:              t2,
		Now:                start.Add(3 * time.Second),
		Product:            "MySQL",
		Operation:          "SELECT",
		Collection:         "my_table",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
		QueryParameters:    vetQueryParameters(map[string]interface{}{"zip": 1}),
		Database:           "my_db",
		Host:               "db-server-1",
		PortPathOrID:       "3306",
	})
	t3 := StartSegment(txndata, thread, start.Add(4*time.Second))
	EndExternalSegment(txndata, thread, t3, start.Add(5*time.Second), parseURL("http://example.com/zip/zap?secret=shhh"), "", nil)
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
	EndExternalSegment(txndata, thread, t8, start.Add(15*time.Second), nil, "", nil)
	EndBasicSegment(txndata, thread, t4, start.Add(16*time.Second), "t4")

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

	expect := `["12345",[[
	   1417136460000000,
	   20000,
	   "WebTransaction/Go/hello",
	   "/url",
	   [
	      0,
	      {},
	      {},
	      [
	         0,
	         20000,
	         "ROOT",
	         {},
	         [
	            [
	               0,
	               20000,
	               "WebTransaction/Go/hello",
	               {"exclusive_duration_millis":0},
	               [
	                  [
	                     1000,
	                     6000,
	                     "Custom/t1",
	                     {},
	                     [
	                        [
	                           2000,
	                           3000,
	                           "Datastore/statement/MySQL/my_table/SELECT",
	                           {
	                              "database_name":"my_db",
	                              "host":"db-server-1",
	                              "port_path_or_id":"3306",
	                              "query":"INSERT INTO users (name, age) VALUES ($1, $2)",
	                              "query_parameters":{
	                                 "zip":1
	                              }
	                           },
	                           []
	                        ],
	                        [
	                           4000,
	                           5000,
	                           "External/example.com/all",
	                           {
	                              "uri":"http://example.com/zip/zap"
	                           },
	                           []
	                        ]
	                     ]
	                  ],
	                  [
	                     7000,
	                     16000,
	                     "Custom/t4",
	                     {},
	                     [
	                        [
	                           8000,
	                           11000,
	                           "Custom/t5",
	                           {},
	                           [
	                              [
	                                 9000,
	                                 10000,
	                                 "Custom/t6",
	                                 {},
	                                 []
	                              ]
	                           ]
	                        ],
	                        [
	                           12000,
	                           13000,
	                           "Datastore/operation/MySQL/SELECT",
	                           {
	                              "query":"'SELECT' on 'unknown' using 'MySQL'"
	                           },
	                           []
	                        ],
	                        [
	                           14000,
	                           15000,
	                           "External/unknown/all",
	                           {},
	                           []
	                        ]
	                     ]
	                  ]
	               ]
	            ]
	         ]
	      ],
	      {
	         "agentAttributes":{
	            "request.uri":"/url"
	         },
	         "userAttributes":{
	            "zap":123
	         },
	         "intrinsics":{
	         	"totalTime":30,
	         	"guid":"txn-id",
	         	"traceId":"txn-id",
	         	"priority":0.500000,
	         	"sampled":false
	         }
	      }
	   ],
	   "",
	   null,
	   false,
	   null,
	   ""
	]]]`

	js, err := ht.Data("12345", start)
	if nil != err {
		t.Fatal(err)
	}
	testExpectedJSON(t, expect, string(js))
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

	expect := `[
	   1417136460000000,
	   20000,
	   "WebTransaction/Go/hello",
	   null,
	   [
	      0,
	      {},
	      {},
	      [
	         0,
	         20000,
	         "ROOT",
	         {},
	         [
	            [
	               0,
	               20000,
	               "WebTransaction/Go/hello",
	               {"exclusive_duration_millis":0},
	               [
	               ]
	            ]
	         ]
	      ],
	      {
	         "agentAttributes":{},
	         "userAttributes":{},
	         "intrinsics":{
	         	"totalTime":30,
	         	"guid":"txn-id",
	         	"traceId":"txn-id",
	         	"priority":0.500000,
	         	"sampled":false
	         }
	      }
	   ],
	   "",
	   null,
	   false,
	   null,
	   ""
	]`

	js, err := ht.slice()[0].MarshalJSON()
	if nil != err {
		t.Fatal(err)
	}
	testExpectedJSON(t, expect, string(js))
}

func TestTxnTraceAsync(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &TxnData{}
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

	expect := `[
   1417136460000000,
   20000,
   "WebTransaction/Go/hello",
   null,
   [
      0,
      {},
      {},
      [
         0,
         20000,
         "ROOT",
         {},
         [
            [
               0,
               20000,
               "WebTransaction/Go/hello",
               {"exclusive_duration_millis":0},
               [
                  [
                     1000,
                     8000,
                     "Custom/thread1.segment1",
                     {},
                     [
                        [
                           2000,
                           4000,
                           "Custom/thread1.segment2",
                           {},
                           []
                        ]
                     ]
                  ],
                  [
                     3000,
                     5000,
                     "Custom/thread2.segment1",
                     {},
                     []
                  ],
                  [
                     6000,
                     10000,
                     "Custom/thread3.segment1",
                     {},
                     [
                        [
                           7000,
                           9000,
                           "Custom/thread3.segment2",
                           {},
                           []
                        ]
                     ]
                  ]
               ]
            ]
         ]
      ],
      {
         "agentAttributes":{

         },
         "userAttributes":{

         },
         "intrinsics":{
            "totalTime":30,
            "guid":"txn-id",
            "traceId":"txn-id",
            "priority":0.500000,
            "sampled":false
         }
      }
   ],
   "",
   null,
   false,
   null,
   ""
]`

	js, err := ht.slice()[0].MarshalJSON()
	if nil != err {
		t.Fatal(err)
	}
	testExpectedJSON(t, expect, string(js))
}

func TestTxnTraceOldCAT(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	txndata := &TxnData{}
	thread := &Thread{}
	txndata.TxnTrace.Enabled = true
	txndata.TxnTrace.StackTraceThreshold = 1 * time.Hour
	txndata.TxnTrace.SegmentThreshold = 0

	t1 := StartSegment(txndata, thread, start.Add(1*time.Second))
	t2 := StartSegment(txndata, thread, start.Add(2*time.Second))
	EndDatastoreSegment(EndDatastoreParams{
		TxnData:            txndata,
		Thread:             thread,
		Start:              t2,
		Now:                start.Add(3 * time.Second),
		Product:            "MySQL",
		Operation:          "SELECT",
		Collection:         "my_table",
		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
		QueryParameters:    vetQueryParameters(map[string]interface{}{"zip": 1}),
		Database:           "my_db",
		Host:               "db-server-1",
		PortPathOrID:       "3306",
	})
	t3 := StartSegment(txndata, thread, start.Add(4*time.Second))
	EndExternalSegment(txndata, thread, t3, start.Add(5*time.Second), parseURL("http://example.com/zip/zap?secret=shhh"), "", nil)
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
	EndExternalSegment(txndata, thread, t8, start.Add(15*time.Second), nil, "", nil)
	EndBasicSegment(txndata, thread, t4, start.Add(16*time.Second), "t4")

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

	expect := `["12345",[[
	   1417136460000000,
	   20000,
	   "WebTransaction/Go/hello",
	   "/url",
	   [
	      0,
	      {},
	      {},
	      [
	         0,
	         20000,
	         "ROOT",
	         {},
	         [
	            [
	               0,
	               20000,
	               "WebTransaction/Go/hello",
	               {"exclusive_duration_millis":0},
	               [
	                  [
	                     1000,
	                     6000,
	                     "Custom/t1",
	                     {},
	                     [
	                        [
	                           2000,
	                           3000,
	                           "Datastore/statement/MySQL/my_table/SELECT",
	                           {
	                              "database_name":"my_db",
	                              "host":"db-server-1",
	                              "port_path_or_id":"3306",
	                              "query":"INSERT INTO users (name, age) VALUES ($1, $2)",
	                              "query_parameters":{
	                                 "zip":1
	                              }
	                           },
	                           []
	                        ],
	                        [
	                           4000,
	                           5000,
	                           "External/example.com/all",
	                           {
	                              "uri":"http://example.com/zip/zap"
	                           },
	                           []
	                        ]
	                     ]
	                  ],
	                  [
	                     7000,
	                     16000,
	                     "Custom/t4",
	                     {},
	                     [
	                        [
	                           8000,
	                           11000,
	                           "Custom/t5",
	                           {},
	                           [
	                              [
	                                 9000,
	                                 10000,
	                                 "Custom/t6",
	                                 {},
	                                 []
	                              ]
	                           ]
	                        ],
	                        [
	                           12000,
	                           13000,
	                           "Datastore/operation/MySQL/SELECT",
	                           {
	                              "query":"'SELECT' on 'unknown' using 'MySQL'"
	                           },
	                           []
	                        ],
	                        [
	                           14000,
	                           15000,
	                           "External/unknown/all",
	                           {},
	                           []
	                        ]
	                     ]
	                  ]
	               ]
	            ]
	         ]
	      ],
	      {
	         "agentAttributes":{"request.uri":"/url"},
	         "userAttributes":{
	            "zap":123
	         },
	         "intrinsics":{
	            "totalTime":30
	         }
	      }
	   ],
	   "",
	   null,
	   false,
	   null,
	   ""
	]]]`

	js, err := ht.Data("12345", start)
	if nil != err {
		t.Fatal(err)
	}
	testExpectedJSON(t, expect, string(js))
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

	expect := `["12345",[[
	   1417136460000000,
	   20000,
	   "WebTransaction/Go/hello",
	   null,
	   [
	      0,
	      {},
	      {},
	      [
	         0,
	         20000,
	         "ROOT",
	         {},
	         [
	            [
	               0,
	               20000,
	               "WebTransaction/Go/hello",
	               {"exclusive_duration_millis":0},
	               []
	            ]
	         ]
	      ],
	      {
	         "agentAttributes":{},
	         "userAttributes":{},
	         "intrinsics":{
	            "totalTime":0,
		        "guid":"txn-id",
	         	"traceId":"txn-id",
	         	"priority":0.500000,
	         	"sampled":false
	         }
	      }
	   ],
	   "",
	   null,
	   false,
	   null,
	   ""
	]]]`
	js, err := ht.Data("12345", start)
	if nil != err {
		t.Fatal(err)
	}
	testExpectedJSON(t, expect, string(js))
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

	expect := `["12345",[[
	   1417136460000000,
	   20000,
	   "WebTransaction/Go/hello",
	   null,
	   [
	      0,
	      {},
	      {},
	      [
	         0,
	         20000,
	         "ROOT",
	         {},
	         [
	            [
	               0,
	               20000,
	               "WebTransaction/Go/hello",
	               {"exclusive_duration_millis":0},
	               []
	            ]
	         ]
	      ],
	      {
	         "agentAttributes":{},
	         "userAttributes":{},
	         "intrinsics":{
	         	"totalTime":30,
	         	"guid":"txn-id",
	         	"traceId":"txn-id",
	         	"priority":0.500000,
	         	"sampled":false
	         }
	      }
	   ],
	   "",
	   null,
	   false,
	   null,
	   ""
	]]]`
	js, err := ht.Data("12345", start)
	if nil != err {
		t.Fatal(err)
	}
	testExpectedJSON(t, expect, string(js))
}

func TestTxnTraceNoSegmentsNoAttributesOldCAT(t *testing.T) {
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
		},
		Trace: txndata.TxnTrace,
	})

	expect := `["12345",[[
	   1417136460000000,
	   20000,
	   "WebTransaction/Go/hello",
	   null,
	   [
	      0,
	      {},
	      {},
	      [
	         0,
	         20000,
	         "ROOT",
	         {},
	         [
	            [
	               0,
	               20000,
	               "WebTransaction/Go/hello",
	               {"exclusive_duration_millis":0},
	               []
	            ]
	         ]
	      ],
	      {
	         "agentAttributes":{},
	         "userAttributes":{},
	         "intrinsics":{
	            "totalTime":30
	         }
	      }
	   ],
	   "",
	   null,
	   false,
	   null,
	   ""
	]]]`
	js, err := ht.Data("12345", start)
	if nil != err {
		t.Fatal(err)
	}
	testExpectedJSON(t, expect, string(js))
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

	expect := `["12345",[[
	   1417136460000000,
	   123000,
	   "WebTransaction/Go/hello",
	   "/url",
	   [
	      0,
	      {},
	      {},
	      [
	         0,
	         123000,
	         "ROOT",
	         {},
	         [
	            [
	               0,
	               123000,
	               "WebTransaction/Go/hello",
	               {"exclusive_duration_millis":0},
	               [
	                  [
	                     0,
	                     5000,
	                     "Custom/5",
	                     {},
	                     []
	                  ],
	                  [
	                     9000,
	                     15000,
	                     "Custom/6",
	                     {},
	                     []
	                  ],
	                  [
	                     18000,
	                     25000,
	                     "Custom/7",
	                     {},
	                     []
	                  ],
	                  [
	                     27000,
	                     35000,
	                     "Custom/8",
	                     {},
	                     []
	                  ],
	                  [
	                     36000,
	                     45000,
	                     "Custom/9",
	                     {},
	                     []
	                  ]
	               ]
	            ]
	         ]
	      ],
	      {
	         "agentAttributes":{"request.uri":"/url"},
	         "userAttributes":{},
	         "intrinsics":{
	         	"totalTime":200,
	         	"guid":"txn-id",
	         	"traceId":"txn-id",
	         	"priority":0.500000,
	         	"sampled":false
	         }
	      }
	   ],
	   "",
	   null,
	   false,
	   null,
	   ""
	]]]`
	js, err := ht.Data("12345", start)
	if nil != err {
		t.Fatal(err)
	}
	testExpectedJSON(t, expect, string(js))
}

func TestTxnTraceSlowestNodesSavedOldCAT(t *testing.T) {
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
		},
		Trace: txndata.TxnTrace,
	})

	expect := `["12345",[[
	   1417136460000000,
	   123000,
	   "WebTransaction/Go/hello",
	   "/url",
	   [
	      0,
	      {},
	      {},
	      [
	         0,
	         123000,
	         "ROOT",
	         {},
	         [
	            [
	               0,
	               123000,
	               "WebTransaction/Go/hello",
	               {"exclusive_duration_millis":0},
	               [
	                  [
	                     0,
	                     5000,
	                     "Custom/5",
	                     {},
	                     []
	                  ],
	                  [
	                     9000,
	                     15000,
	                     "Custom/6",
	                     {},
	                     []
	                  ],
	                  [
	                     18000,
	                     25000,
	                     "Custom/7",
	                     {},
	                     []
	                  ],
	                  [
	                     27000,
	                     35000,
	                     "Custom/8",
	                     {},
	                     []
	                  ],
	                  [
	                     36000,
	                     45000,
	                     "Custom/9",
	                     {},
	                     []
	                  ]
	               ]
	            ]
	         ]
	      ],
	      {
	         "agentAttributes":{"request.uri":"/url"},
	         "userAttributes":{},
	         "intrinsics":{
	            "totalTime":200
	         }
	      }
	   ],
	   "",
	   null,
	   false,
	   null,
	   ""
	]]]`
	js, err := ht.Data("12345", start)
	if nil != err {
		t.Fatal(err)
	}
	testExpectedJSON(t, expect, string(js))
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

	expect := `["12345",[[
	   1417136460000000,
	   123000,
	   "WebTransaction/Go/hello",
	   "/url",
	   [
	      0,
	      {},
	      {},
	      [
	         0,
	         123000,
	         "ROOT",
	         {},
	         [
	            [
	               0,
	               123000,
	               "WebTransaction/Go/hello",
	               {"exclusive_duration_millis":0},
	               [
	                  [
	                     18000,
	                     25000,
	                     "Custom/7",
	                     {},
	                     []
	                  ],
	                  [
	                     27000,
	                     35000,
	                     "Custom/8",
	                     {},
	                     []
	                  ],
	                  [
	                     36000,
	                     45000,
	                     "Custom/9",
	                     {},
	                     []
	                  ]
	               ]
	            ]
	         ]
	      ],
	      {
	         "agentAttributes":{"request.uri":"/url"},
	         "userAttributes":{},
	         "intrinsics":{
				"totalTime":200,
				"guid":"txn-id",
				"traceId":"txn-id",
				"priority":0.500000,
				"sampled":false
	         }
	      }
	   ],
	   "",
	   null,
	   false,
	   null,
	   ""
	]]]`
	js, err := ht.Data("12345", start)
	if nil != err {
		t.Fatal(err)
	}
	testExpectedJSON(t, expect, string(js))
}

func TestTxnTraceSegmentThresholdOldCAT(t *testing.T) {
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
		},
		Trace: txndata.TxnTrace,
	})

	expect := `["12345",[[
	   1417136460000000,
	   123000,
	   "WebTransaction/Go/hello",
	   "/url",
	   [
	      0,
	      {},
	      {},
	      [
	         0,
	         123000,
	         "ROOT",
	         {},
	         [
	            [
	               0,
	               123000,
	               "WebTransaction/Go/hello",
	               {"exclusive_duration_millis":0},
	               [
	                  [
	                     18000,
	                     25000,
	                     "Custom/7",
	                     {},
	                     []
	                  ],
	                  [
	                     27000,
	                     35000,
	                     "Custom/8",
	                     {},
	                     []
	                  ],
	                  [
	                     36000,
	                     45000,
	                     "Custom/9",
	                     {},
	                     []
	                  ]
	               ]
	            ]
	         ]
	      ],
	      {
	         "agentAttributes":{"request.uri":"/url"},
	         "userAttributes":{},
	         "intrinsics":{
	            "totalTime":200
	         }
	      }
	   ],
	   "",
	   null,
	   false,
	   null,
	   ""
	]]]`
	js, err := ht.Data("12345", start)
	if nil != err {
		t.Fatal(err)
	}
	testExpectedJSON(t, expect, string(js))
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

	expect := `
[
	"12345",
	[
		[
			1417136460000000,5000,"WebTransaction/Go/5","/url",
			[
				0,{},{},
				[0,5000,"ROOT",{},
					[[0,5000,"WebTransaction/Go/5",{"exclusive_duration_millis":0},[]]]
				],
				{
					"agentAttributes":{"request.uri":"/url"},
					"userAttributes":{},
					"intrinsics":{
						"totalTime":6,
						"guid":"txn-id-5",
						"traceId":"txn-id-5",
						"priority":0.500000,
						"sampled":false
					}
				}
			],
			"",null,false,null,""
		]
	]
]`
	js, err := ht.Data("12345", start)
	if nil != err {
		t.Fatal(err)
	}
	testExpectedJSON(t, expect, string(js))
}

func TestLongestTraceSavedOldCAT(t *testing.T) {
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
		},
		Trace: txndata.TxnTrace,
	})

	expect := `
[
	"12345",
	[
		[
			1417136460000000,5000,"WebTransaction/Go/5","/url",
			[
				0,{},{},
				[0,5000,"ROOT",{},
					[[0,5000,"WebTransaction/Go/5",{"exclusive_duration_millis":0},[]]]
				],
				{
					"agentAttributes":{"request.uri":"/url"},
					"userAttributes":{},
					"intrinsics":{
						"totalTime":6
					}
				}
			],
			"",null,false,null,""
		]
	]
]`
	js, err := ht.Data("12345", start)
	if nil != err {
		t.Fatal(err)
	}
	testExpectedJSON(t, expect, string(js))
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
	EndDatastoreSegment(EndDatastoreParams{
		TxnData:    txndata,
		Thread:     thread,
		Start:      t2,
		Now:        start.Add(4 * time.Second),
		Product:    "MySQL",
		Collection: "my_table",
		Operation:  "SELECT",
	})

	// node above stack trace threshold w/ params
	t3 := StartSegment(txndata, thread, start.Add(4*time.Second))
	EndExternalSegment(txndata, thread, t3, start.Add(6*time.Second), parseURL("http://example.com/zip/zap?secret=shhh"), "", nil)

	p := txndata.TxnTrace.nodes[0].params
	if nil != p {
		t.Error(p)
	}
	p = txndata.TxnTrace.nodes[1].params
	if nil == p || nil == p.StackTrace || "" != p.CleanURL {
		t.Error(p)
	}
	p = txndata.TxnTrace.nodes[2].params
	if nil == p || nil == p.StackTrace || "http://example.com/zip/zap" != p.CleanURL {
		t.Error(p)
	}
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

	expect := `
[
	"12345",
	[
		[
			1417136460000000,3000,"WebTransaction/Go/3","/url",
			[
				0,{},{},
				[0,3000,"ROOT",{},
					[[0,3000,"WebTransaction/Go/3",{"exclusive_duration_millis":0},[]]]
				],
				{
					"agentAttributes":{"request.uri":"/url"},
					"userAttributes":{},
					"intrinsics":{
						"totalTime":4,
						"synthetics_resource_id":"resource"
					}
				}
			],
			"",null,false,null,"resource"
		],
		[
			1417136460000000,5000,"WebTransaction/Go/5","/url",
			[
				0,{},{},
				[0,5000,"ROOT",{},
					[[0,5000,"WebTransaction/Go/5",{"exclusive_duration_millis":0},[]]]
				],
				{
					"agentAttributes":{"request.uri":"/url"},
					"userAttributes":{},
					"intrinsics":{
						"totalTime":6,
						"synthetics_resource_id":"resource"
					}
				}
			],
			"",null,false,null,"resource"
		],
		[
			1417136460000000,4000,"WebTransaction/Go/4","/url",
			[
				0,{},{},
				[0,4000,"ROOT",{},
					[[0,4000,"WebTransaction/Go/4",{"exclusive_duration_millis":0},[]]]
				],
				{
					"agentAttributes":{"request.uri":"/url"},
					"userAttributes":{},
					"intrinsics":{
						"totalTime":5,
						"synthetics_resource_id":"resource"
					}
				}
			],
			"",null,false,null,"resource"
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
		trace.witnessNode(end, "myNode", nil)
	}
}
