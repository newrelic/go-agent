// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/newrelic/go-agent/v3/integrations/nrlogrus"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Logrus App"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		nrlogrus.ConfigStandardLogger(),
	)

	if nil != err {
		fmt.Println(err)
		os.Exit(1)
	}

	http.HandleFunc(newrelic.WrapHandleFunc(app, "/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello world")
	}))

	http.ListenAndServe(":8000", nil)
}
