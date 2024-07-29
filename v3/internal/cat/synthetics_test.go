// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cat

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestSyntheticsUnmarshalInvalid(t *testing.T) {
	// Test error cases where we get a generic error from the JSON package.
	for _, input := range []string{
		// Basic malformed JSON test: beyond this, we're not going to unit test the
		// Go standard library's JSON package.
		``,
	} {
		synthetics := &SyntheticsHeader{}

		if err := json.Unmarshal([]byte(input), synthetics); err == nil {
			t.Errorf("given %s: error expected to be non-nil; got nil", input)
		}
	}

	// Test error cases where the incorrect number of elements was provided.
	for _, input := range []string{
		`[]`,
		`[1,2,3,4]`,
	} {
		synthetics := &SyntheticsHeader{}

		err := json.Unmarshal([]byte(input), synthetics)
		if _, ok := err.(errUnexpectedArraySize); !ok {
			t.Errorf("given %s: error expected to be errUnexpectedArraySize; got %v", input, err)
		}
	}

	// Test error cases with invalid version numbers.
	for _, input := range []string{
		`[0,1234,"resource","job","monitor"]`,
		`[2,1234,"resource","job","monitor"]`,
	} {
		synthetics := &SyntheticsHeader{}

		err := json.Unmarshal([]byte(input), synthetics)
		if _, ok := err.(errUnexpectedSyntheticsVersion); !ok {
			t.Errorf("given %s: error expected to be errUnexpectedSyntheticsVersion; got %v", input, err)
		}
	}

	// Test error cases where a specific variable is returned.
	for _, tc := range []struct {
		input string
		err   error
	}{
		// Unexpected JSON types.
		{`false`, errInvalidSyntheticsJSON},
		{`true`, errInvalidSyntheticsJSON},
		{`1234`, errInvalidSyntheticsJSON},
		{`{}`, errInvalidSyntheticsJSON},
		{`""`, errInvalidSyntheticsJSON},

		// Invalid data types for each field in turn.
		{`["version",1234,"resource","job","monitor"]`, errInvalidSyntheticsVersion},
		{`[1,"account","resource","job","monitor"]`, errInvalidSyntheticsAccountID},
		{`[1,1234,0,"job","monitor"]`, errInvalidSyntheticsResourceID},
		{`[1,1234,"resource",-1,"monitor"]`, errInvalidSyntheticsJobID},
		{`[1,1234,"resource","job",false]`, errInvalidSyntheticsMonitorID},
	} {
		synthetics := &SyntheticsHeader{}

		if err := json.Unmarshal([]byte(tc.input), synthetics); err != tc.err {
			t.Errorf("given %s: error expected to be %v; got %v", tc.input, tc.err, err)
		}
	}
}

func TestSyntheticsUnmarshalValid(t *testing.T) {
	for _, test := range []struct {
		json       string
		synthetics SyntheticsHeader
	}{
		{
			json: `[1,1234,"resource","job","monitor"]`,
			synthetics: SyntheticsHeader{
				Version:    1,
				AccountID:  1234,
				ResourceID: "resource",
				JobID:      "job",
				MonitorID:  "monitor",
			},
		},
	} {
		// Test unmarshalling.
		synthetics := &SyntheticsHeader{}
		if err := json.Unmarshal([]byte(test.json), synthetics); err != nil {
			t.Errorf("given %s: error expected to be nil; got %v", test.json, err)
		}

		if test.synthetics.Version != synthetics.Version {
			t.Errorf("given %s: Version expected to be %d; got %d", test.json, test.synthetics.Version, synthetics.Version)
		}

		if test.synthetics.AccountID != synthetics.AccountID {
			t.Errorf("given %s: AccountID expected to be %d; got %d", test.json, test.synthetics.AccountID, synthetics.AccountID)
		}

		if test.synthetics.ResourceID != synthetics.ResourceID {
			t.Errorf("given %s: ResourceID expected to be %s; got %s", test.json, test.synthetics.ResourceID, synthetics.ResourceID)
		}

		if test.synthetics.JobID != synthetics.JobID {
			t.Errorf("given %s: JobID expected to be %s; got %s", test.json, test.synthetics.JobID, synthetics.JobID)
		}

		if test.synthetics.MonitorID != synthetics.MonitorID {
			t.Errorf("given %s: MonitorID expected to be %s; got %s", test.json, test.synthetics.MonitorID, synthetics.MonitorID)
		}
	}
}

func TestSyntheticsInfoUnmarshal(t *testing.T) {
	type testCase struct {
		name           string
		json           string
		syntheticsInfo SyntheticsInfo
		expectedError  error
	}

	testCases := []testCase{
		{
			name:           "missing type field",
			json:           `{"version":1,"initiator":"cli"}`,
			syntheticsInfo: SyntheticsInfo{},
			expectedError:  errMissingSyntheticsInfoType,
		},
		{
			name:           "invalid type field",
			json:           `{"version":1,"initiator":"cli","type":1}`,
			syntheticsInfo: SyntheticsInfo{},
			expectedError:  errInvalidSyntheticsInfoType,
		},
		{
			name:           "missing initiator field",
			json:           `{"version":1,"type":"scheduled"}`,
			syntheticsInfo: SyntheticsInfo{},
			expectedError:  errMissingSyntheticsInfoInitiator,
		},
		{
			name:           "invalid initiator field",
			json:           `{"version":1,"initiator":1,"type":"scheduled"}`,
			syntheticsInfo: SyntheticsInfo{},
			expectedError:  errInvalidSyntheticsInfoInitiator,
		},
		{
			name:           "missing version field",
			json:           `{"type":"scheduled"}`,
			syntheticsInfo: SyntheticsInfo{},
			expectedError:  errMissingSyntheticsInfoVersion,
		},
		{
			name:           "invalid version field",
			json:           `{"version":"1","initiator":"cli","type":"scheduled"}`,
			syntheticsInfo: SyntheticsInfo{},
			expectedError:  errInvalidSyntheticsInfoVersion,
		},
		{
			name: "valid synthetics info",
			json: `{"version":1,"type":"scheduled","initiator":"cli"}`,
			syntheticsInfo: SyntheticsInfo{
				Version:   1,
				Type:      "scheduled",
				Initiator: "cli",
			},
			expectedError: nil,
		},
		{
			name: "valid synthetics info with attributes",
			json: `{"version":1,"type":"scheduled","initiator":"cli","attributes":{"hi":"hello"}}`,
			syntheticsInfo: SyntheticsInfo{
				Version:    1,
				Type:       "scheduled",
				Initiator:  "cli",
				Attributes: map[string]string{"hi": "hello"},
			},
			expectedError: nil,
		},
		{
			name: "valid synthetics info with invalid attributes",
			json: `{"version":1,"type":"scheduled","initiator":"cli","attributes":{"hi":1}}`,
			syntheticsInfo: SyntheticsInfo{
				Version:    1,
				Type:       "scheduled",
				Initiator:  "cli",
				Attributes: nil,
			},
			expectedError: errInvalidSyntheticsInfoAttributeVal,
		},
	}

	for _, testCase := range testCases {
		syntheticsInfo := SyntheticsInfo{}
		err := syntheticsInfo.UnmarshalJSON([]byte(testCase.json))
		if testCase.expectedError == nil {
			if err != nil {
				recordError(t, testCase.name, fmt.Sprintf("expected synthetics info to unmarshal without error, but got error: %v", err))
			}

			expect := testCase.syntheticsInfo
			if expect.Version != syntheticsInfo.Version {
				recordError(t, testCase.name, fmt.Sprintf(`expected version "%d", but got "%d"`, expect.Version, syntheticsInfo.Version))
			}

			if expect.Type != syntheticsInfo.Type {
				recordError(t, testCase.name, fmt.Sprintf(`expected version "%s", but got "%s"`, expect.Type, syntheticsInfo.Type))
			}

			if expect.Initiator != syntheticsInfo.Initiator {
				recordError(t, testCase.name, fmt.Sprintf(`expected version "%s", but got "%s"`, expect.Initiator, syntheticsInfo.Initiator))
			}

			if len(expect.Attributes) != 0 {
				if len(syntheticsInfo.Attributes) == 0 {
					recordError(t, testCase.name, fmt.Sprintf(`expected attribute array to have %d elements, but it only had %d`, len(expect.Attributes), len(syntheticsInfo.Attributes)))
				}
				for ek, ev := range expect.Attributes {
					v, ok := syntheticsInfo.Attributes[ek]
					if !ok {
						recordError(t, testCase.name, fmt.Sprintf(`expected attributes to contain key "%s", but it did not`, ek))
					}
					if ev != v {
						recordError(t, testCase.name, fmt.Sprintf(`expected attributes to contain "%s":"%s", but it contained "%s":"%s"`, ek, ev, ek, v))
					}
				}
			}
		} else {
			if err != testCase.expectedError {
				recordError(t, testCase.name, fmt.Sprintf(`expected synthetics info to unmarshal with error "%v", but got "%v"`, testCase.expectedError, err))
			}
		}
	}
}

func recordError(t *testing.T, test, err string) {
	t.Errorf("%s: %s", test, err)
}
