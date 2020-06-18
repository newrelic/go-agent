// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package internal

import "testing"

func TestTraceIDGenerator(t *testing.T) {
	tg := NewTraceIDGenerator(12345)
	id := tg.GenerateTraceID()
	if id != "d9466896a525ccbf" {
		t.Error(id)
	}
}
