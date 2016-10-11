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

	tr := &Tracer{}
	token := StartSegment(tr, start)
	stop := start.Add(1 * time.Second)
	end := endSegment(tr, token, stop)
	if !end.valid {
		t.Error(end.valid)
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
}

func TestTracerRealloc(t *testing.T) {
	max := 3 * startingStackDepthAlloc
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	now := start
	startStack := make([]SegmentStartTime, max)

	tr := &Tracer{}
	for i := 0; i < max; i++ {
		startStack[i] = StartSegment(tr, now)
		now = now.Add(time.Second)
	}

	for i := max - 1; i >= 0; i-- {
		now = now.Add(time.Second)
		end := endSegment(tr, startStack[i], now)

		if !end.valid {
			t.Error(end.valid)
		}
		if end.exclusive != 2*time.Second {
			t.Error(end.exclusive)
		}
		expectDuration := time.Duration((max-i)*2) * time.Second
		if end.duration != expectDuration {
			t.Error(end.duration, expectDuration)
		}
		expectStart := start.Add(time.Duration(i) * time.Second)
		if end.start.Time != expectStart {
			t.Error(end.start, expectStart)
		}
		expectStop := expectStart.Add(expectDuration)
		if end.stop.Time != expectStop {
			t.Error(end.stop, expectStop)
		}
	}
	rootChildren := time.Duration(2*max) * time.Second
	children := TracerRootChildren(tr)
	if children != rootChildren {
		t.Error(children, rootChildren)
	}
}

func TestMultipleChildren(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &Tracer{}

	t1 := StartSegment(tr, start.Add(1*time.Second))
	t2 := StartSegment(tr, start.Add(2*time.Second))
	end2 := endSegment(tr, t2, start.Add(3*time.Second))
	t3 := StartSegment(tr, start.Add(4*time.Second))
	end3 := endSegment(tr, t3, start.Add(5*time.Second))
	end1 := endSegment(tr, t1, start.Add(6*time.Second))
	t4 := StartSegment(tr, start.Add(7*time.Second))
	end4 := endSegment(tr, t4, start.Add(8*time.Second))

	if end1.duration != 5*time.Second || end1.exclusive != 3*time.Second {
		t.Error(end1)
	}
	if end2.duration != end2.exclusive || end2.duration != time.Second {
		t.Error(end2)
	}
	if end3.duration != end3.exclusive || end3.duration != time.Second {
		t.Error(end3)
	}
	if end4.duration != end4.exclusive || end4.duration != time.Second {
		t.Error(end4)
	}
	children := TracerRootChildren(tr)
	if children != 6*time.Second {
		t.Error(children)
	}
}

func TestInvalidStart(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &Tracer{}

	end := endSegment(tr, SegmentStartTime{}, start.Add(1*time.Second))
	if end.valid {
		t.Error(end.valid)
	}
	StartSegment(tr, start.Add(2*time.Second))
	end = endSegment(tr, SegmentStartTime{}, start.Add(3*time.Second))
	if end.valid {
		t.Error(end.valid)
	}
}

func TestSegmentAlreadyEnded(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &Tracer{}

	t1 := StartSegment(tr, start.Add(1*time.Second))
	end := endSegment(tr, t1, start.Add(2*time.Second))
	if !end.valid {
		t.Error(end)
	}
	end = endSegment(tr, t1, start.Add(3*time.Second))
	if end.valid {
		t.Error(end)
	}
}

func TestSegmentBadStamp(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &Tracer{}

	t1 := StartSegment(tr, start.Add(1*time.Second))
	t1.Stamp++
	end := endSegment(tr, t1, start.Add(2*time.Second))
	if end.valid {
		t.Error(end)
	}
}

func TestSegmentBadDepth(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &Tracer{}

	t1 := StartSegment(tr, start.Add(1*time.Second))
	t1.Depth++
	end := endSegment(tr, t1, start.Add(2*time.Second))
	if end.valid {
		t.Error(end)
	}
}

func TestSegmentNegativeDepth(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &Tracer{}

	t1 := StartSegment(tr, start.Add(1*time.Second))
	t1.Depth = -1
	end := endSegment(tr, t1, start.Add(2*time.Second))
	if end.valid {
		t.Error(end)
	}
}

func TestSegmentOutOfOrder(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &Tracer{}

	t1 := StartSegment(tr, start.Add(1*time.Second))
	t2 := StartSegment(tr, start.Add(2*time.Second))
	t3 := StartSegment(tr, start.Add(3*time.Second))
	end2 := endSegment(tr, t2, start.Add(4*time.Second))
	end3 := endSegment(tr, t3, start.Add(5*time.Second))
	t4 := StartSegment(tr, start.Add(6*time.Second))
	end4 := endSegment(tr, t4, start.Add(7*time.Second))
	end1 := endSegment(tr, t1, start.Add(8*time.Second))

	if !end1.valid ||
		end1.duration != 7*time.Second ||
		end1.exclusive != 4*time.Second {
		t.Error(end1)
	}
	if !end2.valid || end2.duration != end2.exclusive || end2.duration != 2*time.Second {
		t.Error(end2)
	}
	if end3.valid {
		t.Error(end3)
	}
	if !end4.valid || end4.duration != end4.exclusive || end4.duration != 1*time.Second {
		t.Error(end4)
	}
}

//                                          |-t3-|    |-t4-|
//                           |-t2-|    |-never-finished----------
//            |-t1-|    |--never-finished------------------------
//       |-------alpha------------------------------------------|
//  0    1    2    3    4    5    6    7    8    9    10   11   12
func TestLostChildren(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &Tracer{}

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
	scope := "WebTransaction/Go/zip"
	MergeBreakdownMetrics(tr, metrics, scope, true)
	ExpectMetrics(t, metrics, []WantMetric{
		{"Custom/alpha", "", false, []float64{1, 11, 7, 11, 11, 121}},
		{"Custom/t1", "", false, []float64{1, 1, 1, 1, 1, 1}},
		{"Custom/t2", "", false, []float64{1, 1, 1, 1, 1, 1}},
		{"Custom/t3", "", false, []float64{1, 1, 1, 1, 1, 1}},
		{"Custom/t4", "", false, []float64{1, 1, 1, 1, 1, 1}},
		{"Custom/alpha", scope, false, []float64{1, 11, 7, 11, 11, 121}},
		{"Custom/t1", scope, false, []float64{1, 1, 1, 1, 1, 1}},
		{"Custom/t2", scope, false, []float64{1, 1, 1, 1, 1, 1}},
		{"Custom/t3", scope, false, []float64{1, 1, 1, 1, 1, 1}},
		{"Custom/t4", scope, false, []float64{1, 1, 1, 1, 1, 1}},
	})
}

//                                          |-t3-|    |-t4-|
//                           |-t2-|    |-never-finished----------
//            |-t1-|    |--never-finished------------------------
//  |-------root-------------------------------------------------
//  0    1    2    3    4    5    6    7    8    9    10   11   12
func TestLostChildrenRoot(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &Tracer{}

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
	scope := "WebTransaction/Go/zip"
	MergeBreakdownMetrics(tr, metrics, scope, true)
	ExpectMetrics(t, metrics, []WantMetric{
		{"Custom/t1", "", false, []float64{1, 1, 1, 1, 1, 1}},
		{"Custom/t2", "", false, []float64{1, 1, 1, 1, 1, 1}},
		{"Custom/t3", "", false, []float64{1, 1, 1, 1, 1, 1}},
		{"Custom/t4", "", false, []float64{1, 1, 1, 1, 1, 1}},
		{"Custom/t1", scope, false, []float64{1, 1, 1, 1, 1, 1}},
		{"Custom/t2", scope, false, []float64{1, 1, 1, 1, 1, 1}},
		{"Custom/t3", scope, false, []float64{1, 1, 1, 1, 1, 1}},
		{"Custom/t4", scope, false, []float64{1, 1, 1, 1, 1, 1}},
	})
}

func TestSegmentBasic(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &Tracer{}

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
	scope := "WebTransaction/Go/zip"
	MergeBreakdownMetrics(tr, metrics, scope, true)
	ExpectMetrics(t, metrics, []WantMetric{
		{"Custom/t1", "", false, []float64{2, 4, 3, 1, 3, 10}},
		{"Custom/t2", "", false, []float64{1, 1, 1, 1, 1, 1}},
		{"Custom/t3", "", false, []float64{1, 2, 2, 2, 2, 4}},
		{"Custom/t1", scope, false, []float64{2, 4, 3, 1, 3, 10}},
		{"Custom/t2", scope, false, []float64{1, 1, 1, 1, 1, 1}},
		{"Custom/t3", scope, false, []float64{1, 2, 2, 2, 2, 4}},
	})
}

func parseURL(raw string) *url.URL {
	u, _ := url.Parse(raw)
	return u
}

func TestSegmentExternal(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &Tracer{}

	t1 := StartSegment(tr, start.Add(1*time.Second))
	t2 := StartSegment(tr, start.Add(2*time.Second))
	EndExternalSegment(tr, t2, start.Add(3*time.Second), nil)
	EndExternalSegment(tr, t1, start.Add(4*time.Second), parseURL("http://f1.com"))
	t3 := StartSegment(tr, start.Add(5*time.Second))
	EndExternalSegment(tr, t3, start.Add(6*time.Second), parseURL("http://f1.com"))
	t4 := StartSegment(tr, start.Add(7*time.Second))
	t4.Stamp++
	EndExternalSegment(tr, t4, start.Add(8*time.Second), parseURL("http://invalid-token.com"))

	if tr.externalCallCount != 3 {
		t.Error(tr.externalCallCount)
	}
	if tr.externalDuration != 5*time.Second {
		t.Error(tr.externalDuration)
	}
	metrics := newMetricTable(100, time.Now())
	scope := "WebTransaction/Go/zip"
	MergeBreakdownMetrics(tr, metrics, scope, true)
	ExpectMetrics(t, metrics, []WantMetric{
		{"External/all", "", true, []float64{3, 5, 4, 1, 3, 11}},
		{"External/allWeb", "", true, []float64{3, 5, 4, 1, 3, 11}},
		{"External/f1.com/all", "", false, []float64{2, 4, 3, 1, 3, 10}},
		{"External/unknown/all", "", false, []float64{1, 1, 1, 1, 1, 1}},
		{"External/f1.com/all", scope, false, []float64{2, 4, 3, 1, 3, 10}},
		{"External/unknown/all", scope, false, []float64{1, 1, 1, 1, 1, 1}},
	})

	metrics = newMetricTable(100, time.Now())
	scope = "OtherTransaction/Go/zip"
	MergeBreakdownMetrics(tr, metrics, scope, false)
	ExpectMetrics(t, metrics, []WantMetric{
		{"External/all", "", true, []float64{3, 5, 4, 1, 3, 11}},
		{"External/allOther", "", true, []float64{3, 5, 4, 1, 3, 11}},
		{"External/f1.com/all", "", false, []float64{2, 4, 3, 1, 3, 10}},
		{"External/unknown/all", "", false, []float64{1, 1, 1, 1, 1, 1}},
		{"External/f1.com/all", scope, false, []float64{2, 4, 3, 1, 3, 10}},
		{"External/unknown/all", scope, false, []float64{1, 1, 1, 1, 1, 1}},
	})
}

func TestSegmentDatastore(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &Tracer{}

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
	scope := "WebTransaction/Go/zip"
	MergeBreakdownMetrics(tr, metrics, scope, true)
	ExpectMetrics(t, metrics, []WantMetric{
		{"Datastore/all", "", true, []float64{4, 6, 5, 1, 3, 12}},
		{"Datastore/allWeb", "", true, []float64{4, 6, 5, 1, 3, 12}},
		{"Datastore/MySQL/all", "", true, []float64{3, 5, 4, 1, 3, 11}},
		{"Datastore/MySQL/allWeb", "", true, []float64{3, 5, 4, 1, 3, 11}},
		{"Datastore/Unknown/all", "", true, []float64{1, 1, 1, 1, 1, 1}},
		{"Datastore/Unknown/allWeb", "", true, []float64{1, 1, 1, 1, 1, 1}},
		{"Datastore/operation/MySQL/SELECT", "", false, []float64{3, 5, 4, 1, 3, 11}},
		{"Datastore/operation/MySQL/SELECT", scope, false, []float64{2, 4, 3, 1, 3, 10}},
		{"Datastore/operation/Unknown/other", "", false, []float64{1, 1, 1, 1, 1, 1}},
		{"Datastore/operation/Unknown/other", scope, false, []float64{1, 1, 1, 1, 1, 1}},
		{"Datastore/statement/MySQL/my_table/SELECT", "", false, []float64{1, 1, 1, 1, 1, 1}},
		{"Datastore/statement/MySQL/my_table/SELECT", scope, false, []float64{1, 1, 1, 1, 1, 1}},
		{"Datastore/instance/MySQL/unknown/unknown", "", false, []float64{3, 5, 4, 1, 3, 11}},
		{"Datastore/instance/Unknown/unknown/unknown", "", false, []float64{1, 1, 1, 1, 1, 1}},
	})

	metrics = newMetricTable(100, time.Now())
	scope = "OtherTransaction/Go/zip"
	MergeBreakdownMetrics(tr, metrics, scope, false)
	ExpectMetrics(t, metrics, []WantMetric{
		{"Datastore/all", "", true, []float64{4, 6, 5, 1, 3, 12}},
		{"Datastore/allOther", "", true, []float64{4, 6, 5, 1, 3, 12}},
		{"Datastore/MySQL/all", "", true, []float64{3, 5, 4, 1, 3, 11}},
		{"Datastore/MySQL/allOther", "", true, []float64{3, 5, 4, 1, 3, 11}},
		{"Datastore/Unknown/all", "", true, []float64{1, 1, 1, 1, 1, 1}},
		{"Datastore/Unknown/allOther", "", true, []float64{1, 1, 1, 1, 1, 1}},
		{"Datastore/operation/MySQL/SELECT", "", false, []float64{3, 5, 4, 1, 3, 11}},
		{"Datastore/operation/MySQL/SELECT", scope, false, []float64{2, 4, 3, 1, 3, 10}},
		{"Datastore/operation/Unknown/other", "", false, []float64{1, 1, 1, 1, 1, 1}},
		{"Datastore/operation/Unknown/other", scope, false, []float64{1, 1, 1, 1, 1, 1}},
		{"Datastore/statement/MySQL/my_table/SELECT", "", false, []float64{1, 1, 1, 1, 1, 1}},
		{"Datastore/statement/MySQL/my_table/SELECT", scope, false, []float64{1, 1, 1, 1, 1, 1}},
		{"Datastore/instance/MySQL/unknown/unknown", "", false, []float64{3, 5, 4, 1, 3, 11}},
		{"Datastore/instance/Unknown/unknown/unknown", "", false, []float64{1, 1, 1, 1, 1, 1}},
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

		tr := &Tracer{}
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
			tc.SystemHostname, thisHost, -1)

		metrics := newMetricTable(100, time.Now())
		scope := "OtherTransaction/Go/zip"
		MergeBreakdownMetrics(tr, metrics, scope, false)
		data := []float64{1, 1, 1, 1, 1, 1}
		ExpectMetrics(ExtendValidator(t, tc.Name), metrics, []WantMetric{
			{"Datastore/all", "", true, data},
			{"Datastore/allOther", "", true, data},
			{"Datastore/" + tc.Product + "/all", "", true, data},
			{"Datastore/" + tc.Product + "/allOther", "", true, data},
			{"Datastore/operation/" + tc.Product + "/SELECT", "", false, data},
			{"Datastore/statement/" + tc.Product + "/my_table/SELECT", "", false, data},
			{"Datastore/statement/" + tc.Product + "/my_table/SELECT", scope, false, data},
			{expect, "", false, data},
		})
	}
}
