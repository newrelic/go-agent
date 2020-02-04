package newrelic

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/newrelic/go-agent/v3/internal"
	v1 "github.com/newrelic/go-agent/v3/internal/com_newrelic_trace_v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

type traceBox struct {
	apiKey        string
	conn          *grpc.ClientConn
	serviceClient v1.IngestServiceClient
	spanClient    v1.IngestService_RecordSpanClient
}

const (
	apiKeyMetadataKey = "api_key"
)

func newTraceBox(endpoint, apiKey string) (*traceBox, error) {
	conn, err := grpc.Dial(endpoint, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
	if nil != err {
		return nil, err
	}

	serviceClient := v1.NewIngestServiceClient(conn)

	ctx := context.Background()

	// TODO: This commented code does not work, dunno why.  Only the
	// metadata.AppendToOutgoingContext pattern works.
	//
	// md := metadata.New(map[string]string{
	// 	apiKeyMetadataKey: apiKey,
	// })
	// opt := grpc.Header(&md)
	// spanClient, err := serviceClient.RecordSpan(ctx, opt)

	spanClient, err := serviceClient.RecordSpan(metadata.AppendToOutgoingContext(ctx, "api_key", apiKey))
	if nil != err {
		fmt.Println("unable to spanClient", err.Error())
		return nil, err
	}

	go func() {
		for {
			status, err := spanClient.Recv()
			fmt.Println("spanClient.Recv", "status:", status, "err:", err)
		}
	}()

	return &traceBox{
		conn:          conn,
		apiKey:        apiKey,
		serviceClient: serviceClient,
		spanClient:    spanClient,
	}, nil
}

func mtbString(s string) *v1.AttributeValue {
	return &v1.AttributeValue{Value: &v1.AttributeValue_StringValue{StringValue: s}}
}

func mtbBool(b bool) *v1.AttributeValue {
	return &v1.AttributeValue{Value: &v1.AttributeValue_BoolValue{BoolValue: b}}
}

func mtbInt(x int64) *v1.AttributeValue {
	return &v1.AttributeValue{Value: &v1.AttributeValue_IntValue{IntValue: x}}
}

func mtbDouble(x float64) *v1.AttributeValue {
	return &v1.AttributeValue{Value: &v1.AttributeValue_DoubleValue{DoubleValue: x}}
}

func transformEvent(e *internal.SpanEvent) *v1.Span {
	span := &v1.Span{
		TraceId:         e.TraceID,
		Intrinsics:      make(map[string]*v1.AttributeValue),
		UserAttributes:  make(map[string]*v1.AttributeValue),
		AgentAttributes: make(map[string]*v1.AttributeValue),
	}

	span.Intrinsics["type"] = mtbString("Span")
	span.Intrinsics["traceId"] = mtbString(e.TraceID)
	span.Intrinsics["guid"] = mtbString(e.GUID)
	if "" != e.ParentID {
		span.Intrinsics["parentId"] = mtbString(e.ParentID)
	}
	span.Intrinsics["transactionId"] = mtbString(e.TransactionID)
	span.Intrinsics["sampled"] = mtbBool(e.Sampled)
	span.Intrinsics["priority"] = mtbDouble(float64(e.Priority.Float32()))
	span.Intrinsics["timestamp"] = mtbInt(e.Timestamp.UnixNano() / (1000 * 1000)) // in milliseconds
	span.Intrinsics["duration"] = mtbDouble(e.Duration.Seconds())
	span.Intrinsics["name"] = mtbString(e.Name)
	span.Intrinsics["category"] = mtbString(string(e.Category))
	if e.IsEntrypoint {
		span.Intrinsics["nr.entryPoint"] = mtbBool(true)
	}
	if e.Component != "" {
		span.Intrinsics["component"] = mtbString(e.Component)
	}
	if e.Kind != "" {
		span.Intrinsics["span.kind"] = mtbString(e.Kind)
	}
	if "" != e.TrustedParentID {
		span.Intrinsics["trustedParentId"] = mtbString(e.TrustedParentID)
	}
	if "" != e.TracingVendors {
		span.Intrinsics["tracingVendors"] = mtbString(e.TracingVendors)
	}

	for key, val := range e.Attributes {
		// This assumes all values are string types.
		// TODO: Future-proof this!
		b := bytes.Buffer{}
		val.WriteJSON(&b)
		s := strings.Trim(b.String(), `"`)
		span.AgentAttributes[key.String()] = mtbString(s)
	}

	return span
}

func (tb *traceBox) sendSpans(events []*internal.SpanEvent) {
	for _, e := range events {
		span := transformEvent(e)
		fmt.Println("sending span", e.Name)
		err := tb.spanClient.Send(span)
		if nil != err {
			// TODO: Deal with this.
			fmt.Println("spanClient.Send error", err.Error())
		}
	}
}
