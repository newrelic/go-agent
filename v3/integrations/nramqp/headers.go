package nramqp

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/newrelic/go-agent/v3/newrelic"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	MaxHeaderLen = 4096
)

// Adds Distributed Tracing headers to the amqp table object
func injectDtHeaders(txn *newrelic.Transaction, headers amqp.Table) amqp.Table {
	dummyHeaders := http.Header{}

	txn.InsertDistributedTraceHeaders(dummyHeaders)
	if headers == nil {
		headers = amqp.Table{}
	}

	dtHeaders := dummyHeaders.Get(newrelic.DistributedTraceNewRelicHeader)
	if dtHeaders != "" {
		headers[newrelic.DistributedTraceNewRelicHeader] = dtHeaders
	}
	traceParent := dummyHeaders.Get(newrelic.DistributedTraceW3CTraceParentHeader)
	if traceParent != "" {
		headers[newrelic.DistributedTraceW3CTraceParentHeader] = traceParent
	}
	traceState := dummyHeaders.Get(newrelic.DistributedTraceW3CTraceStateHeader)
	if traceState != "" {
		headers[newrelic.DistributedTraceW3CTraceStateHeader] = traceState
	}

	return headers
}

func toHeader(headers amqp.Table) http.Header {
	headersHTTP := http.Header{}
	if headers == nil {
		return headersHTTP
	}

	for k, v := range headers {
		headersHTTP.Set(k, fmt.Sprintf("%v", v))
	}

	return headersHTTP
}

func getHeadersAttributeString(hdrs amqp.Table) (string, error) {
	if len(hdrs) == 0 {
		return "", nil
	}

	delete(hdrs, newrelic.DistributedTraceNewRelicHeader)
	delete(hdrs, newrelic.DistributedTraceW3CTraceParentHeader)
	delete(hdrs, newrelic.DistributedTraceW3CTraceStateHeader)

	if len(hdrs) == 0 {
		return "", nil
	}

	bytes, err := json.Marshal(hdrs)
	return string(bytes), err
}
