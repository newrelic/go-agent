// +build go1.9
// This build tag is necessary because GRPC/ProtoBuf libraries only support Go version 1.9 and up.

package newrelic

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	"github.com/newrelic/go-agent/v3/internal"
	v1 "github.com/newrelic/go-agent/v3/internal/com_newrelic_trace_v1"
)

func getInfiniteTracingBackoff(attempt int) time.Duration {
	if attempt < len(infiniteTracingBackoffStrategy) {
		return infiniteTracingBackoffStrategy[attempt]
	}
	return infiniteTracingBackoffStrategy[len(infiniteTracingBackoffStrategy)-1]
}

func newTraceObserver(endpoint, apiKey string, runID internal.AgentRunID, lg Logger, connected chan<- bool) (*traceObserver, error) {
	messages := make(chan *spanEvent, traceObserverMessageQueueSize)

	go func() {
		attempts := 0
		for {
			err := spawnConnection(endpoint, apiKey, runID, lg, messages, connected)
			if nil != err {
				// TODO: Maybe decide if a reconnect should be
				// tried.
				fmt.Println(err)
			}
			time.Sleep(getInfiniteTracingBackoff(attempts))
			attempts++

		}
	}()

	return &traceObserver{messages: messages}, nil
}

func spawnConnection(endpoint, apiKey string, runID internal.AgentRunID, lg Logger, messages <-chan *spanEvent, connected chan<- bool) error {

	responseError := make(chan error, 1)

	conn, err := grpc.Dial(endpoint, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
	if nil != err {
		return fmt.Errorf("unable to dial grpc endpoint %s: %v", endpoint, err)
	}

	serviceClient := v1.NewIngestServiceClient(conn)

	ctx := metadata.AppendToOutgoingContext(context.Background(),
		apiKeyMetadataKey, apiKey,
		runIDMetadataKey, string(runID),
	)
	spanClient, err := serviceClient.RecordSpan(ctx)
	if nil != err {
		return fmt.Errorf("unable to create span client: %v", err)
	}
	connected <- true

	go func() {
		for {
			status, err := spanClient.Recv()
			if nil != err {
				lg.Error("trace observer response error", map[string]interface{}{
					"err": err.Error(),
				})
				responseError <- err
				return
			}
			lg.Debug("trace observer response", map[string]interface{}{
				"messages_seen": status.GetMessagesSeen(),
			})
		}
	}()

	for {
		var err error
		var event *spanEvent
		select {
		case err = <-responseError:
		case event = <-messages:
		}
		if nil != err {
			lg.Debug("trace observer sender received response error", map[string]interface{}{
				"err": err.Error(),
			})
			break
		}
		span := transformEvent(event)
		lg.Debug("sending span to trace observer", map[string]interface{}{
			"name": event.Name,
		})
		err = spanClient.Send(span)
		if nil != err {
			lg.Debug("trace observer sender send error", map[string]interface{}{
				"err": err.Error(),
			})
			break
		}
	}

	lg.Debug("closing trace observer sender", map[string]interface{}{})
	connected <- false
	err = spanClient.CloseSend()
	if nil != err {
		lg.Debug("error closing trace observer sender", map[string]interface{}{
			"err": err.Error(),
		})
	}

	return nil
}

func obsvString(s string) *v1.AttributeValue {
	return &v1.AttributeValue{Value: &v1.AttributeValue_StringValue{StringValue: s}}
}

func obsvBool(b bool) *v1.AttributeValue {
	return &v1.AttributeValue{Value: &v1.AttributeValue_BoolValue{BoolValue: b}}
}

func obsvInt(x int64) *v1.AttributeValue {
	return &v1.AttributeValue{Value: &v1.AttributeValue_IntValue{IntValue: x}}
}

func obsvDouble(x float64) *v1.AttributeValue {
	return &v1.AttributeValue{Value: &v1.AttributeValue_DoubleValue{DoubleValue: x}}
}

func transformEvent(e *spanEvent) *v1.Span {
	span := &v1.Span{
		TraceId:         e.TraceID,
		Intrinsics:      make(map[string]*v1.AttributeValue),
		UserAttributes:  make(map[string]*v1.AttributeValue),
		AgentAttributes: make(map[string]*v1.AttributeValue),
	}

	span.Intrinsics["type"] = obsvString("Span")
	span.Intrinsics["traceId"] = obsvString(e.TraceID)
	span.Intrinsics["guid"] = obsvString(e.GUID)
	if "" != e.ParentID {
		span.Intrinsics["parentId"] = obsvString(e.ParentID)
	}
	span.Intrinsics["transactionId"] = obsvString(e.TransactionID)
	span.Intrinsics["sampled"] = obsvBool(e.Sampled)
	span.Intrinsics["priority"] = obsvDouble(float64(e.Priority.Float32()))
	span.Intrinsics["timestamp"] = obsvInt(e.Timestamp.UnixNano() / (1000 * 1000)) // in milliseconds
	span.Intrinsics["duration"] = obsvDouble(e.Duration.Seconds())
	span.Intrinsics["name"] = obsvString(e.Name)
	span.Intrinsics["category"] = obsvString(string(e.Category))
	if e.IsEntrypoint {
		span.Intrinsics["nr.entryPoint"] = obsvBool(true)
	}
	if e.Component != "" {
		span.Intrinsics["component"] = obsvString(e.Component)
	}
	if e.Kind != "" {
		span.Intrinsics["span.kind"] = obsvString(e.Kind)
	}
	if "" != e.TrustedParentID {
		span.Intrinsics["trustedParentId"] = obsvString(e.TrustedParentID)
	}
	if "" != e.TracingVendors {
		span.Intrinsics["tracingVendors"] = obsvString(e.TracingVendors)
	}

	for key, val := range e.Attributes {
		// This assumes all values are string types.
		// TODO: Future-proof this!
		b := bytes.Buffer{}
		val.WriteJSON(&b)
		s := strings.Trim(b.String(), `"`)
		span.AgentAttributes[key.String()] = obsvString(s)
	}

	return span
}

// func (tb *traceObserver) sendSpans(events []*spanEvent) {
// 	for _, e := range events {
// 		span := transformEvent(e)
// 		fmt.Println("sending span", e.Name)
// 		err := tb.spanClient.Send(span)
// 		if nil != err {
// 			// TODO: Deal with this.
// 			fmt.Println("spanClient.Send error", err.Error())
// 		}
// 	}
// }

func (to *traceObserver) consumeSpan(span *spanEvent) bool {
	select {
	case to.messages <- span:
		return true
	default:
		return false
	}
}
