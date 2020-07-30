package newrelic

import (
	"context"
	"testing"
)

func TestNewContextNilTransaction(t *testing.T) {
	bkg := context.Background()
	if ctx := NewContext(bkg, nil); ctx != bkg {
		t.Error("ctx was updated by a nil transaction")
	}
}

func TestNewContextEmptyTransaction(t *testing.T) {
	// test that using an empty transaction does not panic
	NewContext(context.Background(), new(Transaction))
}
