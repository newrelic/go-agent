// +build go1.7

package newrelic

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

func Example() {
	// First create a Config.
	cfg := NewConfig("Example Application", "__YOUR_NEW_RELIC_LICENSE_KEY__")

	// Modify Config fields to control agent behavior.
	cfg.Logger = NewDebugLogger(os.Stdout)

	// Now use the Config the create an Application.
	app, err := NewApplication(cfg)
	if nil != err {
		fmt.Println(err)
		os.Exit(1)
	}

	// Now you can use the Application to collect data!  Create transactions
	// to time inbound requests or background tasks. You can start and stop
	// transactions directly using Application.StartTransaction and
	// Transaction.End.
	func() {
		txn := app.StartTransaction("myTask", nil, nil)
		defer txn.End()

		time.Sleep(time.Second)
	}()

	// WrapHandler and WrapHandleFunc make it easy to instrument inbound web
	// requests handled by the http standard library without calling
	// StartTransaction.  Popular framework instrumentation packages exist
	// in the _integrations directory.
	http.HandleFunc(WrapHandleFunc(app, "", func(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, "this is the index page")
	}))
	helloHandler := func(w http.ResponseWriter, req *http.Request) {
		// WrapHandler and WrapHandleFunc add the transaction to the
		// inbound request's context.  Access the transaction using
		// FromContext to add attributes, create segments, and notice.
		// errors.
		txn := FromContext(req.Context())

		func() {
			// Segments help you understand where the time in your
			// transaction is being spent.  You can use them to time
			// functions or arbitrary blocks of code.
			defer StartSegment(txn, "helperFunction").End()
		}()

		io.WriteString(w, "hello world")
	}
	http.HandleFunc(WrapHandleFunc(app, "/hello", helloHandler))
	http.ListenAndServe(":8000", nil)
}

func currentTransaction() Transaction {
	return nil
}

func ExampleNewRoundTripper() {
	client := &http.Client{}
	// The RoundTripper returned by NewRoundTripper instruments all requests
	// done by this client with external segments.
	client.Transport = NewRoundTripper(nil, client.Transport)

	request, _ := http.NewRequest("GET", "http://example.com", nil)

	// Be sure to add the current Transaction to each request's context so
	// the Transport has access to it.
	txn := currentTransaction()
	request = RequestWithTransactionContext(request, txn)

	client.Do(request)
}

func getApp() Application {
	return nil
}

func ExampleBrowserTimingHeader() {
	handler := func(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, "<html><head>")
		// The New Relic browser javascript should be placed as high in the
		// HTML as possible.  We suggest including it immediately after the
		// opening <head> tag and any <meta charset> tags.
		if txn := FromContext(req.Context()); nil != txn {
			hdr, err := txn.BrowserTimingHeader()
			if nil != err {
				log.Printf("unable to create browser timing header: %v", err)
			}
			// BrowserTimingHeader() will always return a header whose methods can
			// be safely called.
			if js := hdr.WithTags(); js != nil {
				w.Write(js)
			}
		}
		io.WriteString(w, "</head><body>browser header page</body></html>")
	}
	http.HandleFunc(WrapHandleFunc(getApp(), "/browser", handler))
	http.ListenAndServe(":8000", nil)
}

func ExampleDatastoreSegment() {
	txn := currentTransaction()
	ds := &DatastoreSegment{
		StartTime: StartSegmentNow(txn),
		// Product, Collection, and Operation are the primary metric
		// aggregation fields which we encourage you to populate.
		Product:    DatastoreMySQL,
		Collection: "users_table",
		Operation:  "SELECT",
	}
	// your database call here
	ds.End()
}
