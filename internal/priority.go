package internal

import (
	"encoding/hex"
	"errors"
	"hash/fnv"
)

// Priority determines transaction data sampling.  Its purpose is to ensure that communicating
// applications are sampling the same requests.
type Priority struct {
	// priority == encodingKeyHash XOR input
	encodingKeyHash uint32
	input           uint32
	priority        uint32

	// inputString is a hex string representation of input.  This field is generated lazily when
	// needed.
	inputString string
}

func hashEncodingKey(encodingKey string) uint32 {
	h := fnv.New32()
	h.Write([]byte(encodingKey))
	return h.Sum32()
}

// Value returns the priority number according to which things should be sampled.
func (p Priority) Value() uint32 { return p.priority }

// Input returns the priority input string used in the tracing payload.
func (p *Priority) Input() string {
	if "" == p.inputString {
		p.inputString = hex.EncodeToString([]byte{
			byte(p.input>>24) & 0xff,
			byte(p.input>>16) & 0xff,
			byte(p.input>>8) & 0xff,
			byte(p.input>>0) & 0xff,
		})
	}
	return p.inputString
}

// NewPriority creates a new random Priority.
func NewPriority(encodingKeyHash uint32) Priority {
	input := RandUint32()
	return Priority{
		encodingKeyHash: encodingKeyHash,
		input:           input,
		priority:        input ^ encodingKeyHash,
	}
}

// PriorityFromInput creates a priority from the inputString from the inbound payload.
func PriorityFromInput(encodingKeyHash uint32, inputString string) (Priority, error) {
	bits, err := hex.DecodeString(inputString)
	if nil != err {
		return Priority{}, err
	}
	if len(bits) < 4 {
		return Priority{}, errors.New("invalid priority input length")
	}
	input := uint32(bits[0])<<24 |
		uint32(bits[1])<<16 |
		uint32(bits[2])<<8 |
		uint32(bits[3])<<0
	return Priority{
		encodingKeyHash: encodingKeyHash,
		input:           input,
		priority:        input ^ encodingKeyHash,
		inputString:     inputString,
	}, nil
}
