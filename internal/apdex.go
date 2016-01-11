package internal

import "time"

type ApdexZone int

const (
	ApdexNone ApdexZone = iota
	ApdexSatisfying
	ApdexTolerating
	ApdexFailing
)

// Note that this does not take into account whether or not the transaction
// had an error.  That is expected to be done by the caller.
func calculateApdexZone(threshold, duration time.Duration) ApdexZone {
	if duration <= threshold {
		return ApdexSatisfying
	}
	if duration <= (4 * threshold) {
		return ApdexTolerating
	}
	return ApdexFailing
}

func (zone ApdexZone) label() string {
	switch zone {
	case ApdexSatisfying:
		return "S"
	case ApdexTolerating:
		return "T"
	case ApdexFailing:
		return "F"
	default:
		return ""
	}
}
