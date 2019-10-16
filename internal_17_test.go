// +build go1.7

package newrelic

import (
	"net/http"
	"testing"

	"github.com/newrelic/go-agent/internal"
)

func myErrorHandler(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("my response"))
	// Ensure that the transaction is added to the request's context.
	if txn := FromContext(req.Context()); txn != nil {
		txn.NoticeError(myError{})
	}
}

func TestWrapHandleFunc(t *testing.T) {
	app := testApp(nil, nil, t)
	mux := http.NewServeMux()
	mux.HandleFunc(WrapHandleFunc(app.Application, helloPath, myErrorHandler))
	w := newCompatibleResponseRecorder()
	mux.ServeHTTP(w, helloRequest)

	out := w.Body.String()
	if "my response" != out {
		t.Error(out)
	}

	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "WebTransaction/Go/hello",
		Msg:     "my msg",
		Klass:   "newrelic.myError",
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "newrelic.myError",
			"error.message":   "my msg",
			"transactionName": "WebTransaction/Go/hello",
		},
		AgentAttributes: mergeAttributes(helloRequestAttributes, map[string]interface{}{
			"httpResponseCode": "200",
		}),
	}})
	app.ExpectMetrics(t, webErrorMetrics)
}

func TestWrapHandle(t *testing.T) {
	app := testApp(nil, nil, t)
	mux := http.NewServeMux()
	mux.Handle(WrapHandle(app.Application, helloPath, http.HandlerFunc(myErrorHandler)))
	w := newCompatibleResponseRecorder()
	mux.ServeHTTP(w, helloRequest)

	out := w.Body.String()
	if "my response" != out {
		t.Error(out)
	}

	app.ExpectErrors(t, []internal.WantError{{
		TxnName: "WebTransaction/Go/hello",
		Msg:     "my msg",
		Klass:   "newrelic.myError",
	}})
	app.ExpectErrorEvents(t, []internal.WantEvent{{
		Intrinsics: map[string]interface{}{
			"error.class":     "newrelic.myError",
			"error.message":   "my msg",
			"transactionName": "WebTransaction/Go/hello",
		},
		AgentAttributes: mergeAttributes(helloRequestAttributes, map[string]interface{}{
			"httpResponseCode": "200",
		}),
	}})
	app.ExpectMetrics(t, webErrorMetrics)
}

func TestWrapHandleNilApp(t *testing.T) {
	var app *Application
	mux := http.NewServeMux()
	mux.Handle(WrapHandle(app, helloPath, http.HandlerFunc(myErrorHandler)))
	w := newCompatibleResponseRecorder()
	mux.ServeHTTP(w, helloRequest)

	out := w.Body.String()
	if "my response" != out {
		t.Error(out)
	}
}
