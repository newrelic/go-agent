package nrzap

import (
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/newrelic/go-agent/v3/internal"
	"github.com/newrelic/go-agent/v3/internal/integrationsupport"
	"github.com/newrelic/go-agent/v3/newrelic"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestBackgroundLogger(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)

	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), os.Stdout, zap.InfoLevel)
	wrappedCore, err := WrapBackgroundCore(core, app.Application)
	if err != nil {
		t.Error(err)
	}

	logger := zap.New(wrappedCore)

	err = errors.New("this is a test error")
	msg := "this is a test error message"

	// for background logging:
	logger.Error(msg, zap.Error(err), zap.String("test-key", "test-val"))
	logger.Sync()

	app.ExpectLogEvents(t, []internal.WantLog{
		{
			Severity:  zap.ErrorLevel.String(),
			Message:   msg,
			Timestamp: internal.MatchAnyUnixMilli,
		},
	})
}

func TestBackgroundLoggerSugared(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)

	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(os.Stdout), zap.InfoLevel)

	backgroundCore, err := WrapBackgroundCore(core, app.Application)
	if err != nil && err != ErrNilApp {
		t.Fatal(err)
	}

	logger := zap.New(backgroundCore).Sugar()

	err = errors.New("this is a test error")
	msg := "this is a test error message"

	// for background logging:
	logger.Error(msg, zap.Error(err), zap.String("test-key", "test-val"))
	logger.Sync()

	app.ExpectLogEvents(t, []internal.WantLog{
		{
			Severity:  zap.ErrorLevel.String(),
			Message:   `this is a test error message{error 26 0  this is a test error} {test-key 15 0 test-val <nil>}`,
			Timestamp: internal.MatchAnyUnixMilli,
		},
	})
}

func TestBackgroundLoggerNilApp(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)

	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), os.Stdout, zap.InfoLevel)
	wrappedCore, err := WrapBackgroundCore(core, nil)
	if err != ErrNilApp {
		t.Error(err)
	}
	if wrappedCore == nil {
		t.Error("when the app is nil, the core returned should still be valid")
	}

	logger := zap.New(wrappedCore)

	err = errors.New("this is a test error")
	msg := "this is a test error message"

	// for background logging:
	logger.Error(msg, zap.Error(err), zap.String("test-key", "test-val"))
	logger.Sync()

	// Expect no log events in logger without app in core
	app.ExpectLogEvents(t, []internal.WantLog{})
}

func TestTransactionLogger(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)

	txn := app.StartTransaction("test transaction")
	txnMetadata := txn.GetTraceMetadata()

	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), os.Stdout, zap.InfoLevel)
	wrappedCore, err := WrapTransactionCore(core, txn)
	if err != nil {
		t.Error(err)
	}

	logger := zap.New(wrappedCore)

	err = errors.New("this is a test error")
	msg := "this is a test error message"

	// for background logging:
	logger.Error(msg, zap.Error(err), zap.String("test-key", "test-val"))
	logger.Sync()

	// ensure txn gets written to an event and logs get released
	txn.End()

	app.ExpectLogEvents(t, []internal.WantLog{
		{
			Attributes: map[string]interface{}{
				"test-key": "test-val",
				"error":    "this is a test error",
			},
			Severity:  zap.ErrorLevel.String(),
			Message:   msg,
			Timestamp: internal.MatchAnyUnixMilli,
			TraceID:   txnMetadata.TraceID,
			SpanID:    txnMetadata.SpanID,
		},
	})
}

func TestTransactionLoggerWithFields(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
		newrelic.ConfigZapAttributesEncoder(true),
	)

	txn := app.StartTransaction("test transaction")
	txnMetadata := txn.GetTraceMetadata()

	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), os.Stdout, zap.InfoLevel)
	wrappedCore, err := WrapTransactionCore(core, txn)
	if err != nil {
		t.Error(err)
	}

	wrappedCore = wrappedCore.With([]zapcore.Field{
		zap.String("foo", "bar"),
	})

	logger := zap.New(wrappedCore)

	msg := "this is a test info message"

	// for background logging:
	logger.Info(msg,
		zap.String("region", "region-test-2"),
		zap.Any("anyValue", map[string]interface{}{"pi": 3.14, "duration": 2 * time.Second}),
		zap.Duration("duration", 1*time.Second),
		zap.Int("int", 123),
		zap.Bool("bool", true),
	)

	logger.Sync()

	// ensure txn gets written to an event and logs get released
	txn.End()

	app.ExpectLogEvents(t, []internal.WantLog{
		{
			Attributes: map[string]interface{}{
				"region":   "region-test-2",
				"anyValue": map[string]interface{}{"pi": 3.14, "duration": 2 * time.Second},
				"duration": "1s",
				"int":      int64(123),
				"bool":     true,
				"foo":      "bar",
			},
			Severity:  zap.InfoLevel.String(),
			Message:   msg,
			Timestamp: internal.MatchAnyUnixMilli,
			TraceID:   txnMetadata.TraceID,
			SpanID:    txnMetadata.SpanID,
		},
	})
}

func TestTransactionLoggerWithFieldsAtHarvestTime(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
		newrelic.ConfigZapAttributesEncoder(false),
	)

	txn := app.StartTransaction("test transaction")
	txnMetadata := txn.GetTraceMetadata()

	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), os.Stdout, zap.InfoLevel)
	wrappedCore, err := WrapTransactionCore(core, txn)
	if err != nil {
		t.Error(err)
	}

	logger := zap.New(wrappedCore)

	msg := "this is a test info message"

	// for background logging:
	logger.Info(msg,
		zap.String("region", "region-test-2"),
		zap.Any("anyValue", map[string]interface{}{"pi": 3.14, "duration": 2 * time.Second}),
		zap.Duration("duration", 1*time.Second),
		zap.Int("int", 123),
		zap.Bool("bool", true),
	)

	logger.Sync()

	// ensure txn gets written to an event and logs get released
	txn.End()

	app.ExpectLogEvents(t, []internal.WantLog{
		{
			Attributes: map[string]interface{}{
				"region":   "region-test-2",
				"anyValue": map[string]interface{}{"pi": 3.14, "duration": 2 * time.Second},
				"duration": "1s",
				"int":      int64(123),
				"bool":     true,
			},
			Severity:  zap.InfoLevel.String(),
			Message:   msg,
			Timestamp: internal.MatchAnyUnixMilli,
			TraceID:   txnMetadata.TraceID,
			SpanID:    txnMetadata.SpanID,
		},
	})
}

func TestTransactionLoggerNilTxn(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)

	txn := app.StartTransaction("test transaction")
	//txnMetadata := txn.GetTraceMetadata()

	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), os.Stdout, zap.InfoLevel)
	wrappedCore, err := WrapTransactionCore(core, nil)
	if err != ErrNilTxn {
		t.Error(err)
	}
	if wrappedCore == nil {
		t.Error("when the txn is nil, the core returned should still be valid")
	}

	logger := zap.New(wrappedCore)

	err = errors.New("this is a test error")
	msg := "this is a test error message"

	// for background logging:
	logger.Error(msg, zap.Error(err), zap.String("test-key", "test-val"))
	logger.Sync()

	// ensure txn gets written to an event and logs get released
	txn.End()

	// no data should be captured when nil txn passed to wrapped logger
	app.ExpectLogEvents(t, []internal.WantLog{})
}

func TestWith(t *testing.T) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)

	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), os.Stdout, zap.InfoLevel)
	wrappedCore, err := WrapBackgroundCore(core, app.Application)
	if err != nil {
		t.Error(err)
	}

	newCore := wrappedCore.With([]zapcore.Field{})

	if newCore == core {
		t.Error("core was not coppied during With() operaion")
	}
}

func BenchmarkZapBaseline(b *testing.B) {
	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(io.Discard), zap.InfoLevel)
	logger := zap.New(core)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.Info("this is a test message")
	}
}

func BenchmarkFieldConversion(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		convertFieldWithMapEncoder([]zap.Field{
			zap.String("test-key", "test-val"),
			zap.Any("test-key", map[string]interface{}{"pi": 3.14, "duration": 2 * time.Second}),
		})
	}
}

func BenchmarkFieldUnmarshalling(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		convertFieldsAtHarvestTime([]zap.Field{
			zap.String("test-key", "test-val"),
			zap.Any("test-key", map[string]interface{}{"pi": 3.14, "duration": 2 * time.Second}),
		})

	}
}

func BenchmarkZapWithAttribute(b *testing.B) {
	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(io.Discard), zap.InfoLevel)
	logger := zap.New(core)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.Info("this is a test message", zap.Any("test-key", "test-val"))
	}
}

func BenchmarkZapWrappedCore(b *testing.B) {
	app := integrationsupport.NewTestApp(integrationsupport.SampleEverythingReplyFn,
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)

	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(io.Discard), zap.InfoLevel)
	wrappedCore, err := WrapBackgroundCore(core, app.Application)
	if err != nil {
		b.Error(err)
	}

	logger := zap.New(wrappedCore)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		logger.Info("this is a test message")
	}
}
