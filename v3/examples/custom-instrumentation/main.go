// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// An application that illustrates Distributed Tracing with custom
// instrumentation.
//
// This application simulates simple inter-process communication between a
// calling and a called process.
//
// Invoked without arguments, the application acts as a calling process;
// invoked with one argument representing a payload, it acts as a called
// process. The calling process creates a payload, starts a called process and
// passes on the payload. The calling process waits until the called process is
// done and then terminates. Thus to start both processes, only a single
// invocation of the application (without any arguments) is needed.
package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)

func called(app *newrelic.Application, payload string) {
	txn := app.StartTransaction("called-txn")
	defer txn.End()

	// Accept the payload that was passed on the command line.
	hdrs := http.Header{}
	hdrs.Set(newrelic.DistributedTraceNewRelicHeader, payload)
	txn.AcceptDistributedTraceHeaders(newrelic.TransportOther, hdrs)
	time.Sleep(1 * time.Second)
}

func calling(app *newrelic.Application) {
	txn := app.StartTransaction("calling-txn")
	defer txn.End()

	// Create a payload, start the called process and pass the payload.
	hdrs := http.Header{}
	txn.InsertDistributedTraceHeaders(hdrs)
	cmd := exec.Command(os.Args[0], hdrs.Get(newrelic.DistributedTraceNewRelicHeader))
	cmd.Start()

	// Wait until the called process is done, then exit.
	cmd.Wait()
	time.Sleep(1 * time.Second)
}

func makeApplication(name string) (*newrelic.Application, error) {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName(name),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDebugLogger(os.Stdout),
		newrelic.ConfigDistributedTracerEnabled(true),
	)
	if nil != err {
		return nil, err
	}

	// Wait for the application to connect.
	if err = app.WaitForConnection(5 * time.Second); nil != err {
		return nil, err
	}

	return app, nil
}

func main() {
	// Calling processes have no command line arguments, called processes
	// have one command line argument (the payload).
	isCalled := (len(os.Args) > 1)

	// Initialize the application name.
	name := "Go Custom Instrumentation"
	if isCalled {
		name += " Called"
	} else {
		name += " Calling"
	}

	// Initialize the application.
	app, err := makeApplication(name)
	if nil != err {
		fmt.Println(err)
		os.Exit(1)
	}

	// Run calling/called routines.
	if isCalled {
		payload := os.Args[1]
		called(app, payload)
	} else {
		calling(app)
	}

	// Shut down the application to flush data to New Relic.
	app.Shutdown(10 * time.Second)
}
