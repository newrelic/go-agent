// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package newrelic_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
)

func Example() {
	// Create your application using your preferred app name, license key, and
	// any other configuration options.
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Example Application"),
		newrelic.ConfigLicense("__YOUR_NEW_RELIC_LICENSE_KEY__"),
		newrelic.ConfigDebugLogger(os.Stdout),
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

		// Do some work
		time.Sleep(time.Second)
	}()

	// WrapHandler and WrapHandleFunc make it easy to instrument inbound
	// web requests handled by the http standard library without calling
	// Application.StartTransaction.  Popular framework instrumentation
	// packages exist in the v3/integrations directory.
	http.HandleFunc(newrelic.WrapHandleFunc(app, "", func(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, "this is the index page")
	}))
	helloHandler := func(w http.ResponseWriter, req *http.Request) {
		// WrapHandler and WrapHandleFunc add the transaction to the
		// inbound request's context.  Access the transaction using
		// FromContext to add attributes, create segments, and notice.
		// errors.
		txn := newrelic.FromContext(req.Context())

		func() {
			// Segments help you understand where the time in your
			// transaction is being spent.  You can use them to time
			// functions or arbitrary blocks of code.
			defer txn.StartSegment("helperFunction").End()
		}()

		io.WriteString(w, "hello world")
	}
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/hello", helloHandler))
	http.ListenAndServe(":8000", nil)
}

func currentTransaction() *newrelic.Transaction {
	return nil
}

var txn *newrelic.Transaction

func ExampleNewRoundTripper() {
	client := &http.Client{}
	// The http.RoundTripper returned by NewRoundTripper instruments all
	// requests done by this client with external segments.
	client.Transport = newrelic.NewRoundTripper(client.Transport)

	request, _ := http.NewRequest("GET", "http://example.com", nil)

	// Be sure to add the current Transaction to each request's context so
	// the Transport has access to it.
	txn := currentTransaction()
	request = newrelic.RequestWithTransactionContext(request, txn)

	client.Do(request)
}

func getApp() *newrelic.Application {
	return nil
}

func ExampleBrowserTimingHeader() {
	handler := func(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, "<html><head>")
		// The New Relic browser javascript should be placed as high in the
		// HTML as possible.  We suggest including it immediately after the
		// opening <head> tag and any <meta charset> tags.
		txn := newrelic.FromContext(req.Context())
		hdr := txn.BrowserTimingHeader()
		// BrowserTimingHeader() will always return a header whose methods can
		// be safely called.
		if js := hdr.WithTags(); js != nil {
			w.Write(js)
		}
		io.WriteString(w, "</head><body>browser header page</body></html>")
	}
	http.HandleFunc(newrelic.WrapHandleFunc(getApp(), "/browser", handler))
	http.ListenAndServe(":8000", nil)
}

func ExampleDatastoreSegment() {
	txn := currentTransaction()
	ds := &newrelic.DatastoreSegment{
		StartTime: txn.StartSegmentNow(),
		// Product, Collection, and Operation are the primary metric
		// aggregation fields which we encourage you to populate.
		Product:    newrelic.DatastoreMySQL,
		Collection: "users_table",
		Operation:  "SELECT",
	}
	// your database call here
	ds.End()
}

func ExampleMessageProducerSegment() {
	txn := currentTransaction()
	seg := &newrelic.MessageProducerSegment{
		StartTime:       txn.StartSegmentNow(),
		Library:         "RabbitMQ",
		DestinationType: newrelic.MessageExchange,
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
	txn.NoticeError(newrelic.Error{
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
	segment := newrelic.StartExternalSegment(txn, request)
	response, _ := client.Do(request)
	segment.Response = response
	segment.End()
}

// StartExternalSegment is the recommend way of creating ExternalSegments. If
// you don't have access to an http.Request, however, you may create an
// ExternalSegment and control the URL manually.
func ExampleExternalSegment_url() {
	txn := currentTransaction()
	segment := newrelic.ExternalSegment{
		StartTime: txn.StartSegmentNow(),
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
	segment := newrelic.StartExternalSegment(txn, request)
	response, _ := client.Do(request)
	segment.Response = response
	segment.End()
}

func ExampleStartExternalSegment_context() {
	txn := currentTransaction()
	request, _ := http.NewRequest("GET", "http://www.example.com", nil)

	// If the transaction is added to the request's context then it does not
	// need to be provided as a parameter to StartExternalSegment.
	request = newrelic.RequestWithTransactionContext(request, txn)
	segment := newrelic.StartExternalSegment(nil, request)

	client := &http.Client{}
	response, _ := client.Do(request)
	segment.Response = response
	segment.End()
}

func doSendRequest(*http.Request) int { return 418 }

// Use ExternalSegment.SetStatusCode when you do not have access to an
// http.Response and still want to record the response status code.
func ExampleExternalSegment_SetStatusCode() {
	txn := currentTransaction()
	request, _ := http.NewRequest("GET", "http://www.example.com", nil)
	segment := newrelic.StartExternalSegment(txn, request)
	statusCode := doSendRequest(request)
	segment.SetStatusCode(statusCode)
	segment.End()
}

func ExampleTransaction_SetWebRequest() {
	app := getApp()
	txn := app.StartTransaction("My-Transaction")
	txn.SetWebRequest(newrelic.WebRequest{
		Header:    http.Header{},
		URL:       &url.URL{Path: "path"},
		Method:    "GET",
		Transport: newrelic.TransportHTTP,
	})
}

func ExampleTransaction_SetWebRequestHTTP() {
	app := getApp()
	inboundRequest, _ := http.NewRequest("GET", "http://example.com", nil)
	txn := app.StartTransaction("My-Transaction")
	// Mark transaction as a web transaction, record attributes based on the
	// inbound request, and read any available distributed tracing headers.
	txn.SetWebRequestHTTP(inboundRequest)
}

// Sometimes there is no inbound request, but you may still wish to set a
// Transaction as a web request.  Passing nil to Transaction.SetWebRequestHTTP
// allows you to do just this.
func ExampleTransaction_SetWebRequestHTTP_nil() {
	app := getApp()
	txn := app.StartTransaction("My-Transaction")
	// Mark transaction as a web transaction, but do not record attributes
	// based on an inbound request or read distributed tracing headers.
	txn.SetWebRequestHTTP(nil)
}

// This example (modified from the WrapHandle instrumentation) demonstrates how
// you can replace an http.ResponseWriter in order to capture response headers
// and notice errors based on status code.
//
// Note that this is just an example and that WrapHandle and WrapHandleFunc
// perform this instrumentation for you.
func ExampleTransaction_SetWebResponse() {
	app := getApp()
	handler := http.FileServer(http.Dir("/tmp"))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Begin a Transaction.
		txn := app.StartTransaction("index")
		defer txn.End()

		// Set the transaction as a web request, gather attributes based on the
		// request, and read incoming distributed trace headers.
		txn.SetWebRequestHTTP(r)

		// Prepare to capture attributes, errors, and headers from the
		// response.
		w = txn.SetWebResponse(w)

		// Add the Transaction to the http.Request's Context.
		r = newrelic.RequestWithTransactionContext(r, txn)

		// The http.ResponseWriter passed to ServeHTTP has been replaced with
		// the new instrumented http.ResponseWriter.
		handler.ServeHTTP(w, r)
	})
}

// The order in which the ConfigOptions are added plays an important role when
// using ConfigFromEnvironment.
func ExampleConfigFromEnvironment() {
	os.Setenv("NEW_RELIC_ENABLED", "true")

	// Application is disabled.  Enabled is first set to true from
	// ConfigFromEnvironment then set to false from ConfigEnabled.
	_, _ = newrelic.NewApplication(
		newrelic.ConfigFromEnvironment(),
		newrelic.ConfigEnabled(false),
	)

	// Application is enabled.  Enabled is first set to false from
	// ConfigEnabled then set to true from ConfigFromEnvironment.
	_, _ = newrelic.NewApplication(
		newrelic.ConfigEnabled(false),
		newrelic.ConfigFromEnvironment(),
	)
}

func ExampleNewApplication_configOptionOrder() {
	// In this case, the Application will be disabled because the disabling
	// ConfigOption is last.
	_, _ = newrelic.NewApplication(
		newrelic.ConfigEnabled(true),
		newrelic.ConfigEnabled(false),
	)
}

// While many ConfigOptions are provided for you, it is also possible to create
// your own.  This is necessary if you have complex configuration needs.
func ExampleConfigOption_custom() {
	_, _ = newrelic.NewApplication(
		newrelic.ConfigAppName("Example App"),
		newrelic.ConfigLicense("__YOUR_NEW_RELIC_LICENSE_KEY__"),
		func(cfg *newrelic.Config) {
			// Set specific Config fields inside a custom ConfigOption.
			cfg.Attributes.Enabled = false
			cfg.HighSecurity = true
		},
	)
}

// Setting the Config.Error field will cause the NewApplication function to
// return an error.
func ExampleConfigOption_errors() {
	myError := errors.New("oops")

	_, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Example App"),
		newrelic.ConfigLicense("__YOUR_NEW_RELIC_LICENSE_KEY__"),
		func(cfg *newrelic.Config) {
			cfg.Error = myError
		},
	)

	fmt.Printf("%t", err == myError)
	// Output: true
}

func ExampleTransaction_StartSegmentNow() {
	txn := currentTransaction()
	seg := &newrelic.MessageProducerSegment{
		// The value returned from Transaction.StartSegmentNow is used for the
		// StartTime field on any segment.
		StartTime:       txn.StartSegmentNow(),
		Library:         "RabbitMQ",
		DestinationType: newrelic.MessageExchange,
		DestinationName: "myExchange",
	}
	// add message to queue here
	seg.End()
}

// Passing a new Transaction reference directly to another goroutine.
func ExampleTransaction_NewGoroutine() {
	go func(txn *newrelic.Transaction) {
		defer txn.StartSegment("async").End()
		// Do some work
		time.Sleep(100 * time.Millisecond)
	}(txn.NewGoroutine())
}

// Passing a new Transaction reference on a channel to another goroutine.
func ExampleTransaction_NewGoroutine_channel() {
	ch := make(chan *newrelic.Transaction)
	go func() {
		txn := <-ch
		defer txn.StartSegment("async").End()
		// Do some work
		time.Sleep(100 * time.Millisecond)
	}()
	ch <- txn.NewGoroutine()
}

// Sometimes it is not possible to call txn.NewGoroutine() before the goroutine
// has started.  In this case, it is okay to call the method from inside the
// newly started goroutine.
func ExampleTransaction_NewGoroutine_insideGoroutines() {
	// async will always be called using `go async(ctx)`
	async := func(ctx context.Context) {
		txn := newrelic.FromContext(ctx)
		txn = txn.NewGoroutine()
		defer txn.StartSegment("async").End()

		// Do some work
		time.Sleep(100 * time.Millisecond)
	}
	ctx := newrelic.NewContext(context.Background(), currentTransaction())
	go async(ctx)
}
