package internal

import (
	"testing"

	"github.com/newrelic/go-agent/internal/crossagent"
)

func TestRandomPriority(t *testing.T) {
	encodingKey := "0123456789012345678901234567890123456789"
	encodingKeyHash := hashEncodingKey(encodingKey)
	before := NewPriority(encodingKeyHash)
	input1 := before.Input()
	if len(input1) != 8 {
		t.Fatal(input1)
	}
	input2 := before.Input()
	if len(input2) != 8 {
		t.Fatal(input2)
	}
	if input1 != input2 {
		t.Fatal(input1, input2)
	}
	after, err := PriorityFromInput(encodingKeyHash, input2)
	if nil != err {
		t.Fatal(err)
	}
	if before.Value() != after.Value() {
		t.Fatal(before.Value(), after.Value())
	}
	if before.Input() != after.Input() {
		t.Fatal(before.Input(), after.Input())
	}
}

func TestMalformedInputString(t *testing.T) {
	encodingKey := "0123456789012345678901234567890123456789"
	encodingKeyHash := hashEncodingKey(encodingKey)
	_, err := PriorityFromInput(encodingKeyHash, "012345678")
	if nil == err {
		t.Error("error expected for long input")
	}
	_, err = PriorityFromInput(encodingKeyHash, "012345")
	if nil == err {
		t.Error("error expected for excessively short input")
	}
	_, err = PriorityFromInput(encodingKeyHash, "!!!!!!!!")
	if nil == err {
		t.Error("error expected for non hex string")
	}
	_, err = PriorityFromInput(encodingKeyHash, "01234567")
	if nil != err {
		t.Error("success sanity check", err)
	}
}

func TestCrossAgentPriority(t *testing.T) {
	var testcases []struct {
		EncodingKey    string `json:"encoding_key"`
		PriorityString string `json:"priority_input"`
		PriorityNumber uint32 `json:"priority_number"`
	}
	if err := crossagent.ReadJSON("cat/priority.json", &testcases); nil != err {
		t.Fatal(err)
	}
	for _, tc := range testcases {
		encodingKeyHash := hashEncodingKey(tc.EncodingKey)
		p, err := PriorityFromInput(encodingKeyHash, tc.PriorityString)
		if nil != err {
			t.Error(err)
			continue
		}
		if p.Value() != tc.PriorityNumber {
			t.Error(tc.EncodingKey, tc.PriorityString, tc.PriorityNumber, p.Value())
		}
	}
}
