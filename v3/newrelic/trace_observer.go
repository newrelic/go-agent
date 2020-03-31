// +build go1.9
// This build tag is necessary because GRPC/ProtoBuf libraries only support Go version 1.9 and up.

package newrelic

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	v1 "github.com/newrelic/go-agent/v3/internal/com_newrelic_trace_v1"
)

func newTraceObserver(cfg observerConfig) (*traceObserver, error) {
	messages := make(chan *spanEvent, cfg.queueSize)
	to := &traceObserver{messages: messages}
	go func() {
		attempts := 0
		for {
			err := connectToTraceObserver(to, cfg)
			// If we returned nil, that means we're done.
			if nil == err {
				return
			}
			// TODO: Maybe decide if a reconnect should be
			// tried.
			fmt.Println(err)
			backoff := getConnectBackoffTime(attempts)
			time.Sleep(time.Duration(backoff) * time.Second)
			attempts++
		}
	}()
	return to, nil
}

func connectToTraceObserver(to *traceObserver, cfg observerConfig) error {
	responseError := make(chan error, 1)

	var cred grpc.DialOption
	if cfg.endpoint.secure {
		cred = grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{}))
	} else {
		cred = grpc.WithInsecure()
	}
	conn, err := grpc.Dial(
		cfg.endpoint.host,
		cred,
	)
	if nil != err {
		return fmt.Errorf("unable to dial grpc endpoint %s: %v", cfg.endpoint.host, err)
	}
	defer conn.Close()

	serviceClient := v1.NewIngestServiceClient(conn)

	ctx := metadata.AppendToOutgoingContext(context.Background(),
		licenseMetadataKey, cfg.license,
		runIDMetadataKey, string(cfg.runID),
	)
	spanClient, err := serviceClient.RecordSpan(ctx)
	if nil != err {
		return fmt.Errorf("unable to create span client: %v", err)
	}
	to.setConnectedState(true)

	go func() {
		for {
			status, err := spanClient.Recv()
			if nil != err {
				// If the error is an "already closed" error, break?
				if io.EOF != err {
					cfg.log.Error("trace observer response error", map[string]interface{}{
						"err": err.Error(),
					})
					responseError <- err
				}
				return
			}
			cfg.log.Debug("trace observer response", map[string]interface{}{
				"messages_seen": status.GetMessagesSeen(),
			})
		}
	}()

	// This will loop until the messages channel is closed and the messages have all been drained
	for msg := range to.messages {
		span := transformEvent(msg)
		cfg.log.Debug("sending span to trace observer", map[string]interface{}{
			"name": msg.Name,
		})
		err = spanClient.Send(span)
		if nil != err {
			cfg.log.Debug("trace observer sender send error", map[string]interface{}{
				"err": err.Error(),
			})
		}
	}

	cfg.log.Debug("closing trace observer sender", map[string]interface{}{})
	to.setConnectedState(false)
	err = spanClient.CloseSend()
	if nil != err {
		cfg.log.Debug("error closing trace observer sender", map[string]interface{}{
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

func (to *traceObserver) consumeSpan(span *spanEvent) bool {
	select {
	case to.messages <- span:
		return true
	default:
		return false
	}
}

// getConnectedState returns true if this traceObserver is currently connected
// to the remote traceObserver server
func (to *traceObserver) getConnectedState() bool {
	to.Lock()
	defer to.Unlock()
	return to.connected
}

// setConnectedState sets whether this traceObserver is currently connected
//  to the remote traceObserver server
func (to *traceObserver) setConnectedState(c bool) {
	to.Lock()
	defer to.Unlock()
	to.connected = c
}
