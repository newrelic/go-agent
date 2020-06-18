// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/newrelic/go-agent/v3/integrations/nrpkgerrors"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
	"github.com/pkg/errors"
)

type sampleError string

func (e sampleError) Error() string {
	return string(e)
}

func alpha() error {
	return errors.WithStack(sampleError("alpha is the cause"))
}

func beta() error {
	return errors.WithStack(alpha())
}

func gamma() error {
	return errors.Wrap(beta(), "gamma was involved")
}

func main() {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("pkg/errors App"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDebugLogger(os.Stdout),
	)
	if nil != err {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := app.WaitForConnection(5 * time.Second); nil != err {
		fmt.Println(err)
	}

	txn := app.StartTransaction("has-error")
	e := gamma()
	txn.NoticeError(nrpkgerrors.Wrap(e))
	txn.End()

	app.Shutdown(10 * time.Second)
}
