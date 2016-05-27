package internal

import "time"

type harvestable interface {
	mergeIntoHarvest(h *harvest)
}

type dataConsumer interface {
	consume(AgentRunID, harvestable)
}

type harvest struct {
	metrics      *metricTable
	customEvents *customEvents
	txnEvents    *txnEvents
	errorEvents  *errorEvents
	errorTraces  *harvestErrors
}

func (h *harvest) payloads() map[string]payloadCreator {
	return map[string]payloadCreator{
		cmdMetrics:      h.metrics,
		cmdCustomEvents: h.customEvents,
		cmdTxnEvents:    h.txnEvents,
		cmdErrorEvents:  h.errorEvents,
		cmdErrorData:    h.errorTraces,
	}
}

func newHarvest(now time.Time) *harvest {
	return &harvest{
		metrics:      newMetricTable(maxMetrics, now),
		customEvents: newCustomEvents(maxCustomEvents),
		txnEvents:    newTxnEvents(maxTxnEvents),
		errorEvents:  newErrorEvents(maxErrorEvents),
		errorTraces:  newHarvestErrors(maxHarvestErrors),
	}
}

func (h *harvest) createFinalMetrics() {
	h.metrics.addSingleCount(instanceReporting, forced)

	h.metrics.addCount(customEventsSeen, h.customEvents.numSeen(), forced)
	h.metrics.addCount(customEventsSent, h.customEvents.numSaved(), forced)

	h.metrics.addCount(txnEventsSeen, h.txnEvents.numSeen(), forced)
	h.metrics.addCount(txnEventsSent, h.txnEvents.numSaved(), forced)

	h.metrics.addCount(errorEventsSeen, h.errorEvents.numSeen(), forced)
	h.metrics.addCount(errorEventsSent, h.errorEvents.numSaved(), forced)

	if h.metrics.numDropped > 0 {
		h.metrics.addCount(supportabilityDropped, float64(h.metrics.numDropped), forced)
	}
}

func (h *harvest) applyMetricRules(rules metricRules) {
	h.metrics = h.metrics.applyRules(rules)
}

func (h *harvest) addTxnEvent(t *txnEvent) {
	h.txnEvents.AddTxnEvent(t)
}

type payloadCreator interface {
	// In the event of a rpm request failure (hopefully simply an
	// intermittent collector issue) the payload may be merged into the next
	// time period's harvest.
	harvestable
	// Data prepares JSON in the format expected by the collector endpoint.
	// This method should return (nil, nil) if the payload is empty and no
	// rpm request is necessary.
	Data(agentRunID string, harvestStart time.Time) ([]byte, error)
}

type createTxnMetricsArgs struct {
	IsWeb          bool
	Duration       time.Duration
	Name           string
	Zone           apdexZone
	ApdexThreshold time.Duration
	ErrorsSeen     uint64
}

func (h *harvest) createTxnMetrics(args createTxnMetricsArgs) {
	// Duration Metrics
	rollup := backgroundRollup
	if args.IsWeb {
		rollup = webRollup
		h.metrics.addDuration(dispatcherMetric, "", args.Duration, 0, forced)
	}
	exclusive := args.Duration
	h.metrics.addDuration(args.Name, "", args.Duration, exclusive, forced)
	h.metrics.addDuration(rollup, "", args.Duration, exclusive, forced)

	// Apdex Metrics
	if args.Zone != apdexNone {
		h.metrics.addApdex(apdexRollup, "", args.ApdexThreshold, args.Zone, forced)

		mname := apdexPrefix + removeFirstSegment(args.Name)
		h.metrics.addApdex(mname, "", args.ApdexThreshold, args.Zone, unforced)
	}

	// Error Metrics
	if args.ErrorsSeen > 0 {
		h.metrics.addSingleCount(errorsAll, forced)
		if args.IsWeb {
			h.metrics.addSingleCount(errorsWeb, forced)
		} else {
			h.metrics.addSingleCount(errorsBackground, forced)
		}
		h.metrics.addSingleCount(errorsPrefix+args.Name, forced)
	}
}
