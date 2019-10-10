// +build go1.13

package newrelic

import (
	"fmt"
	"testing"

	"github.com/newrelic/go-agent/internal"
)

type socketError struct{}

func (e socketError) Error() string { return "socket error" }

func TestNoticedWrappedError(t *testing.T) {
	gamma := func() error { return socketError{} }
	beta := func() error { return fmt.Errorf("problem in beta: %w", gamma()) }
	alpha := func() error { return fmt.Errorf("problem in alpha: %w", beta()) }

	app := testApp(nil, nil, t)
	txn := app.StartTransaction("hello", nil, nil)
	err := txn.NoticeError(alpha())
	if nil != err {
		t.Error(err)
	}
	txn.End()
	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "OtherTransaction/Go/hello",
		Msg:     "problem in alpha: problem in beta: socket error",
		Klass:   "newrelic.socketError",
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "newrelic.socketError",
			"error.message":   "problem in alpha: problem in beta: socket error",
			"transactionName": "OtherTransaction/Go/hello",
		},
	}})
	app.ExpectMetrics(t, backgroundErrorMetrics)
}
