// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"testing"

	"github.com/newrelic/go-agent/v3/internal"
)

// Check Default Value
func TestCustomLimitsBasic(t *testing.T) {
	limit := internal.MaxCustomEvents
	limits := &internal.RequestEventLimits{
		CustomEvents: limit,
	}
	// This function will mock a connect reply from the server
	mockReplyFunction := func(reply *internal.ConnectReply) {
		reply.MockConnectReplyEventLimits(limits)
	}
	testApp := newTestApp(
		mockReplyFunction,
		ConfigCustomInsightsEventsMaxSamplesStored(limit),
	)

	customEventRate := limit / (60 / internal.CustomEventHarvestsPerMinute)

	// Check if custom event queue capacity == rate
	if customEventRate != testApp.app.testHarvest.CustomEvents.capacity() {
		t.Errorf("Custom Events Rate is not equal to harvest: expected %d, actual %d", customEventRate, testApp.app.testHarvest.CustomEvents.capacity())
	}
}
func TestCustomEventLimitUserSet(t *testing.T) {
	limit := 7000
	limits := &internal.RequestEventLimits{
		CustomEvents: limit,
	}
	mockReplyFunction := func(reply *internal.ConnectReply) {
		reply.MockConnectReplyEventLimits(limits)
	}
	testApp := newTestApp(
		mockReplyFunction,
		ConfigCustomInsightsEventsMaxSamplesStored(limit),
	)

	customEventRate := limit / (60 / internal.CustomEventHarvestsPerMinute)

	if customEventRate != testApp.app.testHarvest.CustomEvents.capacity() {
		t.Errorf("Custom Events Rate is not equal to harvest: expected %d, actual %d", customEventRate, testApp.app.testHarvest.CustomEvents.capacity())
	}
}

func TestCustomLimitEnthusiast(t *testing.T) {
	limit := 100000
	limits := &internal.RequestEventLimits{
		CustomEvents: limit,
	}
	// This function will mock a connect reply from the server
	mockReplyFunction := func(reply *internal.ConnectReply) {
		reply.MockConnectReplyEventLimits(limits)
	}
	testApp := newTestApp(
		mockReplyFunction,
		ConfigCustomInsightsEventsMaxSamplesStored(limit),
	)

	customEventRate := limit / (60 / internal.CustomEventHarvestsPerMinute)

	// Check if custom event queue capacity == rate
	if customEventRate != testApp.app.testHarvest.CustomEvents.capacity() {
		t.Errorf("Custom Events Rate is not equal to harvest: expected %d, actual %d", customEventRate, testApp.app.testHarvest.CustomEvents.capacity())
	}
}

func TestCustomLimitsTypo(t *testing.T) {
	limit := 1000000
	limits := &internal.RequestEventLimits{
		CustomEvents: limit,
	}
	// This function will mock a connect reply from the server
	mockReplyFunction := func(reply *internal.ConnectReply) {
		reply.MockConnectReplyEventLimits(limits)
	}
	testApp := newTestApp(
		mockReplyFunction,
		ConfigCustomInsightsEventsMaxSamplesStored(limit),
	)

	customEventRate := 100000 / (60 / internal.CustomEventHarvestsPerMinute)

	// Check if custom event queue capacity == rate
	if customEventRate != testApp.app.testHarvest.CustomEvents.capacity() {
		t.Errorf("Custom Events Rate is not equal to harvest: expected %d, actual %d", 8333, testApp.app.testHarvest.CustomEvents.capacity())
	}
}

func TestCustomLimitZero(t *testing.T) {
	limit := 0
	limits := &internal.RequestEventLimits{
		CustomEvents: limit,
	}
	// This function will mock a connect reply from the server
	mockReplyFunction := func(reply *internal.ConnectReply) {
		reply.MockConnectReplyEventLimits(limits)
	}
	testApp := newTestApp(
		mockReplyFunction,
		ConfigCustomInsightsEventsMaxSamplesStored(limit),
	)

	customEventRate := limit / (60 / internal.CustomEventHarvestsPerMinute)

	// Check if custom event queue capacity == rate
	if customEventRate != testApp.app.testHarvest.CustomEvents.capacity() {
		t.Errorf("Custom Events Rate is not equal to harvest: expected %d, actual %d", customEventRate, testApp.app.testHarvest.CustomEvents.capacity())
	}
}
