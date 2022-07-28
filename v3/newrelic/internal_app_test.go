// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/newrelic/go-agent/v3/internal"
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

const (
	sampleAppName = "my app"
)

// expectApp combines Application and Expect, for use in validating data in test apps
type expectApplication struct {
	internal.Expect
	*Application
}

// newTestApp creates an ExpectApp with the given ConnectReply function and Config function
func newTestApp(replyfn func(*internal.ConnectReply), cfgFn ...ConfigOption) expectApplication {
	cfgFn = append(cfgFn,
		func(cfg *Config) {
			// Prevent spawning app goroutines in tests.
			if !cfg.ServerlessMode.Enabled {
				cfg.Enabled = false
			}
		},
		ConfigAppName(sampleAppName),
		ConfigLicense(testLicenseKey),
	)

	app, err := NewApplication(cfgFn...)
	if nil != err {
		panic(err)
	}

	internal.HarvestTesting(app.Private, replyfn)

	return expectApplication{
		Expect:      app.Private.(internal.Expect),
		Application: app,
	}
}

const (
	testEntityGUID = "testEntityGUID123"
)

var sampleEverythingReplyFn = func(reply *internal.ConnectReply) {
	reply.SetSampleEverything()
	reply.EntityGUID = testEntityGUID
}

var configTestAppLogFn = func(cfg *Config) {
	cfg.Enabled = false
	cfg.ApplicationLogging.Enabled = true
	cfg.ApplicationLogging.Forwarding.Enabled = true
	cfg.ApplicationLogging.Metrics.Enabled = true
}

func TestRecordLog(t *testing.T) {
	testApp := newTestApp(
		sampleEverythingReplyFn,
		configTestAppLogFn,
	)

	time := int64(timeToUnixMilliseconds(time.Now()))

	testApp.Application.RecordLog(LogData{
		Severity:  "Debug",
		Message:   "Test Message",
		Timestamp: time,
	})

	testApp.ExpectLogEvents(t, []internal.WantLog{
		{
			Severity:  "Debug",
			Message:   "Test Message",
			Timestamp: time,
		},
	})
}
