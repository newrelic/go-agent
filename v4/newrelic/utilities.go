// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"strings"
)

// minorVersion takes a given version string and returns only the major and
// minor portions of it. If the input is malformed, it returns the input
// untouched.
func minorVersion(v string) string {
	split := strings.SplitN(v, ".", 3)
	if len(split) < 2 {
		return v
	}
	return split[0] + "." + split[1]
}
