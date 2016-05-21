package internal

import "time"

type apdexZone int

const (
	apdexNone apdexZone = iota
	apdexSatisfying
	apdexTolerating
	apdexFailing
)

// Note that this does not take into account whether or not the transaction
// had an error.  That is expected to be done by the caller.
func calculateApdexZone(threshold, duration time.Duration) apdexZone {
	if duration <= threshold {
		return apdexSatisfying
	}
	if duration <= (4 * threshold) {
		return apdexTolerating
	}
	return apdexFailing
}

func (zone apdexZone) label() string {
	switch zone {
	case apdexSatisfying:
		return "S"
	case apdexTolerating:
		return "T"
	case apdexFailing:
		return "F"
	default:
		return ""
	}
}
