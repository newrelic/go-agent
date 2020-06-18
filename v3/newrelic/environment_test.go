// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"encoding/json"
	"runtime"
	"testing"

	"github.com/newrelic/go-agent/v3/internal"
)

func TestMarshalEnvironment(t *testing.T) {
	js, err := json.Marshal(&sampleEnvironment)
	if nil != err {
		t.Fatal(err)
	}
	expect := internal.CompactJSONString(`[
		["runtime.Compiler","comp"],
		["runtime.GOARCH","arch"],
		["runtime.GOOS","goos"],
		["runtime.Version","vers"],
		["runtime.NumCPU",8]]`)
	if string(js) != expect {
		t.Fatal(string(js))
	}
}

func TestEnvironmentFields(t *testing.T) {
	env := newEnvironment()
	if env.Compiler != runtime.Compiler {
		t.Error(env.Compiler, runtime.Compiler)
	}
	if env.GOARCH != runtime.GOARCH {
		t.Error(env.GOARCH, runtime.GOARCH)
	}
	if env.GOOS != runtime.GOOS {
		t.Error(env.GOOS, runtime.GOOS)
	}
	if env.Version != runtime.Version() {
		t.Error(env.Version, runtime.Version())
	}
	if env.NumCPU != runtime.NumCPU() {
		t.Error(env.NumCPU, runtime.NumCPU())
	}
}
