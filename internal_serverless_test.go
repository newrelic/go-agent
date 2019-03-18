package newrelic

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/newrelic/go-agent/internal"
)

func TestServerlessDistributedTracingConfigPresent(t *testing.T) {
	cfgFn := func(cfg *Config) {
		cfg.ServerlessMode.Enabled = true
		cfg.DistributedTracer.Enabled = true
		cfg.CrossApplicationTracer.Enabled = false
		cfg.ServerlessMode.AccountID = "123"
		cfg.ServerlessMode.TrustedAccountKey = "trustkey"
		cfg.ServerlessMode.PrimaryAppID = "456"
	}
	app := testApp(nil, cfgFn, t)
	payload := app.StartTransaction("hello", nil, nil).CreateDistributedTracePayload()
	txn := app.StartTransaction("hello", nil, nil)
	txn.AcceptDistributedTracePayload(TransportHTTP, payload)
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/456/HTTP/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/456/HTTP/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "Supportability/DistributedTrace/AcceptPayload/Success", Scope: "", Forced: true, Data: singleCount},
	})
}

func TestServerlessDistributedTracingConfigPartiallyPresent(t *testing.T) {
	// This tests that if ServerlessMode.PrimaryAppID is unset it should
	// default to "Unknown".
	cfgFn := func(cfg *Config) {
		cfg.ServerlessMode.Enabled = true
		cfg.DistributedTracer.Enabled = true
		cfg.CrossApplicationTracer.Enabled = false
		cfg.ServerlessMode.AccountID = "123"
		cfg.ServerlessMode.TrustedAccountKey = "trustkey"
	}
	app := testApp(nil, cfgFn, t)
	payload := app.StartTransaction("hello", nil, nil).CreateDistributedTracePayload()
	txn := app.StartTransaction("hello", nil, nil)
	txn.AcceptDistributedTracePayload(TransportHTTP, payload)
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/App/123/Unknown/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/Unknown/HTTP/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/Unknown/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/Unknown/HTTP/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "Supportability/DistributedTrace/AcceptPayload/Success", Scope: "", Forced: true, Data: singleCount},
	})
}

func TestServerlessDistributedTracingConfigTrustKeyAbsent(t *testing.T) {
	// Test that distributed tracing works if only AccountID has been set.
	cfgFn := func(cfg *Config) {
		cfg.ServerlessMode.Enabled = true
		cfg.DistributedTracer.Enabled = true
		cfg.CrossApplicationTracer.Enabled = false
		cfg.ServerlessMode.AccountID = "123"
	}
	app := testApp(nil, cfgFn, t)
	payload := app.StartTransaction("hello", nil, nil).CreateDistributedTracePayload()
	txn := app.StartTransaction("hello", nil, nil)
	txn.AcceptDistributedTracePayload(TransportHTTP, payload)
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/App/123/Unknown/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/App/123/Unknown/HTTP/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/Unknown/HTTP/all", Scope: "", Forced: false, Data: nil},
		{Name: "TransportDuration/App/123/Unknown/HTTP/allOther", Scope: "", Forced: false, Data: nil},
		{Name: "Supportability/DistributedTrace/AcceptPayload/Success", Scope: "", Forced: true, Data: singleCount},
	})
}

func TestServerlessDistributedTracingConfigAbsent(t *testing.T) {
	// Test that payloads do not get created or accepted when distributed
	// tracing configuration is not present.
	cfgFn := func(cfg *Config) {
		cfg.ServerlessMode.Enabled = true
		cfg.DistributedTracer.Enabled = true
		cfg.CrossApplicationTracer.Enabled = false
	}
	app := testApp(nil, cfgFn, t)
	txn := app.StartTransaction("hello", nil, nil)
	payload := txn.CreateDistributedTracePayload()
	if "" != payload.Text() {
		t.Error(payload.Text())
	}
	nonemptyPayload := func() DistributedTracePayload {
		app := testApp(nil, func(cfg *Config) {
			cfgFn(cfg)
			cfg.ServerlessMode.AccountID = "123"
			cfg.ServerlessMode.TrustedAccountKey = "trustkey"
			cfg.ServerlessMode.PrimaryAppID = "456"
		}, t)
		return app.StartTransaction("hello", nil, nil).CreateDistributedTracePayload()
	}()
	if "" == nonemptyPayload.Text() {
		t.Error(nonemptyPayload.Text())
	}
	err := txn.AcceptDistributedTracePayload(TransportHTTP, nonemptyPayload)
	if err != nil {
		t.Error(err)
	}
	txn.End()
	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "OtherTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransaction/all", Scope: "", Forced: true, Data: nil},
		{Name: "OtherTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "OtherTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/all", Scope: "", Forced: false, Data: nil},
		{Name: "DurationByCaller/Unknown/Unknown/Unknown/Unknown/allOther", Scope: "", Forced: false, Data: nil},
	})
}

func TestServerlessLowApdex(t *testing.T) {
	apdex := -1 * time.Second
	cfgFn := func(cfg *Config) {
		cfg.ServerlessMode.Enabled = true
		cfg.ServerlessMode.ApdexThreshold = apdex
	}
	app := testApp(nil, cfgFn, t)
	txn := app.StartTransaction("hello", nil, nil)
	txn.SetWebRequest(nil) // only web gets apdex
	txn.End()

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		// third apdex field is failed count
		{Name: "Apdex", Scope: "", Forced: true, Data: []float64{0, 0, 1, apdex.Seconds(), apdex.Seconds(), 0}},
		{Name: "Apdex/Go/hello", Scope: "", Forced: false, Data: []float64{0, 0, 1, apdex.Seconds(), apdex.Seconds(), 0}},
	})
}

func TestServerlessHighApdex(t *testing.T) {
	apdex := 1 * time.Hour
	cfgFn := func(cfg *Config) {
		cfg.ServerlessMode.Enabled = true
		cfg.ServerlessMode.ApdexThreshold = apdex
	}
	app := testApp(nil, cfgFn, t)
	txn := app.StartTransaction("hello", nil, nil)
	txn.SetWebRequest(nil) // only web gets apdex
	txn.End()

	app.ExpectMetrics(t, []internal.WantMetric{
		{Name: "WebTransaction/Go/hello", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransaction", Scope: "", Forced: true, Data: nil},
		{Name: "WebTransactionTotalTime/Go/hello", Scope: "", Forced: false, Data: nil},
		{Name: "WebTransactionTotalTime", Scope: "", Forced: true, Data: nil},
		{Name: "HttpDispatcher", Scope: "", Forced: true, Data: nil},
		// first apdex field is satisfied count
		{Name: "Apdex", Scope: "", Forced: true, Data: []float64{1, 0, 0, apdex.Seconds(), apdex.Seconds(), 0}},
		{Name: "Apdex/Go/hello", Scope: "", Forced: false, Data: []float64{1, 0, 0, apdex.Seconds(), apdex.Seconds(), 0}},
	})
}

func TestServerlessRecordCustomMetric(t *testing.T) {
	cfgFn := func(cfg *Config) { cfg.ServerlessMode.Enabled = true }
	app := testApp(nil, cfgFn, t)
	err := app.RecordCustomMetric("myMetric", 123.0)
	if err != errMetricServerless {
		t.Error(err)
	}
}

func TestServerlessRecordCustomEvent(t *testing.T) {
	cfgFn := func(cfg *Config) { cfg.ServerlessMode.Enabled = true }
	app := testApp(nil, cfgFn, t)
	err := app.RecordCustomEvent("myType", validParams)
	if err != errCustomEventsServerless {
		t.Error(err)
	}
}

func decodeUncompress(input string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(input)
	if nil != err {
		return nil, err
	}

	buf := bytes.NewBuffer(decoded)
	gz, err := gzip.NewReader(buf)
	if nil != err {
		return nil, err
	}
	var out bytes.Buffer
	io.Copy(&out, gz)
	gz.Close()

	return out.Bytes(), nil
}

func TestServerlessJSON(t *testing.T) {
	cfgFn := func(cfg *Config) {
		cfg.ServerlessMode.Enabled = true
	}
	app := testApp(nil, cfgFn, t)
	txn := app.StartTransaction("hello", nil, nil)
	txn.(internal.AddAgentAttributer).AddAgentAttribute(internal.AttributeAWSLambdaARN, "thearn", nil)
	txn.End()
	payloadJSON, err := txn.(serverlessTransaction).serverlessJSON("executionEnv")
	if nil != err {
		t.Fatal(err)
	}
	var payload []interface{}
	err = json.Unmarshal(payloadJSON, &payload)
	if nil != err {
		t.Fatal(err)
	}
	if len(payload) != 4 {
		t.Fatal(payload)
	}
	if v := payload[0].(float64); v != lambdaMetadataVersion {
		t.Fatal(payload[0], lambdaMetadataVersion)
	}
	if v := payload[1].(string); v != "NR_LAMBDA_MONITORING" {
		t.Fatal(payload[1])
	}
	dataJSON, err := decodeUncompress(payload[3].(string))
	if nil != err {
		t.Fatal(err)
	}
	var data map[string]interface{}
	err = json.Unmarshal(dataJSON, &data)
	if nil != err {
		t.Fatal(err)
	}
	// Data should contain txn event and metrics.  Timestamps make exact
	// JSON comparison tough.
	if _, ok := data["metric_data"]; !ok {
		t.Fatal(data)
	}
	if _, ok := data["analytic_event_data"]; !ok {
		t.Fatal(data)
	}

	metadata, ok := payload[2].(map[string]interface{})
	if !ok {
		t.Fatal(payload[2])
	}
	if v, ok := metadata["metadata_version"].(float64); !ok || v != float64(lambdaMetadataVersion) {
		t.Fatal(metadata["metadata_version"])
	}
	if v, ok := metadata["arn"].(string); !ok || v != "thearn" {
		t.Fatal(metadata["arn"])
	}
	if v, ok := metadata["protocol_version"].(float64); !ok || v != float64(internal.ProcotolVersion) {
		t.Fatal(metadata["protocol_version"])
	}
	if v, ok := metadata["execution_environment"].(string); !ok || v != "executionEnv" {
		t.Fatal(metadata["execution_environment"])
	}
	if v, ok := metadata["agent_version"].(string); !ok || v != Version {
		t.Fatal(metadata["agent_version"])
	}
	if v, ok := metadata["agent_language"].(string); !ok || v != agentLanguage {
		t.Fatal(metadata["agent_language"])
	}
}

func TestServerlessJSONMissingARN(t *testing.T) {
	// serverlessPayloadJSON should not panic if the Lambda ARN is missing.
	// The Lambda ARN is not expected to be missing, but to be safe we need
	// to ensure that txn.Attrs.Agent.StringVal won't panic if its not
	// there.
	cfgFn := func(cfg *Config) {
		cfg.ServerlessMode.Enabled = true
	}
	app := testApp(nil, cfgFn, t)
	txn := app.StartTransaction("hello", nil, nil)
	txn.End()
	payloadJSON, err := txn.(serverlessTransaction).serverlessJSON("executionEnv")
	if nil != err {
		t.Fatal(err)
	}
	if nil == payloadJSON {
		t.Error("missing JSON")
	}
}

func BenchmarkServerlessJSON(b *testing.B) {
	cfgFn := func(cfg *Config) {
		cfg.ServerlessMode.Enabled = true
	}
	app := testApp(nil, cfgFn, b)
	txn := app.StartTransaction("hello", nil, nil)
	txn.(internal.AddAgentAttributer).AddAgentAttribute(internal.AttributeAWSLambdaARN, "thearn", nil)
	segment := StartSegment(txn, "mySegment")
	segment.End()
	txn.End()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		txn.(serverlessTransaction).serverlessJSON("executionEnv")
	}
}

func validSampler(s internal.AdaptiveSampler) bool {
	_, isSampleEverything := s.(internal.SampleEverything)
	_, isSampleNothing := s.(internal.SampleEverything)
	return (nil != s) && !isSampleEverything && !isSampleNothing
}

func TestServerlessConnectReply(t *testing.T) {
	cfg := NewConfig("", "")
	cfg.ServerlessMode.ApdexThreshold = 2 * time.Second
	cfg.ServerlessMode.AccountID = "the-account-id"
	cfg.ServerlessMode.TrustedAccountKey = "the-trust-key"
	cfg.ServerlessMode.PrimaryAppID = "the-primary-app"
	reply := newServerlessConnectReply(cfg)
	if reply.ApdexThresholdSeconds != 2 {
		t.Error(reply.ApdexThresholdSeconds)
	}
	if reply.AccountID != "the-account-id" {
		t.Error(reply.AccountID)
	}
	if reply.TrustedAccountKey != "the-trust-key" {
		t.Error(reply.TrustedAccountKey)
	}
	if reply.PrimaryAppID != "the-primary-app" {
		t.Error(reply.PrimaryAppID)
	}
	if !validSampler(reply.AdaptiveSampler) {
		t.Error(reply.AdaptiveSampler)
	}

	// Now test the defaults:
	cfg = NewConfig("", "")
	reply = newServerlessConnectReply(cfg)
	if reply.ApdexThresholdSeconds != 0.5 {
		t.Error(reply.ApdexThresholdSeconds)
	}
	if reply.AccountID != "" {
		t.Error(reply.AccountID)
	}
	if reply.TrustedAccountKey != "" {
		t.Error(reply.TrustedAccountKey)
	}
	if reply.PrimaryAppID != "Unknown" {
		t.Error(reply.PrimaryAppID)
	}
	if !validSampler(reply.AdaptiveSampler) {
		t.Error(reply.AdaptiveSampler)
	}
}
