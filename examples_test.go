// +build go1.7

package newrelic

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

func Example() {
	// Create your application using your license key and preferred app name.
	app, err := NewApplication(
		ConfigAppName("Example Application"),
		ConfigLicense("__YOUR_NEW_RELIC_LICENSE_KEY__"),
		ConfigDebugLogger(os.Stdout),
	)
	if nil != err {
		fmt.Println(err)
		os.Exit(1)
	}

	// Now you can use the Application to collect data!  Create transactions
	// to time inbound requests or background tasks. You can start and stop
	// transactions directly using Application.StartTransaction and
	// Transaction.End.
	func() {
		txn := app.StartTransaction("myTask")
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
			defer txn.StartSegment("helperFunction").End()
		}()

		io.WriteString(w, "hello world")
	}
	http.HandleFunc(WrapHandleFunc(app, "/hello", helloHandler))
	http.ListenAndServe(":8000", nil)
}

func currentTransaction() *Transaction {
	return nil
}

func ExampleNewRoundTripper() {
	client := &http.Client{}
	// The RoundTripper returned by NewRoundTripper instruments all requests
	// done by this client with external segments.
	client.Transport = NewRoundTripper(client.Transport)

	request, _ := http.NewRequest("GET", "http://example.com", nil)

	// Be sure to add the current Transaction to each request's context so
	// the Transport has access to it.
	txn := currentTransaction()
	request = RequestWithTransactionContext(request, txn)

	client.Do(request)
}

func getApp() *Application {
	return nil
}

func ExampleBrowserTimingHeader() {
	handler := func(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, "<html><head>")
		// The New Relic browser javascript should be placed as high in the
		// HTML as possible.  We suggest including it immediately after the
		// opening <head> tag and any <meta charset> tags.
		if txn := FromContext(req.Context()); nil != txn {
			hdr := txn.BrowserTimingHeader()
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

func ExampleMessageProducerSegment() {
	txn := currentTransaction()
	seg := &MessageProducerSegment{
		StartTime:       StartSegmentNow(txn),
		Library:         "RabbitMQ",
		DestinationType: MessageExchange,
		DestinationName: "myExchange",
	}
	// add message to queue here
	seg.End()
}

func ExampleError() {
	txn := currentTransaction()
	username := "gopher"
	e := fmt.Errorf("error unable to login user %s", username)
	// txn.NoticeError(newrelic.Error{...}) instead of txn.NoticeError(e)
	// allows more control over error fields.  Class is how errors are
	// aggregated and Attributes are added to the error event and error
	// trace.
	txn.NoticeError(Error{
		Message: e.Error(),
		Class:   "LoginError",
		Attributes: map[string]interface{}{
			"username": username,
		},
	})
}

func ExampleExternalSegment() {
	txn := currentTransaction()
	client := &http.Client{}
	request, _ := http.NewRequest("GET", "http://www.example.com", nil)
	segment := StartExternalSegment(txn, request)
	response, _ := client.Do(request)
	segment.Response = response
	segment.End()
}

// StartExternalSegment is the recommend way of creating ExternalSegments. If
// you don't have access to an http.Request, however, you may create an
// ExternalSegment and control the URL manually.
func ExampleExternalSegment_url() {
	txn := currentTransaction()
	segment := ExternalSegment{
		StartTime: StartSegmentNow(txn),
		// URL is parsed using url.Parse so it must include the protocol
		// scheme (eg. "http://").  The host of the URL is used to
		// create metrics.  Change the host to alter aggregation.
		URL: "http://www.example.com",
	}
	http.Get("http://www.example.com")
	segment.End()
}

func ExampleStartExternalSegment() {
	txn := currentTransaction()
	client := &http.Client{}
	request, _ := http.NewRequest("GET", "http://www.example.com", nil)
	segment := StartExternalSegment(txn, request)
	response, _ := client.Do(request)
	segment.Response = response
	segment.End()
}

func ExampleStartExternalSegment_context() {
	txn := currentTransaction()
	request, _ := http.NewRequest("GET", "http://www.example.com", nil)

	// If the transaction is added to the request's context then it does not
	// need to be provided as a parameter to StartExternalSegment.
	request = RequestWithTransactionContext(request, txn)
	segment := StartExternalSegment(nil, request)

	client := &http.Client{}
	response, _ := client.Do(request)
	segment.Response = response
	segment.End()
}

func ExampleTransaction_SetWebRequest() {
	app := getApp()
	txn := app.StartTransaction("My-Transaction")
	txn.SetWebRequest(WebRequest{
		Header:    http.Header{},
		URL:       &url.URL{Path: "path"},
		Method:    "GET",
		Transport: TransportHTTP,
	})
}

// The order in which the ConfigOptions are added plays an important role when
// using ConfigFromEnvironment.
func ExampleConfigFromEnvironment() {
	os.Setenv("NEW_RELIC_ENABLED", "true")

	// Applicaiton is disabled.  Enabled is first set to true from
	// ConfigFromEnvironment then set to false from ConfigEnabled.
	_, _ = NewApplication(
		ConfigFromEnvironment(),
		ConfigEnabled(false),
	)

	// Application is enabled.  Enabled is first set to false from
	// ConfigEnabled then set to true from ConfigFromEnvironment.
	_, _ = NewApplication(
		ConfigEnabled(false),
		ConfigFromEnvironment(),
	)
}
