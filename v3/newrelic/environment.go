// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"encoding/json"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
)

// environment describes the application's environment.
type environment struct {
	NumCPU   int            `env:"runtime.NumCPU"`
	Compiler string         `env:"runtime.Compiler"`
	GOARCH   string         `env:"runtime.GOARCH"`
	GOOS     string         `env:"runtime.GOOS"`
	Version  string         `env:"runtime.Version"`
	Modules  []debug.Module `env:"Modules"`
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
func newEnvironment(c *config) environment {
	return environment{
		Compiler: runtime.Compiler,
		GOARCH:   runtime.GOARCH,
		GOOS:     runtime.GOOS,
		Version:  runtime.Version(),
		NumCPU:   runtime.NumCPU(),
		Modules:  getDependencyModuleList(c),
	}
}

func getDependencyModuleList(c *config) []debug.Module {
	var modList []debug.Module

	if c != nil && c.ModuleDependencyMetrics.Enabled {
		info, ok := debug.ReadBuildInfo()
		if info != nil && ok {
			for _, module := range info.Deps {
				if module != nil && includeModule(module.Path, c.ModuleDependencyMetrics.IgnoredPrefixes) {
					modList = append(modList, *module)
				}
			}
		}
	}
	return modList
}

func includeModule(name string, ignoredModulePrefixes []string) bool {
	for _, excluded := range ignoredModulePrefixes {
		if strings.HasPrefix(name, excluded) {
			return false
		}
	}

	return true
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
