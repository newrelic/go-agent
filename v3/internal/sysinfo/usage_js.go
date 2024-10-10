// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package sysinfo

import "errors"

// GetUsage gathers process times.
func GetUsage() (Usage, error) {
	return Usage{}, errors.New("not supported on GOOS=js")
}
