// +build go1.9
// This build tag is necessary because GRPC/ProtoBuf libraries only support Go version 1.9 and up.

package newrelic

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/newrelic/go-agent/v3/internal"
	v1 "github.com/newrelic/go-agent/v3/internal/com_newrelic_trace_v1"
)

type gRPCtraceObserver struct {
	messages chan *spanEvent
	// messagesOnce protects messages from being closed multiple times.
	messagesOnce sync.Once

	initialConnSuccess chan struct{}
	// initConnOnce protects initialConnSuccess from being closed multiple times.
	initConnOnce sync.Once

	restartChan chan struct{}

	initiateShutdown chan struct{}
	// initShutdownOnce protects initiateShutdown from being closed multiple times.
	initShutdownOnce sync.Once

	shutdownComplete chan struct{}

	runID     internal.AgentRunID
	runIDLock sync.Mutex

	supportability *observerSupport

	observerConfig
}

type observerSupport struct {
	increment chan string
	dump      chan map[string]float64
}

const (
	// versionSupports8T records whether we are using a supported version of Go
	// for Infinite Tracing
	versionSupports8T = true
	// recordSpanBackoff is the time to wait after a failure on the RecordSpan
	// endpoint before retrying
	recordSpanBackoff = 15 * time.Second
	// numCodes is the total number of grpc.Codes
	numCodes = 17

	licenseMetadataKey = "license_key"
	runIDMetadataKey   = "agent_run_token"

	observerSeen        = "Supportability/InfiniteTracing/Span/Seen"
	observerSent        = "Supportability/InfiniteTracing/Span/Sent"
	observerCodeErr     = "Supportability/InfiniteTracing/Span/gRPC/"
	observerResponseErr = "Supportability/InfiniteTracing/Span/Response/Error"
)

var (
	codeStrings = func() map[codes.Code]string {
		codeStrings := make(map[codes.Code]string, numCodes)
		for i := 0; i < numCodes; i++ {
			code := codes.Code(i)
			codeStrings[code] = strings.ToUpper(code.String())
		}
		return codeStrings
	}()
)

type obsResult struct {
	// shutdown is if the trace observer should shutdown and stop sending
	// spans.
	shutdown bool
	// withoutBackoff is true if RecordSpan should be retried immediately and
	// without a backoff.
	withoutBackoff bool
}

func newTraceObserver(runID internal.AgentRunID, cfg observerConfig) (traceObserver, error) {
	to := &gRPCtraceObserver{
		messages:           make(chan *spanEvent, cfg.queueSize),
		initialConnSuccess: make(chan struct{}),
		restartChan:        make(chan struct{}, 1),
		initiateShutdown:   make(chan struct{}),
		shutdownComplete:   make(chan struct{}),
		runID:              runID,
		observerConfig:     cfg,
		supportability:     newObserverSupport(),
	}
	go to.handleSupportability()
	go func() {
		to.connectToTraceObserver()

		// Closing shutdownComplete must be done before closing messages.  This
		// prevents a panic from happening if consumeSpan is called between the
		// time when the messages and the shutdownComplete channels are closed.
		close(to.shutdownComplete)
		to.closeMessages()
		for range to.messages {
			// drain the channel
		}
	}()
	return to, nil
}

// closeMessages closes the gRPCtraceObserver messages channel and is safe to call
// multiple times.
func (to *gRPCtraceObserver) closeMessages() {
	to.messagesOnce.Do(func() {
		close(to.messages)
	})
}

// markInitialConnSuccessful closes the gRPCtraceObserver initialConnSuccess channel and
// is safe to call multiple times.
func (to *gRPCtraceObserver) markInitialConnSuccessful() {
	to.initConnOnce.Do(func() {
		close(to.initialConnSuccess)
	})
}

// startShutdown closes the gRPCtraceObserver initiateShutdown channel and
// is safe to call multiple times.
func (to *gRPCtraceObserver) startShutdown() {
	to.initShutdownOnce.Do(func() {
		close(to.initiateShutdown)
	})
}

func (to *gRPCtraceObserver) connectToTraceObserver() {
	var cred grpc.DialOption
	if nil == to.endpoint || !to.endpoint.secure {
		cred = grpc.WithInsecure()
	} else {
		cred = grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{}))
	}
	var conn *grpc.ClientConn
	var err error
	connectParams := grpc.ConnectParams{
		Backoff: backoff.Config{
			BaseDelay:  15 * time.Second,
			Multiplier: 2,
			MaxDelay:   300 * time.Second,
		},
	}
	if nil == to.dialer {
		conn, err = grpc.Dial(
			to.endpoint.host,
			cred,
			grpc.WithConnectParams(connectParams),
		)
	} else {
		conn, err = grpc.Dial("bufnet",
			grpc.WithDialer(to.dialer),
			grpc.WithInsecure(),
			grpc.WithConnectParams(connectParams),
		)
	}
	if nil != err {
		// this error is unrecoverable and will not be retried
		to.log.Error("trace observer unable to dial grpc endpoint", map[string]interface{}{
			"host": to.endpoint.host,
			"err":  err.Error(),
		})
		return
	}
	defer func() {
		// Related to https://github.com/grpc/grpc-go/issues/2159
		// If we call conn.Close() immediately, some messages may still be
		// buffered and will never be sent. Initial testing suggests this takes
		// around 150-200ms with a full channel.
		time.Sleep(500 * time.Millisecond)
		if err := conn.Close(); nil != err {
			to.log.Info("closing trace observer connection was not successful", map[string]interface{}{
				"err": err.Error(),
			})
		}
	}()

	serviceClient := v1.NewIngestServiceClient(conn)

	for {
		result := to.connectToStream(serviceClient)
		if result.shutdown {
			return
		}
		if !result.withoutBackoff && !to.removeBackoff {
			time.Sleep(recordSpanBackoff)
		}
	}
}

func (to *gRPCtraceObserver) connectToStream(serviceClient v1.IngestServiceClient) obsResult {
	to.runIDLock.Lock()
	runID := to.runID
	to.runIDLock.Unlock()
	ctx := metadata.AppendToOutgoingContext(context.Background(),
		licenseMetadataKey, to.license,
		runIDMetadataKey, string(runID),
	)
	spanClient, err := serviceClient.RecordSpan(ctx)
	if nil != err {
		to.log.Error("trace observer unable to create span client", map[string]interface{}{
			"err": err.Error(),
		})
		return obsResult{}
	}
	defer func() {
		to.log.Debug("closing trace observer sender", map[string]interface{}{})
		if err := spanClient.CloseSend(); err != nil {
			to.log.Debug("error closing trace observer sender", map[string]interface{}{
				"err": err.Error(),
			})
		}
	}()
	to.markInitialConnSuccessful()

	responseError := make(chan error, 1)

	go func() {
		for {
			s, err := spanClient.Recv()
			if nil != err {
				to.log.Error("trace observer response error", map[string]interface{}{
					"err": err.Error(),
				})
				// NOTE: even when the trace observer is shutting down
				// properly, an EOF error will be received here and a
				// supportability metric created.
				to.supportabilityError(err)
				responseError <- err
				return
			}
			to.log.Debug("trace observer response", map[string]interface{}{
				"messages_seen": s.GetMessagesSeen(),
			})
		}
	}()

	for {
		select {
		case msg := <-to.messages:
			if sendErr := to.sendSpan(spanClient, msg); sendErr != nil {
				// When send closes so does recv. Check the error on recv
				// because it could be a shutdown request when the error from
				// send was not.
				var respErr error
				ticker := time.NewTicker(10 * time.Millisecond)
				defer ticker.Stop()
				select {
				case respErr = <-responseError:
				case <-ticker.C:
					to.log.Debug("timeout waiting for response error from trace observer", nil)
				}
				return obsResult{
					shutdown: errShouldShutdown(sendErr) || errShouldShutdown(respErr),
				}
			}
		case <-to.restartChan:
			return obsResult{
				withoutBackoff: true,
			}
		case err := <-responseError:
			return obsResult{
				shutdown: errShouldShutdown(err),
			}
		case <-to.initiateShutdown:
			to.closeMessages()
			for msg := range to.messages {
				if err := to.sendSpan(spanClient, msg); err != nil {
					// if we fail to send a span, do not send the rest
					break
				}
			}
			return obsResult{
				shutdown: true,
			}
		}
	}
}

// restart reconnects to the remote trace observer with the given runID.
func (to *gRPCtraceObserver) restart(runID internal.AgentRunID) {
	if to.isShutdownComplete() {
		return
	}
	to.runIDLock.Lock()
	to.runID = runID
	to.runIDLock.Unlock()

	// If there is already a restart on the channel, we don't need to add another
	select {
	case to.restartChan <- struct{}{}:
	default:
	}
}

var errTimeout = errors.New("timeout exceeded while waiting for trace observer shutdown to complete")

// shutdown initiates a shutdown of the trace observer and blocks until either
// shutdown is complete or the given timeout is hit.
func (to *gRPCtraceObserver) shutdown(timeout time.Duration) error {
	to.startShutdown()
	ticker := time.NewTicker(timeout)
	defer ticker.Stop()
	// Block until the observer shutdown is complete or timeout hit
	select {
	case <-to.shutdownComplete:
		return nil
	case <-ticker.C:
		return errTimeout
	}
}

// initialConnCompleted returns true if the trace observer was ever able to
// connect successfully. It does not indicate the current connected state of
// the trace observer.
func (to *gRPCtraceObserver) initialConnCompleted() bool {
	select {
	case <-to.initialConnSuccess:
		return true
	default:
		return false
	}
}

// errShouldShutdown returns true if the given error is an Unimplemented error
// meaning the connection to the trace observer should be shutdown.
func errShouldShutdown(err error) bool {
	return status.Code(err) == codes.Unimplemented
}

func (to *gRPCtraceObserver) sendSpan(spanClient v1.IngestService_RecordSpanClient, msg *spanEvent) error {
	span := transformEvent(msg)
	to.log.Debug("sending span to trace observer", map[string]interface{}{
		"name": msg.Name,
	})
	if err := spanClient.Send(span); err != nil {
		to.log.Error("trace observer send error", map[string]interface{}{
			"err": err.Error(),
		})
		to.supportabilityError(err)
		return err
	}
	to.supportability.increment <- observerSent
	return nil
}

func (to *gRPCtraceObserver) handleSupportability() {
	metrics := newSupportMetrics()
	for {
		select {
		case <-to.appShutdown:
			// Only close this goroutine once the application _and_ the trace
			// observer have shutdown. This is because we will want to continue
			// to increment the Seen/Sent metrics when the application is
			// running but the trace observer is not.
			return
		case key := <-to.supportability.increment:
			metrics[key]++
		case to.supportability.dump <- metrics:
			// reset the metrics map
			metrics = newSupportMetrics()
		}
	}
}

func newSupportMetrics() map[string]float64 {
	// grpc codes, plus 2 for seen/sent, plus 1 for response errs
	metrics := make(map[string]float64, numCodes+3)
	// these two metrics must always be sent
	metrics[observerSeen] = 0
	metrics[observerSent] = 0
	return metrics
}

func newObserverSupport() *observerSupport {
	return &observerSupport{
		increment: make(chan string),
		dump:      make(chan map[string]float64),
	}
}

func (to *gRPCtraceObserver) dumpSupportabilityMetrics() map[string]float64 {
	if to.isAppShutdownComplete() {
		return nil
	}
	return <-to.supportability.dump
}

func errToCodeString(err error) string {
	code := status.Code(err)
	str, ok := codeStrings[code]
	if !ok {
		str = strings.ToUpper(code.String())
	}
	return str
}

func (to *gRPCtraceObserver) supportabilityError(err error) {
	to.supportability.increment <- observerCodeErr + errToCodeString(err)
	to.supportability.increment <- observerResponseErr
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
		switch v := val.(type) {
		case stringJSONWriter:
			span.AgentAttributes[key.String()] = obsvString(string(v))
		case intJSONWriter:
			span.AgentAttributes[key.String()] = obsvInt(int64(v))
		default:
			b := bytes.Buffer{}
			val.WriteJSON(&b)
			s := strings.Trim(b.String(), `"`)
			span.AgentAttributes[key.String()] = obsvString(s)
		}
	}

	return span
}

func (to *gRPCtraceObserver) consumeSpan(span *spanEvent) {
	if to.isAppShutdownComplete() {
		return
	}

	to.supportability.increment <- observerSeen

	if to.isShutdownComplete() {
		return
	}

	select {
	case to.messages <- span:
	default:
	}

	return
}

// isShutdownComplete returns a bool if the trace observer has been shutdown.
func (to *gRPCtraceObserver) isShutdownComplete() bool {
	select {
	case <-to.shutdownComplete:
		return true
	default:
	}
	return false
}

// isAppShutdownComplete returns a bool if the trace observer's application has
// been shutdown.
func (to *gRPCtraceObserver) isAppShutdownComplete() bool {
	select {
	case <-to.appShutdown:
		return true
	default:
	}
	return false
}

func expectObserverEvents(v internal.Validator, events *analyticsEvents, expect []internal.WantEvent, extraAttributes map[string]interface{}) {
	for i, e := range expect {
		if nil != e.Intrinsics {
			e.Intrinsics = mergeAttributes(extraAttributes, e.Intrinsics)
		}
		event := events.events[i].jsonWriter.(*spanEvent)
		expectObserverEvent(v, event, e)
	}
}

func expectObserverEvent(v internal.Validator, e *spanEvent, expect internal.WantEvent) {
	span := transformEvent(e)
	if nil != expect.Intrinsics {
		expectObserverAttributes(v, span.Intrinsics, expect.Intrinsics)
	}
	if nil != expect.UserAttributes {
		expectObserverAttributes(v, span.UserAttributes, expect.UserAttributes)
	}
	if nil != expect.AgentAttributes {
		expectObserverAttributes(v, span.AgentAttributes, expect.AgentAttributes)
	}
}

func expectObserverAttributes(v internal.Validator, actual map[string]*v1.AttributeValue, expect map[string]interface{}) {
	if len(actual) != len(expect) {
		v.Error("attributes length difference in trace observer. actual:", len(actual), "expect:", len(expect))
	}
	for key, val := range expect {
		found, ok := actual[key]
		if !ok {
			v.Error("expected attribute not found in trace observer: ", key)
			continue
		}
		if val == internal.MatchAnything {
			continue
		}
		switch exp := val.(type) {
		case bool:
			if f := found.GetBoolValue(); f != exp {
				v.Error("incorrect bool value for key", key, "in trace observer. actual:", f, "expect:", exp)
			}
		case string:
			if f := found.GetStringValue(); f != exp {
				v.Error("incorrect string value for key", key, "in trace observer. actual:", f, "expect:", exp)
			}
		case float64:
			plusOrMinus := 0.0000001 // with floating point math we can only get so close
			if f := found.GetDoubleValue(); f-exp > plusOrMinus || exp-f > plusOrMinus {
				v.Error("incorrect double value for key", key, "in trace observer. actual:", f, "expect:", exp)
			}
		case int:
			if f := found.GetIntValue(); f != int64(exp) {
				v.Error("incorrect int value for key", key, "in trace observer. actual:", f, "expect:", exp)
			}
		default:
			v.Error("unknown type for key", key, "in trace observer. expected:", exp)
		}
	}
	for key, val := range actual {
		_, ok := expect[key]
		if !ok {
			v.Error("unexpected attribute present in trace observer. key:", key, "value:", val)
			continue
		}
	}
}
