package newrelic

import (
	"fmt"
	"testing"
	"time"
)

func TestConnectBackoff(t *testing.T) {
	attempts := map[int]int{
		0:   15,
		2:   30,
		5:   300,
		6:   300,
		100: 300,
		-5:  300,
	}

	for k, v := range attempts {
		if b := getConnectBackoffTime(k); b != v {
			t.Error(fmt.Sprintf("Invalid connect backoff for attempt #%d:", k), v)
		}
	}
}

func TestNilApplication(t *testing.T) {
	var app *Application
	if txn := app.StartTransaction("name"); txn != nil {
		t.Error(txn)
	}
	if err := app.RecordCustomEvent("myEventType", map[string]interface{}{"zip": "zap"}); nil != err {
		t.Error(err)
	}
	if err := app.RecordCustomMetric("myMetric", 123.45); nil != err {
		t.Error(err)
	}
	if err := app.WaitForConnection(2 * time.Second); nil != err {
		t.Error(err)
	}
	app.Shutdown(2 * time.Second)
}

func TestEmptyApplication(t *testing.T) {
	app := &Application{}
	if txn := app.StartTransaction("name"); txn != nil {
		t.Error(txn)
	}
	if err := app.RecordCustomEvent("myEventType", map[string]interface{}{"zip": "zap"}); nil != err {
		t.Error(err)
	}
	if err := app.RecordCustomMetric("myMetric", 123.45); nil != err {
		t.Error(err)
	}
	if err := app.WaitForConnection(2 * time.Second); nil != err {
		t.Error(err)
	}
	app.Shutdown(2 * time.Second)
}
