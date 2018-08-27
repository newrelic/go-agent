package internal

import (
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/newrelic/go-agent/internal/crossagent"
)

func TestStartEndSegment(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)

	tr := &TxnData{}
	token := StartSegment(tr, start)
	stop := start.Add(1 * time.Second)
	end, err := endSegment(tr, token, stop)
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
	if 0 != len(tr.spanEvents) {
		t.Error(tr.spanEvents)
	}
}

func TestMultipleChildren(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &TxnData{}

	t1 := StartSegment(tr, start.Add(1*time.Second))
	t2 := StartSegment(tr, start.Add(2*time.Second))
	end2, err2 := endSegment(tr, t2, start.Add(3*time.Second))
	t3 := StartSegment(tr, start.Add(4*time.Second))
	end3, err3 := endSegment(tr, t3, start.Add(5*time.Second))
	end1, err1 := endSegment(tr, t1, start.Add(6*time.Second))
	t4 := StartSegment(tr, start.Add(7*time.Second))
	end4, err4 := endSegment(tr, t4, start.Add(8*time.Second))

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
	children := TracerRootChildren(tr)
	if children != 6*time.Second {
		t.Error(children)
	}
}

func TestInvalidStart(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &TxnData{}

	end, err := endSegment(tr, SegmentStartTime{}, start.Add(1*time.Second))
	if err != errMalformedSegment {
		t.Error(end, err)
	}
	StartSegment(tr, start.Add(2*time.Second))
	end, err = endSegment(tr, SegmentStartTime{}, start.Add(3*time.Second))
	if err != errMalformedSegment {
		t.Error(end, err)
	}
}

func TestSegmentAlreadyEnded(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &TxnData{}

	t1 := StartSegment(tr, start.Add(1*time.Second))
	end, err := endSegment(tr, t1, start.Add(2*time.Second))
	if err != nil {
		t.Error(end, err)
	}
	end, err = endSegment(tr, t1, start.Add(3*time.Second))
	if err != errSegmentOrder {
		t.Error(end, err)
	}
}

func TestSegmentBadStamp(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &TxnData{}

	t1 := StartSegment(tr, start.Add(1*time.Second))
	t1.Stamp++
	end, err := endSegment(tr, t1, start.Add(2*time.Second))
	if err != errSegmentOrder {
		t.Error(end, err)
	}
}

func TestSegmentBadDepth(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &TxnData{}

	t1 := StartSegment(tr, start.Add(1*time.Second))
	t1.Depth++
	end, err := endSegment(tr, t1, start.Add(2*time.Second))
	if err != errSegmentOrder {
		t.Error(end, err)
	}
}

func TestSegmentNegativeDepth(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &TxnData{}

	t1 := StartSegment(tr, start.Add(1*time.Second))
	t1.Depth = -1
	end, err := endSegment(tr, t1, start.Add(2*time.Second))
	if err != errMalformedSegment {
		t.Error(end, err)
	}
}

func TestSegmentOutOfOrder(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &TxnData{}

	t1 := StartSegment(tr, start.Add(1*time.Second))
	t2 := StartSegment(tr, start.Add(2*time.Second))
	t3 := StartSegment(tr, start.Add(3*time.Second))
	end2, err2 := endSegment(tr, t2, start.Add(4*time.Second))
	end3, err3 := endSegment(tr, t3, start.Add(5*time.Second))
	t4 := StartSegment(tr, start.Add(6*time.Second))
	end4, err4 := endSegment(tr, t4, start.Add(7*time.Second))
	end1, err1 := endSegment(tr, t1, start.Add(8*time.Second))

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
	tr := &TxnData{}

	alpha := StartSegment(tr, start.Add(1*time.Second))
	t1 := StartSegment(tr, start.Add(2*time.Second))
	EndBasicSegment(tr, t1, start.Add(3*time.Second), "t1")
	StartSegment(tr, start.Add(4*time.Second))
	t2 := StartSegment(tr, start.Add(5*time.Second))
	EndBasicSegment(tr, t2, start.Add(6*time.Second), "t2")
	StartSegment(tr, start.Add(7*time.Second))
	t3 := StartSegment(tr, start.Add(8*time.Second))
	EndBasicSegment(tr, t3, start.Add(9*time.Second), "t3")
	t4 := StartSegment(tr, start.Add(10*time.Second))
	EndBasicSegment(tr, t4, start.Add(11*time.Second), "t4")
	EndBasicSegment(tr, alpha, start.Add(12*time.Second), "alpha")

	metrics := newMetricTable(100, time.Now())
	tr.FinalName = "WebTransaction/Go/zip"
	tr.IsWeb = true
	MergeBreakdownMetrics(tr, metrics)
	ExpectMetrics(t, metrics, []WantMetric{
		{"Custom/alpha", "", false, []float64{1, 11, 7, 11, 11, 121}},
		{"Custom/t1", "", false, []float64{1, 1, 1, 1, 1, 1}},
		{"Custom/t2", "", false, []float64{1, 1, 1, 1, 1, 1}},
		{"Custom/t3", "", false, []float64{1, 1, 1, 1, 1, 1}},
		{"Custom/t4", "", false, []float64{1, 1, 1, 1, 1, 1}},
		{"Custom/alpha", tr.FinalName, false, []float64{1, 11, 7, 11, 11, 121}},
		{"Custom/t1", tr.FinalName, false, []float64{1, 1, 1, 1, 1, 1}},
		{"Custom/t2", tr.FinalName, false, []float64{1, 1, 1, 1, 1, 1}},
		{"Custom/t3", tr.FinalName, false, []float64{1, 1, 1, 1, 1, 1}},
		{"Custom/t4", tr.FinalName, false, []float64{1, 1, 1, 1, 1, 1}},
	})
}

//                                          |-t3-|    |-t4-|
//                           |-t2-|    |-never-finished----------
//            |-t1-|    |--never-finished------------------------
//  |-------root-------------------------------------------------
//  0    1    2    3    4    5    6    7    8    9    10   11   12
func TestLostChildrenRoot(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &TxnData{}

	t1 := StartSegment(tr, start.Add(2*time.Second))
	EndBasicSegment(tr, t1, start.Add(3*time.Second), "t1")
	StartSegment(tr, start.Add(4*time.Second))
	t2 := StartSegment(tr, start.Add(5*time.Second))
	EndBasicSegment(tr, t2, start.Add(6*time.Second), "t2")
	StartSegment(tr, start.Add(7*time.Second))
	t3 := StartSegment(tr, start.Add(8*time.Second))
	EndBasicSegment(tr, t3, start.Add(9*time.Second), "t3")
	t4 := StartSegment(tr, start.Add(10*time.Second))
	EndBasicSegment(tr, t4, start.Add(11*time.Second), "t4")

	children := TracerRootChildren(tr)
	if children != 4*time.Second {
		t.Error(children)
	}

	metrics := newMetricTable(100, time.Now())
	tr.FinalName = "WebTransaction/Go/zip"
	tr.IsWeb = true
	MergeBreakdownMetrics(tr, metrics)
	ExpectMetrics(t, metrics, []WantMetric{
		{"Custom/t1", "", false, []float64{1, 1, 1, 1, 1, 1}},
		{"Custom/t2", "", false, []float64{1, 1, 1, 1, 1, 1}},
		{"Custom/t3", "", false, []float64{1, 1, 1, 1, 1, 1}},
		{"Custom/t4", "", false, []float64{1, 1, 1, 1, 1, 1}},
		{"Custom/t1", tr.FinalName, false, []float64{1, 1, 1, 1, 1, 1}},
		{"Custom/t2", tr.FinalName, false, []float64{1, 1, 1, 1, 1, 1}},
		{"Custom/t3", tr.FinalName, false, []float64{1, 1, 1, 1, 1, 1}},
		{"Custom/t4", tr.FinalName, false, []float64{1, 1, 1, 1, 1, 1}},
	})
}

func TestNilSpanEvent(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)

	tr := &TxnData{}
	token := StartSegment(tr, start)
	stop := start.Add(1 * time.Second)
	end, err := endSegment(tr, token, stop)
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

	tr := &TxnData{}
	token := StartSegment(tr, start)
	stop := start.Add(1 * time.Second)
	end, err := endSegment(tr, token, stop)
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

func TestNewSpanID(t *testing.T) {
	id := NewSpanID()
	if len(id) != 16 {
		t.Error(id)
	}
}

func TestGetRootSpanID(t *testing.T) {
	tr := &TxnData{}
	id := tr.getRootSpanID()
	if id == "" {
		t.Error(id)
	}
	tr.rootSpanID = "0123456789ABCDEF"
	id = tr.getRootSpanID()
	if id != "0123456789ABCDEF" {
		t.Error(id)
	}
}

func TestCurrentSpanIdentifier(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &TxnData{}
	tr.rootSpanID = "0123456789ABCDEF"
	id := tr.CurrentSpanIdentifier()
	if id != "0123456789ABCDEF" {
		t.Error(id)
	}

	// After starting and ending a segment, the current span id is still the root.
	t1 := StartSegment(tr, start.Add(1*time.Second))
	_, err1 := endSegment(tr, t1, start.Add(3*time.Second))
	if nil != err1 {
		t.Error(err1)
	}

	id = tr.CurrentSpanIdentifier()
	if id != "0123456789ABCDEF" {
		t.Error(id)
	}

	// After starting a new segment, there should be a new current span id.
	StartSegment(tr, start.Add(2*time.Second))
	id2 := tr.CurrentSpanIdentifier()
	if id2 == "0123456789ABCDEF" ||
		id2 != tr.stack[0].spanID {
		t.Error(id2)
	}

	// The current segment has not ended, so there should be no new current span id.
	id = tr.CurrentSpanIdentifier()
	if id != id2 {
		t.Error(id)
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
	tr := &TxnData{}

	t1 := StartSegment(tr, start.Add(1*time.Second))
	t2 := StartSegment(tr, start.Add(2*time.Second))
	EndBasicSegment(tr, t2, start.Add(3*time.Second), "t2")
	EndBasicSegment(tr, t1, start.Add(4*time.Second), "t1")
	t3 := StartSegment(tr, start.Add(5*time.Second))
	t4 := StartSegment(tr, start.Add(6*time.Second))
	EndBasicSegment(tr, t3, start.Add(7*time.Second), "t3")
	EndBasicSegment(tr, t4, start.Add(8*time.Second), "out-of-order")
	t5 := StartSegment(tr, start.Add(9*time.Second))
	EndBasicSegment(tr, t5, start.Add(10*time.Second), "t1")

	metrics := newMetricTable(100, time.Now())
	tr.FinalName = "WebTransaction/Go/zip"
	tr.IsWeb = true
	MergeBreakdownMetrics(tr, metrics)
	ExpectMetrics(t, metrics, []WantMetric{
		{"Custom/t1", "", false, []float64{2, 4, 3, 1, 3, 10}},
		{"Custom/t2", "", false, []float64{1, 1, 1, 1, 1, 1}},
		{"Custom/t3", "", false, []float64{1, 2, 2, 2, 2, 4}},
		{"Custom/t1", tr.FinalName, false, []float64{2, 4, 3, 1, 3, 10}},
		{"Custom/t2", tr.FinalName, false, []float64{1, 1, 1, 1, 1, 1}},
		{"Custom/t3", tr.FinalName, false, []float64{1, 2, 2, 2, 2, 4}},
	})
}

func parseURL(raw string) *url.URL {
	u, _ := url.Parse(raw)
	return u
}

func TestSegmentExternal(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &TxnData{}

	t1 := StartSegment(tr, start.Add(1*time.Second))
	t2 := StartSegment(tr, start.Add(2*time.Second))
	EndExternalSegment(tr, t2, start.Add(3*time.Second), nil, "", nil)
	EndExternalSegment(tr, t1, start.Add(4*time.Second), parseURL("http://f1.com"), "", nil)
	t3 := StartSegment(tr, start.Add(5*time.Second))
	EndExternalSegment(tr, t3, start.Add(6*time.Second), parseURL("http://f1.com"), "", nil)
	t4 := StartSegment(tr, start.Add(7*time.Second))
	t4.Stamp++
	EndExternalSegment(tr, t4, start.Add(8*time.Second), parseURL("http://invalid-token.com"), "", nil)

	if tr.externalCallCount != 3 {
		t.Error(tr.externalCallCount)
	}
	if tr.externalDuration != 5*time.Second {
		t.Error(tr.externalDuration)
	}
	metrics := newMetricTable(100, time.Now())
	tr.FinalName = "WebTransaction/Go/zip"
	tr.IsWeb = true
	MergeBreakdownMetrics(tr, metrics)
	ExpectMetrics(t, metrics, []WantMetric{
		{"External/all", "", true, []float64{3, 5, 4, 1, 3, 11}},
		{"External/allWeb", "", true, []float64{3, 5, 4, 1, 3, 11}},
		{"External/f1.com/all", "", false, []float64{2, 4, 3, 1, 3, 10}},
		{"External/unknown/all", "", false, []float64{1, 1, 1, 1, 1, 1}},
		{"External/f1.com/all", tr.FinalName, false, []float64{2, 4, 3, 1, 3, 10}},
		{"External/unknown/all", tr.FinalName, false, []float64{1, 1, 1, 1, 1, 1}},
	})

	metrics = newMetricTable(100, time.Now())
	tr.FinalName = "OtherTransaction/Go/zip"
	tr.IsWeb = false
	MergeBreakdownMetrics(tr, metrics)
	ExpectMetrics(t, metrics, []WantMetric{
		{"External/all", "", true, []float64{3, 5, 4, 1, 3, 11}},
		{"External/allOther", "", true, []float64{3, 5, 4, 1, 3, 11}},
		{"External/f1.com/all", "", false, []float64{2, 4, 3, 1, 3, 10}},
		{"External/unknown/all", "", false, []float64{1, 1, 1, 1, 1, 1}},
		{"External/f1.com/all", tr.FinalName, false, []float64{2, 4, 3, 1, 3, 10}},
		{"External/unknown/all", tr.FinalName, false, []float64{1, 1, 1, 1, 1, 1}},
	})
}

func TestSegmentDatastore(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &TxnData{}

	t1 := StartSegment(tr, start.Add(1*time.Second))
	t2 := StartSegment(tr, start.Add(2*time.Second))
	EndDatastoreSegment(EndDatastoreParams{
		Tracer:     tr,
		Start:      t2,
		Now:        start.Add(3 * time.Second),
		Product:    "MySQL",
		Operation:  "SELECT",
		Collection: "my_table",
	})
	EndDatastoreSegment(EndDatastoreParams{
		Tracer:    tr,
		Start:     t1,
		Now:       start.Add(4 * time.Second),
		Product:   "MySQL",
		Operation: "SELECT",
		// missing collection
	})
	t3 := StartSegment(tr, start.Add(5*time.Second))
	EndDatastoreSegment(EndDatastoreParams{
		Tracer:    tr,
		Start:     t3,
		Now:       start.Add(6 * time.Second),
		Product:   "MySQL",
		Operation: "SELECT",
		// missing collection
	})
	t4 := StartSegment(tr, start.Add(7*time.Second))
	t4.Stamp++
	EndDatastoreSegment(EndDatastoreParams{
		Tracer:    tr,
		Start:     t4,
		Now:       start.Add(8 * time.Second),
		Product:   "MySQL",
		Operation: "invalid-token",
	})
	t5 := StartSegment(tr, start.Add(9*time.Second))
	EndDatastoreSegment(EndDatastoreParams{
		Tracer: tr,
		Start:  t5,
		Now:    start.Add(10 * time.Second),
		// missing datastore, collection, and operation
	})

	if tr.datastoreCallCount != 4 {
		t.Error(tr.datastoreCallCount)
	}
	if tr.datastoreDuration != 6*time.Second {
		t.Error(tr.datastoreDuration)
	}
	metrics := newMetricTable(100, time.Now())
	tr.FinalName = "WebTransaction/Go/zip"
	tr.IsWeb = true
	MergeBreakdownMetrics(tr, metrics)
	ExpectMetrics(t, metrics, []WantMetric{
		{"Datastore/all", "", true, []float64{4, 6, 5, 1, 3, 12}},
		{"Datastore/allWeb", "", true, []float64{4, 6, 5, 1, 3, 12}},
		{"Datastore/MySQL/all", "", true, []float64{3, 5, 4, 1, 3, 11}},
		{"Datastore/MySQL/allWeb", "", true, []float64{3, 5, 4, 1, 3, 11}},
		{"Datastore/Unknown/all", "", true, []float64{1, 1, 1, 1, 1, 1}},
		{"Datastore/Unknown/allWeb", "", true, []float64{1, 1, 1, 1, 1, 1}},
		{"Datastore/operation/MySQL/SELECT", "", false, []float64{3, 5, 4, 1, 3, 11}},
		{"Datastore/operation/MySQL/SELECT", tr.FinalName, false, []float64{2, 4, 3, 1, 3, 10}},
		{"Datastore/operation/Unknown/other", "", false, []float64{1, 1, 1, 1, 1, 1}},
		{"Datastore/operation/Unknown/other", tr.FinalName, false, []float64{1, 1, 1, 1, 1, 1}},
		{"Datastore/statement/MySQL/my_table/SELECT", "", false, []float64{1, 1, 1, 1, 1, 1}},
		{"Datastore/statement/MySQL/my_table/SELECT", tr.FinalName, false, []float64{1, 1, 1, 1, 1, 1}},
	})

	metrics = newMetricTable(100, time.Now())
	tr.FinalName = "OtherTransaction/Go/zip"
	tr.IsWeb = false
	MergeBreakdownMetrics(tr, metrics)
	ExpectMetrics(t, metrics, []WantMetric{
		{"Datastore/all", "", true, []float64{4, 6, 5, 1, 3, 12}},
		{"Datastore/allOther", "", true, []float64{4, 6, 5, 1, 3, 12}},
		{"Datastore/MySQL/all", "", true, []float64{3, 5, 4, 1, 3, 11}},
		{"Datastore/MySQL/allOther", "", true, []float64{3, 5, 4, 1, 3, 11}},
		{"Datastore/Unknown/all", "", true, []float64{1, 1, 1, 1, 1, 1}},
		{"Datastore/Unknown/allOther", "", true, []float64{1, 1, 1, 1, 1, 1}},
		{"Datastore/operation/MySQL/SELECT", "", false, []float64{3, 5, 4, 1, 3, 11}},
		{"Datastore/operation/MySQL/SELECT", tr.FinalName, false, []float64{2, 4, 3, 1, 3, 10}},
		{"Datastore/operation/Unknown/other", "", false, []float64{1, 1, 1, 1, 1, 1}},
		{"Datastore/operation/Unknown/other", tr.FinalName, false, []float64{1, 1, 1, 1, 1, 1}},
		{"Datastore/statement/MySQL/my_table/SELECT", "", false, []float64{1, 1, 1, 1, 1, 1}},
		{"Datastore/statement/MySQL/my_table/SELECT", tr.FinalName, false, []float64{1, 1, 1, 1, 1, 1}},
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

		tr := &TxnData{}
		s := StartSegment(tr, start)
		EndDatastoreSegment(EndDatastoreParams{
			Tracer:       tr,
			Start:        s,
			Now:          start.Add(1 * time.Second),
			Product:      tc.Product,
			Operation:    "SELECT",
			Collection:   "my_table",
			PortPathOrID: portPathOrID,
			Host:         tc.DBHostname,
		})

		expect := strings.Replace(tc.ExpectedMetric,
			tc.SystemHostname, ThisHost, -1)

		metrics := newMetricTable(100, time.Now())
		tr.FinalName = "OtherTransaction/Go/zip"
		tr.IsWeb = false
		MergeBreakdownMetrics(tr, metrics)
		data := []float64{1, 1, 1, 1, 1, 1}
		ExpectMetrics(ExtendValidator(t, tc.Name), metrics, []WantMetric{
			{"Datastore/all", "", true, data},
			{"Datastore/allOther", "", true, data},
			{"Datastore/" + tc.Product + "/all", "", true, data},
			{"Datastore/" + tc.Product + "/allOther", "", true, data},
			{"Datastore/operation/" + tc.Product + "/SELECT", "", false, data},
			{"Datastore/statement/" + tc.Product + "/my_table/SELECT", "", false, data},
			{"Datastore/statement/" + tc.Product + "/my_table/SELECT", tr.FinalName, false, data},
			{expect, "", false, data},
		})
	}
}

func TestGenericSpanEventCreation(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &TxnData{}

	// Enable that which is necessary to generate span events when segments are ended.
	tr.BetterCAT.Sampled = true
	tr.SpanEventsEnabled = true

	t1 := StartSegment(tr, start.Add(1*time.Second))
	EndBasicSegment(tr, t1, start.Add(3*time.Second), "t1")

	// Since a basic segment has just ended, there should be exactly one generic span event in tr.spanEvents[]
	if 1 != len(tr.spanEvents) {
		t.Error(tr.spanEvents)
	}
	if tr.spanEvents[0].Category != spanCategoryGeneric {
		t.Error(tr.spanEvents[0].Category)
	}
}

func TestDatastoreSpanEventCreation(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &TxnData{}

	// Enable that which is necessary to generate span events when segments are ended.
	tr.BetterCAT.Sampled = true
	tr.SpanEventsEnabled = true

	t1 := StartSegment(tr, start.Add(1*time.Second))
	EndDatastoreSegment(EndDatastoreParams{
		Tracer:     tr,
		Start:      t1,
		Now:        start.Add(3 * time.Second),
		Product:    "MySQL",
		Operation:  "SELECT",
		Collection: "my_table",
	})

	// Since a datastore segment has just ended, there should be exactly one datastore span event in tr.spanEvents[]
	if 1 != len(tr.spanEvents) {
		t.Error(tr.spanEvents)
	}
	if tr.spanEvents[0].Category != spanCategoryDatastore {
		t.Error(tr.spanEvents[0].Category)
	}
}

func TestHTTPSpanEventCreation(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &TxnData{}

	// Enable that which is necessary to generate span events when segments are ended.
	tr.BetterCAT.Sampled = true
	tr.SpanEventsEnabled = true

	t1 := StartSegment(tr, start.Add(1*time.Second))
	EndExternalSegment(tr, t1, start.Add(3*time.Second), nil, "", nil)

	// Since an external segment has just ended, there should be exactly one HTTP span event in tr.spanEvents[]
	if 1 != len(tr.spanEvents) {
		t.Error(tr.spanEvents)
	}
	if tr.spanEvents[0].Category != spanCategoryHTTP {
		t.Error(tr.spanEvents[0].Category)
	}
}
