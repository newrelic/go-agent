package internal

import (
	"bytes"
	"fmt"
	"strings"
)

// Priority allows for a priority sampling of events.  When an event
// is created it is given a Priority.  Whenever an event pool is
// full and events need to be dropped, the events with the lowest priority
// are dropped.
type Priority float32

// According to spec, Agents SHOULD truncate the value to at most 6
// digits past the decimal point.
const (
	priorityFormat = "%.6f"
)

func newPriorityFromRandom(rnd func() float32) Priority {
	for {
		if r := rnd(); 0.0 != r {
			return Priority(r)
		}
	}
}

// NewPriority returns a new priority.
func NewPriority() Priority {
	return newPriorityFromRandom(RandFloat32)
}

// Float32 returns the priority as a float32.
func (p Priority) Float32() float32 {
	return float32(p)
}

func (p Priority) isLowerPriority(y Priority) bool {
	return p < y
}

// MarshalJSON limits the number of decimals.
func (p Priority) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(priorityFormat, p)), nil
}

// WriteJSON limits the number of decimals.
func (p Priority) WriteJSON(buf *bytes.Buffer) {
	fmt.Fprintf(buf, priorityFormat, p)
}

func (p Priority) traceStateFormat() string {
	s := fmt.Sprintf(priorityFormat, p)
	s = strings.TrimRight(s, "0")
	return strings.TrimRight(s, ".")
}
