// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"encoding/json"
	"reflect"
	"runtime"
)

// environment describes the application's environment.
type environment struct {
	Compiler string `env:"runtime.Compiler"`
	GOARCH   string `env:"runtime.GOARCH"`
	GOOS     string `env:"runtime.GOOS"`
	Version  string `env:"runtime.Version"`
	NumCPU   int    `env:"runtime.NumCPU"`
}

var (
	// sampleEnvironment is useful for testing.
	sampleEnvironment = environment{
		Compiler: "comp",
		GOARCH:   "arch",
		GOOS:     "goos",
		Version:  "vers",
		NumCPU:   8,
	}
)

// newEnvironment returns a new Environment.
func newEnvironment() environment {
	return environment{
		Compiler: runtime.Compiler,
		GOARCH:   runtime.GOARCH,
		GOOS:     runtime.GOOS,
		Version:  runtime.Version(),
		NumCPU:   runtime.NumCPU(),
	}
}

// MarshalJSON prepares Environment JSON in the format expected by the collector
// during the connect command.
func (e environment) MarshalJSON() ([]byte, error) {
	var arr [][]interface{}

	val := reflect.ValueOf(e)
	numFields := val.NumField()

	arr = make([][]interface{}, numFields)

	for i := 0; i < numFields; i++ {
		v := val.Field(i)
		t := val.Type().Field(i).Tag.Get("env")

		arr[i] = []interface{}{
			t,
			v.Interface(),
		}
	}

	return json.Marshal(arr)
}
