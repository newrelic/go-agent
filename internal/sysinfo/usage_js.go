// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build js && wasm

package sysinfo

import "errors"

// GetUsage gathers process times.
func GetUsage() (Usage, error) {
	return Usage{}, errors.New("unsupported js/wasm arch")
}
