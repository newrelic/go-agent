// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"errors"
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
	app.RecordCustomEvent("myEventType", map[string]interface{}{"zip": "zap"})
	app.RecordCustomMetric("myMetric", 123.45)
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
	app.RecordCustomEvent("myEventType", map[string]interface{}{"zip": "zap"})
	app.RecordCustomMetric("myMetric", 123.45)
	if err := app.WaitForConnection(2 * time.Second); nil != err {
		t.Error(err)
	}
	app.Shutdown(2 * time.Second)
}

func TestConfigOptionError(t *testing.T) {
	err := errors.New("myError")
	app, got := NewApplication(
		nil, // nil config options should be ignored
		func(cfg *Config) {
			cfg.Error = err
		},
		func(cfg *Config) {
			t.Fatal("this config option should not be run")
		},
	)
	if err != got {
		t.Error("config option not returned", err, got)
	}
	if app != nil {
		t.Error("app not nil")
	}
}
