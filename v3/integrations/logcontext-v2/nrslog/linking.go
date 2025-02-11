package nrslog

import (
	"strings"

	"github.com/newrelic/go-agent/v3/newrelic"
)

type linkingCache struct {
	loaded     bool
	entityGUID string
	entityName string
	hostname   string
}

func newLinkingCache() *linkingCache {
	return &linkingCache{}
}

func (data *linkingCache) clone() *linkingCache {
	return &linkingCache{
		entityGUID: data.entityGUID,
		entityName: data.entityName,
		hostname:   data.hostname,
	}
}

// getAgentLinkingMetadata returns the linking metadata for the agent.
// we save a lot of time making calls to the go agent by caching the linking metadata
// which will never change during the lifetime of the agent.
//
// This returns a shallow copy of the cached metadata object
// 50% faster than calling GetLinkingMetadata() on every log message
//
// worst case: data race --> performance degrades to the cost of querying newrelic.Application.GetLinkingMetadata()
func (cache *linkingCache) getAgentLinkingMetadata(app *newrelic.Application) newrelic.LinkingMetadata {
	// entityGUID will be empty until the agent has connected
	if !cache.loaded {
		metadata := app.GetLinkingMetadata()
		cache.entityGUID = metadata.EntityGUID
		cache.entityName = metadata.EntityName
		cache.hostname = metadata.Hostname

		if cache.entityGUID != "" {
			cache.loaded = true
		}
		return metadata
	}

	return newrelic.LinkingMetadata{
		EntityGUID: cache.entityGUID,
		EntityName: cache.entityName,
		Hostname:   cache.hostname,
	}
}

// getTransactionLinkingMetadata returns the linking metadata for a transaction.
// we save a lot of time making calls to the go agent by caching the linking metadata
// which will never change during the lifetime of the transaction. This still needs to
// query for the trace and span IDs, but this is much cheaper than getting the linking metadata.
//
// This returns a shallow copy of the cached metadata object
func (cache *linkingCache) getTransactionLinkingMetadata(txn *newrelic.Transaction) newrelic.LinkingMetadata {
	if !cache.loaded {
		metadata := txn.GetLinkingMetadata() // marginally more expensive
		cache.entityGUID = metadata.EntityGUID
		cache.entityName = metadata.EntityName
		cache.hostname = metadata.Hostname

		if cache.entityGUID != "" {
			cache.loaded = true
		}
		return metadata
	}

	traceData := txn.GetTraceMetadata()
	return newrelic.LinkingMetadata{
		EntityGUID: cache.entityGUID,
		EntityName: cache.entityName,
		Hostname:   cache.hostname,
		TraceID:    traceData.TraceID,
		SpanID:     traceData.SpanID,
	}
}

const nrlinking = "NR-LINKING"

// nrLinkingString returns a string that represents the linking metadata
func nrLinkingString(data newrelic.LinkingMetadata) string {
	if data.EntityGUID == "" {
		return ""
	}

	len := 16 + len(data.EntityGUID) + len(data.Hostname) + len(data.TraceID) + len(data.SpanID) + len(data.EntityName)
	str := strings.Builder{}
	str.Grow(len) // only 1 alloc

	str.WriteString(nrlinking)
	str.WriteByte('|')
	str.WriteString(data.EntityGUID)
	str.WriteByte('|')
	str.WriteString(data.Hostname)
	str.WriteByte('|')
	str.WriteString(data.TraceID)
	str.WriteByte('|')
	str.WriteString(data.SpanID)
	str.WriteByte('|')
	str.WriteString(data.EntityName)
	str.WriteByte('|')

	return str.String()
}
