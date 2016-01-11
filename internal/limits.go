package internal

import "time"

const (
	// app behavior
	ConnectBackoff            = 20 * time.Second
	HarvestPeriod             = 60 * time.Second
	CollectorTimeout          = 20 * time.Second
	AppDataChanSize           = 200
	failedMetricAttemptsLimit = 5
	failedEventsAttemptsLimit = 10

	// transaction behavior
	maxStackTraceFrames = 100
	MaxTxnErrors        = 5

	// harvest data
	maxMetrics       = 2 * 1000
	maxCustomEvents  = 10 * 1000
	maxTxnEvents     = 10 * 1000
	maxErrorEvents   = 100
	maxHarvestErrors = 20

	// attributes
	attributeKeyLengthLimit   = 255
	attributeValueLengthLimit = 255
	attributeUserLimit        = 64
	attributeAgentLimit       = 255 - attributeUserLimit // TODO(willhf): check this
	customEventAttributeLimit = 64

	// Limits affecting Config validation are found in the config package.
)
