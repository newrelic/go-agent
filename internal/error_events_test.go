package internal

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestErrorEventMarshal(t *testing.T) {
	e := txnErrorFromError(errors.New("hello"))
	e.when = time.Date(2014, time.November, 28, 1, 1, 0, 0, time.UTC)
	e.stack = getStackTrace(0)

	event := createErrorEvent(&e, "myName", 3*time.Second)

	js, err := json.Marshal(event)
	if nil != err {
		t.Error(err)
	}
	expect := compactJSONString(`
	[
		{
			"type":"TransactionError",
			"error.class":"*errors.errorString",
			"error.message":"hello",
			"timestamp":1.41713646e+09,
			"transactionName":"myName",
			"duration":3
		},
		{},
		{}
	]`)
	if string(js) != expect {
		t.Error(string(js))
	}
}
