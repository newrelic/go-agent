// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"encoding/json"
	"testing"
)

func testBool(t *testing.T, name string, expected, got bool) {
	if expected != got {
		t.Errorf("%v: expected=%v got=%v", name, expected, got)
	}
}

func TestSecurityPoliciesPresent(t *testing.T) {
	inputJSON := []byte(`{
		"record_sql":                    { "enabled": false, "required": false },
	        "attributes_include":            { "enabled": false, "required": false },
	        "allow_raw_exception_messages":  { "enabled": false, "required": false },
	        "custom_events":                 { "enabled": false, "required": false },
	        "custom_parameters":             { "enabled": false, "required": false },
	        "custom_instrumentation_editor": { "enabled": false, "required": false },
	        "message_parameters":            { "enabled": false, "required": false },
	        "job_arguments":                 { "enabled": false, "required": false }
	}`)
	var policies SecurityPolicies
	err := json.Unmarshal(inputJSON, &policies)
	if nil != err {
		t.Fatal(err)
	}
	connectJSON, err := json.Marshal(policies)
	if nil != err {
		t.Fatal(err)
	}
	expectJSON := CompactJSONString(`{
		"record_sql":                      { "enabled": false },
		"attributes_include":              { "enabled": false },
		"allow_raw_exception_messages":    { "enabled": false },
		"custom_events":                   { "enabled": false },
		"custom_parameters":               { "enabled": false }
	}`)
	if string(connectJSON) != expectJSON {
		t.Error(string(connectJSON), expectJSON)
	}
	testBool(t, "PointerIfPopulated", true, nil != policies.PointerIfPopulated())
	testBool(t, "RecordSQLEnabled", false, policies.RecordSQL.Enabled())
	testBool(t, "AttributesIncludeEnabled", false, policies.AttributesInclude.Enabled())
	testBool(t, "AllowRawExceptionMessages", false, policies.AllowRawExceptionMessages.Enabled())
	testBool(t, "CustomEventsEnabled", false, policies.CustomEvents.Enabled())
	testBool(t, "CustomParametersEnabled", false, policies.CustomParameters.Enabled())
}

func TestNilSecurityPolicies(t *testing.T) {
	var policies SecurityPolicies
	testBool(t, "PointerIfPopulated", false, nil != policies.PointerIfPopulated())
	testBool(t, "RecordSQLEnabled", true, policies.RecordSQL.Enabled())
	testBool(t, "AttributesIncludeEnabled", true, policies.AttributesInclude.Enabled())
	testBool(t, "AllowRawExceptionMessages", true, policies.AllowRawExceptionMessages.Enabled())
	testBool(t, "CustomEventsEnabled", true, policies.CustomEvents.Enabled())
	testBool(t, "CustomParametersEnabled", true, policies.CustomParameters.Enabled())
}

func TestUnknownRequiredPolicy(t *testing.T) {
	inputJSON := []byte(`{
		"record_sql":                    { "enabled": false, "required": false },
	        "attributes_include":            { "enabled": false, "required": false },
	        "allow_raw_exception_messages":  { "enabled": false, "required": false },
	        "custom_events":                 { "enabled": false, "required": false },
	        "custom_parameters":             { "enabled": false, "required": false },
	        "custom_instrumentation_editor": { "enabled": false, "required": false },
	        "message_parameters":            { "enabled": false, "required": false },
	        "job_arguments":                 { "enabled": false, "required": false },
		"unknown_policy":                { "enabled": false, "required": true  }
	}`)
	var policies SecurityPolicies
	err := json.Unmarshal(inputJSON, &policies)
	if nil == err {
		t.Fatal(err)
	}
	testBool(t, "PointerIfPopulated", false, nil != policies.PointerIfPopulated())
	testBool(t, "unknown required policy should be disconnect", true, IsDisconnectSecurityPolicyError(err))
}

func TestSecurityPolicyMissing(t *testing.T) {
	inputJSON := []byte(`{
		"record_sql":                    { "enabled": false, "required": false },
		"attributes_include":            { "enabled": false, "required": false },
		"allow_raw_exception_messages":  { "enabled": false, "required": false },
		"custom_events":                 { "enabled": false, "required": false },
		"request_parameters":            { "enabled": false, "required": false }
	}`)
	var policies SecurityPolicies
	err := json.Unmarshal(inputJSON, &policies)
	_, ok := err.(errUnsetPolicy)
	if !ok {
		t.Fatal(err)
	}
	testBool(t, "PointerIfPopulated", false, nil != policies.PointerIfPopulated())
	testBool(t, "missing policy should be disconnect", true, IsDisconnectSecurityPolicyError(err))
}

func TestMalformedPolicies(t *testing.T) {
	inputJSON := []byte(`{`)
	var policies SecurityPolicies
	err := json.Unmarshal(inputJSON, &policies)
	if nil == err {
		t.Fatal(err)
	}
	testBool(t, "malformed policies should not be disconnect", false, IsDisconnectSecurityPolicyError(err))
}
