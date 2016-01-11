package internal

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestErrorEventMarshal(t *testing.T) {
	when := time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)

	stack := GetStackTrace(0)
	e := newTxnError(false, errors.New("hello"), stack, when)
	event := CreateErrorEvent(e, "myName", 3*time.Second)

	js, err := json.Marshal(event)
	if nil != err {
		t.Error(err)
	}
	if string(js) != `[{"type":"TransactionError","error.class":"*errors.errorString","error.message":"hello","timestamp":1.41713646e+09,"transactionName":"myName","duration":3},{},{}]` {
		t.Error(string(js))
	}
}
