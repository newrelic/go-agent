// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cat

import (
	"encoding/json"
	"errors"
	"fmt"
)

// SyntheticsHeader represents a decoded Synthetics header.
type SyntheticsHeader struct {
	Version    int
	AccountID  int
	ResourceID string
	JobID      string
	MonitorID  string
}

// SyntheticsInfo represents a decoded synthetics info payload.
type SyntheticsInfo struct {
	Version    int
	Type       string
	Initiator  string
	Attributes map[string]string
}

var (
	errInvalidSyntheticsJSON             = errors.New("invalid synthetics JSON")
	errInvalidSyntheticsInfoJSON         = errors.New("invalid synthetics info JSON")
	errInvalidSyntheticsVersion          = errors.New("version is not a float64")
	errInvalidSyntheticsAccountID        = errors.New("account ID is not a float64")
	errInvalidSyntheticsResourceID       = errors.New("synthetics resource ID is not a string")
	errInvalidSyntheticsJobID            = errors.New("synthetics job ID is not a string")
	errInvalidSyntheticsMonitorID        = errors.New("synthetics monitor ID is not a string")
	errInvalidSyntheticsInfoVersion      = errors.New("synthetics info version is not a float64")
	errMissingSyntheticsInfoVersion      = errors.New("synthetics info version is missing from JSON object")
	errInvalidSyntheticsInfoType         = errors.New("synthetics info type is not a string")
	errMissingSyntheticsInfoType         = errors.New("synthetics info type is missing from JSON object")
	errInvalidSyntheticsInfoInitiator    = errors.New("synthetics info initiator is not a string")
	errMissingSyntheticsInfoInitiator    = errors.New("synthetics info initiator is missing from JSON object")
	errInvalidSyntheticsInfoAttributes   = errors.New("synthetics info attributes is not a map")
	errInvalidSyntheticsInfoAttributeVal = errors.New("synthetics info keys and values must be strings")
)

type errUnexpectedSyntheticsVersion int

func (e errUnexpectedSyntheticsVersion) Error() string {
	return fmt.Sprintf("unexpected synthetics header version: %d", e)
}

// UnmarshalJSON unmarshalls a SyntheticsHeader from raw JSON.
func (s *SyntheticsHeader) UnmarshalJSON(data []byte) error {
	var ok bool
	var v interface{}

	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	arr, ok := v.([]interface{})
	if !ok {
		return errInvalidSyntheticsJSON
	}
	if len(arr) != 5 {
		return errUnexpectedArraySize{
			label:    "unexpected number of application data elements",
			expected: 5,
			actual:   len(arr),
		}
	}

	version, ok := arr[0].(float64)
	if !ok {
		return errInvalidSyntheticsVersion
	}
	s.Version = int(version)
	if s.Version != 1 {
		return errUnexpectedSyntheticsVersion(s.Version)
	}

	accountID, ok := arr[1].(float64)
	if !ok {
		return errInvalidSyntheticsAccountID
	}
	s.AccountID = int(accountID)

	if s.ResourceID, ok = arr[2].(string); !ok {
		return errInvalidSyntheticsResourceID
	}

	if s.JobID, ok = arr[3].(string); !ok {
		return errInvalidSyntheticsJobID
	}

	if s.MonitorID, ok = arr[4].(string); !ok {
		return errInvalidSyntheticsMonitorID
	}

	return nil
}

const (
	versionKey    = "version"
	typeKey       = "type"
	initiatorKey  = "initiator"
	attributesKey = "attributes"
)

// UnmarshalJSON unmarshalls a SyntheticsInfo from raw JSON.
func (s *SyntheticsInfo) UnmarshalJSON(data []byte) error {
	var v any

	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	m, ok := v.(map[string]any)
	if !ok {
		return errInvalidSyntheticsInfoJSON
	}

	version, ok := m[versionKey]
	if !ok {
		return errMissingSyntheticsInfoVersion
	}

	versionFloat, ok := version.(float64)
	if !ok {
		return errInvalidSyntheticsInfoVersion
	}

	s.Version = int(versionFloat)
	if s.Version != 1 {
		return errUnexpectedSyntheticsVersion(s.Version)
	}

	infoType, ok := m[typeKey]
	if !ok {
		return errMissingSyntheticsInfoType
	}

	s.Type, ok = infoType.(string)
	if !ok {
		return errInvalidSyntheticsInfoType
	}

	initiator, ok := m[initiatorKey]
	if !ok {
		return errMissingSyntheticsInfoInitiator
	}

	s.Initiator, ok = initiator.(string)
	if !ok {
		return errInvalidSyntheticsInfoInitiator
	}

	attrs, ok := m[attributesKey]
	if ok {
		attrMap, ok := attrs.(map[string]any)
		if !ok {
			return errInvalidSyntheticsInfoAttributes
		}
		for k, v := range attrMap {
			val, ok := v.(string)
			if !ok {
				return errInvalidSyntheticsInfoAttributeVal
			}
			if s.Attributes == nil {
				s.Attributes = map[string]string{k: val}
			} else {
				s.Attributes[k] = val
			}
		}
	}

	return nil
}
