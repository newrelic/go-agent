// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"encoding/json"
	"regexp"
	"runtime"
	"runtime/debug"
	"testing"

	"github.com/newrelic/go-agent/v3/internal"
)

func TestMarshalEnvironment(t *testing.T) {
	js, err := json.Marshal(&sampleEnvironment)
	if nil != err {
		t.Fatal(err)
	}
	expect := internal.CompactJSONString(`[
		["runtime.NumCPU",8],
		["runtime.Compiler","comp"],
		["runtime.GOARCH","arch"],
		["runtime.GOOS","goos"],
		["runtime.Version","vers"],
		["Modules",null]]`)
	if string(js) != expect {
		t.Fatal(string(js))
	}
}

func TestEnvironmentFields(t *testing.T) {
	env := newEnvironment(nil)
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
	if env.Modules != nil {
		t.Error(env.Modules, nil)
	}
}

func TestModuleDependency(t *testing.T) {
	cfg := config{Config: defaultConfig()}

	// check that the default is to be enabled
	if !cfg.ModuleDependencyMetrics.Enabled {
		t.Error("MDM should be enabled, was", cfg.ModuleDependencyMetrics.Enabled)
	}

	// if disabled, we shouldn't get any data
	cfg.ModuleDependencyMetrics.Enabled = false
	env := newEnvironment(&cfg)
	if env.Modules != nil && len(env.Modules) != 0 {
		t.Error("MDM module list not empty:", env.Modules)
	}

	// enabled, and we should see our list of modules reported.
	// first, get the list of modules we should expect to see.
	// of course, we can't do that from a unit test, so we'll mock up a set
	// of modules to at least check that the various options work.
	expectedModules := make(map[string]*debug.Module)
	mockedModuleList := []*debug.Module{
		&debug.Module{Path: "example/path/to/module", Version: "v1.2.3"},
		&debug.Module{Path: "github.com/another/module", Version: "v0.1.2"},
		&debug.Module{Path: "some/development/module", Version: "(develop)"},
	}
	for _, module := range mockedModuleList {
		expectedModules[module.Path] = module
	}

	cfg.ModuleDependencyMetrics.Enabled = true
	env = newEnvironment(&cfg)
	env.Modules = injectDependencyModuleList(&cfg, mockedModuleList)
	checkModuleListsMatch(t, expectedModules, env.Modules, "full module list")

	// try to elide some modules now
	cfg.ModuleDependencyMetrics.IgnoredPrefixes = []string{"github.com"}
	env = newEnvironment(&cfg)
	env.Modules = injectDependencyModuleList(&cfg, mockedModuleList)
	delete(expectedModules, "github.com/another/module")
	checkModuleListsMatch(t, expectedModules, env.Modules, "reduced module list")

	// more...
	cfg.ModuleDependencyMetrics.IgnoredPrefixes = []string{"github.com", "exam"}
	env = newEnvironment(&cfg)
	env.Modules = injectDependencyModuleList(&cfg, mockedModuleList)
	delete(expectedModules, "example/path/to/module")
	checkModuleListsMatch(t, expectedModules, env.Modules, "reduced module list")
}

func checkModuleListsMatch(t *testing.T, expected map[string]*debug.Module, actual []string, message string) {
	if expected == nil {
		t.Error(message, "expected list is nil")
	}
	if len(expected) > 0 && actual == nil {
		t.Error(message, "actual list is nil")
	}
	if len(expected) != len(actual) {
		t.Error(message, "actual list has", len(actual), "module(s) but expected", len(expected))
	}

	modulePattern := regexp.MustCompile(`^(.+?)\((.+)\)$`)
	checked := make(map[string]bool)
	for path, _ := range expected {
		checked[path] = false
	}

	for i, actualName := range actual {
		matches := modulePattern.FindStringSubmatch(actualName)
		if matches == nil || len(matches) != 3 {
			t.Errorf("%s: actual module element #%d could not be parsed: \"%v\"", message, i, actualName)
			continue
		}

		if module, present := expected[matches[1]]; present {
			if matches[1] != module.Path {
				t.Errorf("%s: actual module element #%d \"%v\" mismatch to path \"%v\" which really shouldn't be possible",
					message, i, matches[1], module.Path)
				continue
			}
			if matches[2] != module.Version {
				t.Errorf("%s: actual module element #%d \"%v\" version \"%v\" mismatch to expected version \"%v\"",
					message, i, matches[1], matches[2], module.Version)
				continue
			}
			if checked[matches[1]] {
				t.Errorf("%s: actual module element #%d \"%v\" was already seen earlier in the module list",
					message, i, matches[1])
				continue
			}
			checked[matches[1]] = true
		} else {
			t.Errorf("%s: actual module element #%d \"%v\" unexpected", message, i, matches[1])
		}
	}

	for expectedName, wasChecked := range checked {
		if !wasChecked {
			t.Errorf("%s: did not see expected module \"%v\"", message, expectedName)
		}
	}
}
