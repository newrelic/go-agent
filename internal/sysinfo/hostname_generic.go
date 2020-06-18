// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// +build !linux

package sysinfo

import "os"

// Hostname returns the host name.
func Hostname() (string, error) {
	return os.Hostname()
}
