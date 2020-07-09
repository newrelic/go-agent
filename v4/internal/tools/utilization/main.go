// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/newrelic/go-agent/v3/internal/utilization"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)

func main() {
	util := utilization.Gather(utilization.Config{
		DetectAWS:        true,
		DetectAzure:      true,
		DetectDocker:     true,
		DetectPCF:        true,
		DetectGCP:        true,
		DetectKubernetes: true,
	}, newrelic.NewDebugLogger(os.Stdout))

	js, err := json.MarshalIndent(util, "", "\t")
	if err != nil {
		fmt.Printf("%s\n", err)
	} else {
		fmt.Printf("%s\n", js)
	}
}
