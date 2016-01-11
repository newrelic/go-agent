package internal

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTxnEventMarshal(t *testing.T) {
	start := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)

	event := CreateTxnEvent(ApdexNone, "myName", 2*time.Second, start)
	js, err := json.Marshal(event)
	if nil != err {
		t.Error(err)
	}
	if string(js) != `[{"type":"Transaction","name":"myName","timestamp":1.41713646e+09,"duration":2},{},{}]` {
		t.Error(string(js))
	}

	event = CreateTxnEvent(ApdexFailing, "myName", 2*time.Second, start)
	js, err = json.Marshal(event)
	if nil != err {
		t.Error(err)
	}
	if string(js) != `[{"type":"Transaction","name":"myName","timestamp":1.41713646e+09,"duration":2,"nr.apdexPerfZone":"F"},{},{}]` {
		t.Error(string(js))
	}
}
