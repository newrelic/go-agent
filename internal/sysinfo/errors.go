package sysinfo

import (
	"errors"
)

var (
	ErrFeatureUnsupported = errors.New("That feature is not supported on this platform")
)
