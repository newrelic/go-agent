package newrelic

import (
	"fmt"
	"net/http"
	"testing"
)

func TestIsEnded(t *testing.T) {
	tests := []struct {
		name     string
		txn      *Transaction
		expected bool
	}{
		{"txn is nil", nil, true},
		{"thread is nil", &Transaction{thread: nil}, true},
		{"txn.thread.txn is nil", &Transaction{thread: &thread{}}, true},
		{"txn.thread.txn.finished is true", &Transaction{thread: &thread{txn: &txn{finished: true}}}, true},
		{"txn.thread.txn.finished is false", &Transaction{thread: &thread{txn: &txn{finished: false}}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.txn.IsEnded()
			if result != tt.expected {
				t.Errorf("IsEnded() = %v; want %v", result, tt.expected)
			}
		})
	}
}

func TestTransaction_MethodsWithNilTransaction(t *testing.T) {
	var nilTxn *Transaction

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panics should not occur on methods of Transaction: %v", r)
		}
	}()

	// Ensure no panic occurs when calling methods on a nil transaction
	nilTxn.End()
	nilTxn.SetOption()
	nilTxn.Ignore()
	nilTxn.IgnoreApdex()
	nilTxn.SetName("test")
	name := nilTxn.Name()
	if name != "" {
		t.Errorf("expected empty string, got %s", name)
	}
	nilTxn.NoticeError(fmt.Errorf("test error"))
	nilTxn.NoticeExpectedError(fmt.Errorf("test expected error"))
	nilTxn.AddAttribute("key", "value")
	nilTxn.SetUserID("user123")
	nilTxn.RecordLog(LogData{})
	nilTxn.SetWebRequestHTTP(nil)
	nilTxn.SetWebRequest(WebRequest{})
	nilTxn.SetWebResponse(nil)
	nilTxn.StartSegmentNow()
	nilTxn.StartSegment("test segment")
	nilTxn.InsertDistributedTraceHeaders(http.Header{})
	nilTxn.AcceptDistributedTraceHeaders(TransportHTTP, http.Header{})
	err := nilTxn.AcceptDistributedTraceHeadersFromJSON(TransportHTTP, "{}")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	app := nilTxn.Application()
	if app != nil {
		t.Errorf("expected nil, got %v", app)
	}
	bth := nilTxn.BrowserTimingHeader()
	if bth != nil {
		t.Errorf("expected nil, got %v", bth)
	}
	newTxn := nilTxn.NewGoroutine()
	if newTxn != nil {
		t.Errorf("expected nil, got %v", newTxn)
	}
	traceMetadata := nilTxn.GetTraceMetadata()
	if traceMetadata != (TraceMetadata{}) {
		t.Errorf("expected empty TraceMetadata, got %v", traceMetadata)
	}
	linkingMetadata := nilTxn.GetLinkingMetadata()
	if linkingMetadata != (LinkingMetadata{}) {
		t.Errorf("expected empty LinkingMetadata, got %v", linkingMetadata)
	}
	isSampled := nilTxn.IsSampled()
	if isSampled {
		t.Errorf("expected false, got %v", isSampled)
	}
	csecAttributes := nilTxn.GetCsecAttributes()
	if csecAttributes != nil {
		t.Errorf("expected nil, got %v", csecAttributes)
	}
	nilTxn.SetCsecAttributes("key", "value")
}
