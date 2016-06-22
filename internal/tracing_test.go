package internal

import (
	"testing"
	"time"

	"github.com/newrelic/go-agent/api"
	"github.com/newrelic/go-agent/api/datastore"
)

func TestCreateParseToken(t *testing.T) {
	token := createToken(1122334, 123456789012)
	depth, stamp := parseToken(token)
	if depth != 1122334 {
		t.Error(depth)
	}
	if stamp != 123456789012 {
		t.Error(stamp)
	}
}

func TestStartEndSegment(t *testing.T) {
	start = time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)

	tr := &tracer{}
	token := startSegment(tr, start)
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
	if end.start != start {
		t.Error(end.start, start)
	}
	if end.stop != stop {
		t.Error(end.stop, stop)
	}
}

func TestTracerRealloc(t *testing.T) {
	max := 3 * startingStackDepthAlloc
	start = time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	now := start
	tokenStack := make([]api.Token, max)

	tr := &tracer{}
	for i := 0; i < max; i++ {
		tokenStack[i] = startSegment(tr, now)
		now = now.Add(time.Second)
	}

	for i := max - 1; i >= 0; i-- {
		now = now.Add(time.Second)
		end := endSegment(tr, tokenStack[i], now)

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
		if end.start != expectStart {
			t.Error(end.start, expectStart)
		}
		expectStop := expectStart.Add(expectDuration)
		if end.stop != expectStop {
			t.Error(end.stop, expectStop)
		}
	}
	rootChildren := time.Duration(2*max) * time.Second
	if tr.children != rootChildren {
		t.Error(tr.children, rootChildren)
	}
}

func TestMultipleChildren(t *testing.T) {
	start = time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &tracer{}

	t1 := startSegment(tr, start.Add(1*time.Second))
	t2 := startSegment(tr, start.Add(2*time.Second))
	end2 := endSegment(tr, t2, start.Add(3*time.Second))
	t3 := startSegment(tr, start.Add(4*time.Second))
	end3 := endSegment(tr, t3, start.Add(5*time.Second))
	end1 := endSegment(tr, t1, start.Add(6*time.Second))
	t4 := startSegment(tr, start.Add(7*time.Second))
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
	if tr.children != 6*time.Second {
		t.Error(tr.children)
	}
}

func TestInvalidToken(t *testing.T) {
	start = time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &tracer{}

	end := endSegment(tr, invalidToken, start.Add(1*time.Second))
	if end.valid {
		t.Error(end.valid)
	}
	startSegment(tr, start.Add(2*time.Second))
	end = endSegment(tr, invalidToken, start.Add(3*time.Second))
	if end.valid {
		t.Error(end.valid)
	}
}

func TestSegmentAlreadyEnded(t *testing.T) {
	start = time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &tracer{}

	t1 := startSegment(tr, start.Add(1*time.Second))
	end := endSegment(tr, t1, start.Add(2*time.Second))
	if !end.valid {
		t.Error(end)
	}
	end = endSegment(tr, t1, start.Add(3*time.Second))
	if end.valid {
		t.Error(end)
	}
}

func TestSegmentBadToken(t *testing.T) {
	start = time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &tracer{}

	t1 := startSegment(tr, start.Add(1*time.Second))
	t1++
	end := endSegment(tr, t1, start.Add(2*time.Second))
	if end.valid {
		t.Error(end)
	}
}

func TestSegmentOutOfOrder(t *testing.T) {
	start = time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &tracer{}

	t1 := startSegment(tr, start.Add(1*time.Second))
	t2 := startSegment(tr, start.Add(2*time.Second))
	t3 := startSegment(tr, start.Add(3*time.Second))
	end2 := endSegment(tr, t2, start.Add(4*time.Second))
	end3 := endSegment(tr, t3, start.Add(5*time.Second))
	t4 := startSegment(tr, start.Add(6*time.Second))
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

func TestSegmentBasic(t *testing.T) {
	start = time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &tracer{}

	t1 := startSegment(tr, start.Add(1*time.Second))
	t2 := startSegment(tr, start.Add(2*time.Second))
	endBasicSegment(tr, t2, start.Add(3*time.Second), "f2")
	endBasicSegment(tr, t1, start.Add(4*time.Second), "f1")
	t3 := startSegment(tr, start.Add(5*time.Second))
	endBasicSegment(tr, t3, start.Add(6*time.Second), "f1")
	t4 := startSegment(tr, start.Add(7*time.Second))
	endBasicSegment(tr, t4+1, start.Add(8*time.Second), "invalid-token")

	metrics := newMetricTable(100, time.Now())
	scope := "WebTransaction/Go/zip"
	mergeBreakdownMetrics(tr, metrics, scope, true)
	expectMetrics(t, metrics, []WantMetric{
		{"Custom/f1", "", false, []float64{2, 4, 3, 1, 3, 10}},
		{"Custom/f2", "", false, []float64{1, 1, 1, 1, 1, 1}},
		{"Custom/f1", scope, false, []float64{2, 4, 3, 1, 3, 10}},
		{"Custom/f2", scope, false, []float64{1, 1, 1, 1, 1, 1}},
	})
}

func TestSegmentExternal(t *testing.T) {
	start = time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &tracer{}

	t1 := startSegment(tr, start.Add(1*time.Second))
	t2 := startSegment(tr, start.Add(2*time.Second))
	endExternalSegment(tr, t2, start.Add(3*time.Second), "")
	endExternalSegment(tr, t1, start.Add(4*time.Second), "f1.com")
	t3 := startSegment(tr, start.Add(5*time.Second))
	endExternalSegment(tr, t3, start.Add(6*time.Second), "f1.com")
	t4 := startSegment(tr, start.Add(7*time.Second))
	endExternalSegment(tr, t4+1, start.Add(8*time.Second), "invalid-token.com")

	if tr.externalCallCount != 3 {
		t.Error(tr.externalCallCount)
	}
	if tr.externalDuration != 5*time.Second {
		t.Error(tr.externalDuration)
	}
	metrics := newMetricTable(100, time.Now())
	scope := "WebTransaction/Go/zip"
	mergeBreakdownMetrics(tr, metrics, scope, true)
	expectMetrics(t, metrics, []WantMetric{
		{"External/all", "", true, []float64{3, 5, 4, 1, 3, 11}},
		{"External/allWeb", "", true, []float64{3, 5, 4, 1, 3, 11}},
		{"External/f1.com/all", "", false, []float64{2, 4, 3, 1, 3, 10}},
		{"External/unknown/all", "", false, []float64{1, 1, 1, 1, 1, 1}},
		{"External/f1.com/all", scope, false, []float64{2, 4, 3, 1, 3, 10}},
		{"External/unknown/all", scope, false, []float64{1, 1, 1, 1, 1, 1}},
	})

	metrics = newMetricTable(100, time.Now())
	scope = "OtherTransaction/Go/zip"
	mergeBreakdownMetrics(tr, metrics, scope, false)
	expectMetrics(t, metrics, []WantMetric{
		{"External/all", "", true, []float64{3, 5, 4, 1, 3, 11}},
		{"External/allOther", "", true, []float64{3, 5, 4, 1, 3, 11}},
		{"External/f1.com/all", "", false, []float64{2, 4, 3, 1, 3, 10}},
		{"External/unknown/all", "", false, []float64{1, 1, 1, 1, 1, 1}},
		{"External/f1.com/all", scope, false, []float64{2, 4, 3, 1, 3, 10}},
		{"External/unknown/all", scope, false, []float64{1, 1, 1, 1, 1, 1}},
	})
}

func TestSegmentDatastore(t *testing.T) {
	start = time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	tr := &tracer{}

	allDatastoreFields := datastore.Segment{
		Product:    datastore.MySQL,
		Operation:  "SELECT",
		Collection: "my_table",
	}
	missingCollection := datastore.Segment{
		Product:   datastore.MySQL,
		Operation: "SELECT",
	}
	emptyDatastore := datastore.Segment{}
	invalidTokenOperation := datastore.Segment{
		Product:   datastore.MySQL,
		Operation: "invalid-token",
	}

	t1 := startSegment(tr, start.Add(1*time.Second))
	t2 := startSegment(tr, start.Add(2*time.Second))
	endDatastoreSegment(tr, t2, start.Add(3*time.Second), allDatastoreFields)
	endDatastoreSegment(tr, t1, start.Add(4*time.Second), missingCollection)
	t3 := startSegment(tr, start.Add(5*time.Second))
	endDatastoreSegment(tr, t3, start.Add(6*time.Second), missingCollection)
	t4 := startSegment(tr, start.Add(7*time.Second))
	endDatastoreSegment(tr, t4+1, start.Add(8*time.Second), invalidTokenOperation)
	t5 := startSegment(tr, start.Add(9*time.Second))
	endDatastoreSegment(tr, t5, start.Add(10*time.Second), emptyDatastore)

	if tr.datastoreCallCount != 4 {
		t.Error(tr.datastoreCallCount)
	}
	if tr.datastoreDuration != 6*time.Second {
		t.Error(tr.datastoreDuration)
	}
	metrics := newMetricTable(100, time.Now())
	scope := "WebTransaction/Go/zip"
	mergeBreakdownMetrics(tr, metrics, scope, true)
	expectMetrics(t, metrics, []WantMetric{
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
	})

	metrics = newMetricTable(100, time.Now())
	scope = "OtherTransaction/Go/zip"
	mergeBreakdownMetrics(tr, metrics, scope, false)
	expectMetrics(t, metrics, []WantMetric{
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
	})
}
