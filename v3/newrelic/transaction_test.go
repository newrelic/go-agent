package newrelic

import "testing"

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
