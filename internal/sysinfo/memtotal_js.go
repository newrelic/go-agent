// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build js && wasm

package sysinfo

import "errors"

// PhysicalMemoryBytes returns the total amount of host memory.
func PhysicalMemoryBytes() (uint64, error) {
	return 0, errors.New("unsupported js/wasm arch")
}
