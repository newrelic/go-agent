// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
)

func index(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "hello world")
}

func versionHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "New Relic Go Agent Version: "+newrelic.Version)
}

func noticeError(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "noticing an error")

	txn := newrelic.FromContext(r.Context())
	txn.NoticeError(errors.New("my error message"))
}

func noticeExpectedError(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "noticing an error")

	txn := newrelic.FromContext(r.Context())
	txn.NoticeExpectedError(errors.New("my expected error message"))
}

func noticeErrorWithAttributes(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "noticing an error")

	txn := newrelic.FromContext(r.Context())
	txn.NoticeError(newrelic.Error{
		Message: "uh oh. something went very wrong",
		Class:   "errors are aggregated by class",
		Attributes: map[string]interface{}{
			"important_number": 97232,
			"relevant_string":  "zap",
		},
	})
}

func customEvent(w http.ResponseWriter, r *http.Request) {
	txn := newrelic.FromContext(r.Context())

	io.WriteString(w, "recording a custom event")

	txn.Application().RecordCustomEvent("my_event_type", map[string]interface{}{
		"myString": "hello",
		"myFloat":  0.603,
		"myInt":    123,
		"myBool":   true,
	})
}

func setName(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "changing the transaction's name")

	txn := newrelic.FromContext(r.Context())
	txn.SetName("other-name")
}

func addAttribute(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "adding attributes")

	txn := newrelic.FromContext(r.Context())
	txn.AddAttribute("myString", "hello")
	txn.AddAttribute("myInt", 123)
}

func addSpanAttribute(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "adding span attributes")

	txn := newrelic.FromContext(r.Context())
	sgmt := txn.StartSegment("segment1")
	defer sgmt.End()
	sgmt.AddAttribute("mySpanString", "hello")
	sgmt.AddAttribute("mySpanInt", 123)
}

func ignore(w http.ResponseWriter, r *http.Request) {
	if coinFlip := (0 == rand.Intn(2)); coinFlip {
		txn := newrelic.FromContext(r.Context())
		txn.Ignore()
		io.WriteString(w, "ignoring the transaction")
	} else {
		io.WriteString(w, "not ignoring the transaction")
	}
}

func segments(w http.ResponseWriter, r *http.Request) {
	txn := newrelic.FromContext(r.Context())

	func() {
		defer txn.StartSegment("f1").End()

		func() {
			defer txn.StartSegment("f2").End()

			io.WriteString(w, "segments!")
			time.Sleep(10 * time.Millisecond)
		}()
		time.Sleep(15 * time.Millisecond)
	}()
	time.Sleep(20 * time.Millisecond)
}

func mysql(w http.ResponseWriter, r *http.Request) {
	txn := newrelic.FromContext(r.Context())
	s := newrelic.DatastoreSegment{
		StartTime: txn.StartSegmentNow(),
		// Product, Collection, and Operation are the most important
		// fields to populate because they are used in the breakdown
		// metrics.
		Product:    newrelic.DatastoreMySQL,
		Collection: "users",
		Operation:  "INSERT",

		ParameterizedQuery: "INSERT INTO users (name, age) VALUES ($1, $2)",
		QueryParameters: map[string]interface{}{
			"name": "Dracula",
			"age":  439,
		},
		Host:         "mysql-server-1",
		PortPathOrID: "3306",
		DatabaseName: "my_database",
	}
	defer s.End()

	time.Sleep(20 * time.Millisecond)
	io.WriteString(w, `performing fake query "INSERT * from users"`)
}

func message(w http.ResponseWriter, r *http.Request) {
	txn := newrelic.FromContext(r.Context())
	s := newrelic.MessageProducerSegment{
		StartTime:       txn.StartSegmentNow(),
		Library:         "RabbitMQ",
		DestinationType: newrelic.MessageQueue,
		DestinationName: "myQueue",
	}
	defer s.End()

	time.Sleep(20 * time.Millisecond)
	io.WriteString(w, `producing a message queue message`)
}

func external(w http.ResponseWriter, r *http.Request) {
	txn := newrelic.FromContext(r.Context())
	req, _ := http.NewRequest("GET", "http://example.com", nil)

	// Using StartExternalSegment is recommended because it does distributed
	// tracing header setup, but if you don't have an *http.Request and
	// instead only have a url string then you can start the external
	// segment like this:
	//
	// es := newrelic.ExternalSegment{
	// 	StartTime: txn.StartSegmentNow(),
	// 	URL:       urlString,
	// }
	//
	es := newrelic.StartExternalSegment(txn, req)
	resp, err := http.DefaultClient.Do(req)
	es.End()

	if err != nil {
		io.WriteString(w, err.Error())
		return
	}
	defer resp.Body.Close()
	io.Copy(w, resp.Body)
}

func roundtripper(w http.ResponseWriter, r *http.Request) {
	// NewRoundTripper allows you to instrument external calls without
	// calling StartExternalSegment by modifying the http.Client's Transport
	// field.  If the Transaction parameter is nil, the RoundTripper
	// returned will look for a Transaction in the request's context (using
	// FromContext). This is recommended because it allows you to reuse the
	// same client for multiple transactions.
	client := &http.Client{}
	client.Transport = newrelic.NewRoundTripper(client.Transport)

	request, _ := http.NewRequest("GET", "http://example.com", nil)
	// Since the transaction is already added to the inbound request's
	// context by WrapHandleFunc, we just need to copy the context from the
	// inbound request to the external request.
	request = request.WithContext(r.Context())
	// Alternatively, if you don't want to copy entire context, and instead
	// wanted just to add the transaction to the external request's context,
	// you could do that like this:
	//
	//	txn := newrelic.FromContext(r.Context())
	//	request = newrelic.RequestWithTransactionContext(request, txn)

	resp, err := client.Do(request)
	if err != nil {
		io.WriteString(w, err.Error())
		return
	}
	defer resp.Body.Close()
	io.Copy(w, resp.Body)
}

func async(w http.ResponseWriter, r *http.Request) {
	txn := newrelic.FromContext(r.Context())
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func(txn *newrelic.Transaction) {
		defer wg.Done()
		defer txn.StartSegment("async").End()
		time.Sleep(100 * time.Millisecond)
	}(txn.NewGoroutine())

	segment := txn.StartSegment("wg.Wait")
	wg.Wait()
	segment.End()
	w.Write([]byte("done!"))
}

func customMetric(w http.ResponseWriter, r *http.Request) {
	txn := newrelic.FromContext(r.Context())
	for _, vals := range r.Header {
		for _, v := range vals {
			// This custom metric will have the name
			// "Custom/HeaderLength" in the New Relic UI.
			txn.Application().RecordCustomMetric("HeaderLength", float64(len(v)))
		}
	}
	io.WriteString(w, "custom metric recorded")
}

func browser(w http.ResponseWriter, r *http.Request) {
	txn := newrelic.FromContext(r.Context())
	hdr := txn.BrowserTimingHeader()
	// BrowserTimingHeader() will always return a header whose methods can
	// be safely called.
	if js := hdr.WithTags(); js != nil {
		w.Write(js)
	}
	io.WriteString(w, "browser header page")
}

func logTxnMessage(w http.ResponseWriter, r *http.Request) {
	txn := newrelic.FromContext(r.Context())
	txn.RecordLog(newrelic.LogData{
		Message:  "Log Message",
		Severity: "info",
	})

	io.WriteString(w, "A log message was recorded")
}

func main() {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Example App"),
		newrelic.ConfigFromEnvironment(),
		newrelic.ConfigDebugLogger(os.Stdout),
		newrelic.ConfigAppLogForwardingEnabled(true),
		newrelic.ConfigCodeLevelMetricsEnabled(true),
		newrelic.ConfigCodeLevelMetricsPathPrefix("go-agent/v3"),
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	http.HandleFunc(newrelic.WrapHandleFunc(app, "/", index))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/version", versionHandler))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/notice_error", noticeError))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/notice_expected_error", noticeExpectedError))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/notice_error_with_attributes", noticeErrorWithAttributes))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/custom_event", customEvent))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/set_name", setName))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/add_attribute", addAttribute))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/add_span_attribute", addSpanAttribute))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/ignore", ignore))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/segments", segments))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/mysql", mysql))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/external", external))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/roundtripper", roundtripper))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/custommetric", customMetric))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/browser", browser))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/async", async))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/message", message))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/log", logTxnMessage))

	//loc := newrelic.ThisCodeLocation()
	backgroundCache := newrelic.NewCachedCodeLocation()
	http.HandleFunc("/background", func(w http.ResponseWriter, req *http.Request) {
		// Transactions started without an http.Request are classified as
		// background transactions.
		txn := app.StartTransaction("background", backgroundCache.WithThisCodeLocation())
		defer txn.End()

		io.WriteString(w, "background transaction")
		time.Sleep(150 * time.Millisecond)
	})

	http.HandleFunc("/background_log", func(w http.ResponseWriter, req *http.Request) {
		// Logs that occur outside of a transaction are classified as
		// background logs.

		app.RecordLog(newrelic.LogData{
			Message:  "Background Log Message",
			Severity: "info",
		})

		io.WriteString(w, "A background log message was recorded")
	})

	http.ListenAndServe(":8000", nil)
}
