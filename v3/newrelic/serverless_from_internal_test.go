// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func createCompressedData(data map[string]interface{}) string {
	jsonData, _ := json.Marshal(data)

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err := gz.Write(jsonData)
	if err != nil {
		return ""
	}
	err = gz.Close()
	if err != nil {
		return ""
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func createServerlessPayload(metadata map[string]interface{}, compressedData string) []byte {
	metadataJSON, _ := json.Marshal(metadata)
	payload := []interface{}{
		nil,
		nil,
		json.RawMessage(metadataJSON),
		json.RawMessage(`"` + compressedData + `"`),
	}
	payloadJSON, _ := json.Marshal(payload)
	return payloadJSON
}

func TestParseServerlessPayload(t *testing.T) {
	testData := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	}
	compressedData := createCompressedData(testData)

	metadata := map[string]interface{}{
		"version": "1.0",
		"type":    "serverless",
	}
	payloadJSON := createServerlessPayload(metadata, compressedData)

	resultMetadata, resultData, err := parseServerlessPayload(payloadJSON)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	if len(resultMetadata) != 2 {
		t.Errorf("expected 2 metadata items, got %d", len(resultMetadata))
	}

	if len(resultData) != 2 {
		t.Errorf("expected 2 data items, got %d", len(resultData))
	}

	// Verify metadata values
	var version string
	err = json.Unmarshal(resultMetadata["version"], &version)
	if err != nil {
		t.Fail()
	}

	if version != "1.0" {
		t.Errorf("expected version '1.0', got '%s'", version)
	}

	var dataType string
	err = json.Unmarshal(resultMetadata["type"], &dataType)
	if err != nil {
		t.Fail()
	}

	if dataType != "serverless" {
		t.Errorf("expected type 'serverless', got '%s'", dataType)
	}

	// Verify data values
	var key1 string
	err = json.Unmarshal(resultData["key1"], &key1)
	if err != nil {
		t.Fail()
	}
	if key1 != "value1" {
		t.Errorf("expected key1 'value1', got '%s'", key1)
	}

	var key2 int
	err = json.Unmarshal(resultData["key2"], &key2)
	if err != nil {
		t.Fail()
	}
	if key2 != 123 {
		t.Errorf("expected key2 123, got %d", key2)
	}
}

func TestParseServerlessPayloadErrors(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		wantErr  bool
		errCheck func(error) bool
	}{
		{
			name:    "invalid JSON",
			input:   []byte(`{"invalid": json}`),
			wantErr: true,
		},
		{
			name: "invalid array length",
			input: func() []byte {
				invalidArray := []interface{}{nil, nil}
				data, _ := json.Marshal(invalidArray)
				return data
			}(),
			wantErr: true,
		},
		{
			name: "invalid base64 data",
			input: func() []byte {
				metadata := map[string]interface{}{"version": "1.0"}
				return createServerlessPayload(metadata, "invalid-base64-!@#")
			}(),
			wantErr: true,
		},
		{
			name: "invalid metadata JSON",
			input: func() []byte {
				testData := map[string]interface{}{"key": "value"}
				compressedData := createCompressedData(testData)
				payload := []interface{}{
					nil,
					nil,
					json.RawMessage(`{invalid json}`),
					json.RawMessage(`"` + compressedData + `"`),
				}
				data, _ := json.Marshal(payload)
				return data
			}(),
			wantErr: true,
		},
		{
			name: "invalid metadata type",
			input: func() []byte {
				testData := map[string]interface{}{"key": "value"}
				compressedData := createCompressedData(testData)
				payload := []interface{}{
					nil,
					nil,
					json.RawMessage(`"not a json object"`),
					json.RawMessage(`"` + compressedData + `"`),
				}
				data, _ := json.Marshal(payload)
				return data
			}(),
			wantErr: true,
			errCheck: func(err error) bool {
				return strings.Contains(err.Error(), "unable to unmarshal serverless metadata")
			},
		},
		{
			name: "invalid uncompressed data",
			input: func() []byte {
				invalidJSON := []byte(`{"invalid": json syntax error}`)
				var buf bytes.Buffer
				gz := gzip.NewWriter(&buf)
				_, err := gz.Write(invalidJSON)
				if err != nil {
					return nil
				}
				err = gz.Close()
				if err != nil {
					return nil
				}
				compressedInvalidData := base64.StdEncoding.EncodeToString(buf.Bytes())

				metadata := map[string]interface{}{"version": "1.0"}
				return createServerlessPayload(metadata, compressedInvalidData)
			}(),
			wantErr: true,
			errCheck: func(err error) bool {
				return strings.Contains(err.Error(), "unable to unmarshal uncompressed serverless data")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := parseServerlessPayload(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("unexpected error state: got error = %v, wantErr = %v", err, tt.wantErr)
			}
			if tt.errCheck != nil && err != nil {
				if !tt.errCheck(err) {
					t.Errorf("error check failed: %v", err)
				}
			}
		})
	}
}

func TestDecodeUncompress(t *testing.T) {
	t.Run("successful decode and uncompress", func(t *testing.T) {
		originalData := []byte(`{"test": "data", "number": 42}`)

		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		_, err := gz.Write(originalData)
		if err != nil {
			t.Fail()
		}
		err = gz.Close()
		if err != nil {
			t.Fail()
		}

		encoded := base64.StdEncoding.EncodeToString(buf.Bytes())

		result, err := decodeUncompress(encoded)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if string(result) != string(originalData) {
			t.Errorf("expected %s, got %s", string(originalData), string(result))
		}
	})

	errorTests := []struct {
		name  string
		input string
	}{
		{
			name:  "invalid base64",
			input: "invalid-base64-!@#$%",
		},
		{
			name:  "valid base64 but not gzip",
			input: base64.StdEncoding.EncodeToString([]byte("not gzip data")),
		},
		{
			name:  "empty input",
			input: "",
		},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decodeUncompress(tt.input)
			if err == nil {
				t.Errorf("expected error for input '%s', got nil", tt.input)
			}
		})
	}
}
