package main

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"time"

	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/api/datastore"
	"github.com/newrelic/go-agent/log"

	// "github.com/Sirupsen/logrus"
	// _ "github.com/newrelic/go-agent/log/_nrlogrus"
)

var (
	app newrelic.Application
)

func init() {
	log.SetFile("stdout", log.LevelDebug)
	// logrus.SetOutput(os.Stdout)
	// logrus.SetLevel(logrus.DebugLevel)
}

func index(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "hello world")
}

func noticeError(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "noticing an error")

	if txn, ok := w.(newrelic.Transaction); ok {
		txn.NoticeError(errors.New("my error message"))
	}
}

func customEvent(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "recording a custom event")

	app.RecordCustomEvent("my_event_type", map[string]interface{}{
		"myString": "hello",
		"myFloat":  0.603,
		"myInt":    123,
		"myBool":   true,
	})
}

func setName(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "changing the transaction's name")

	if txn, ok := w.(newrelic.Transaction); ok {
		txn.SetName("other-name")
	}
}

func addAttribute(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "adding attributes")

	if txn, ok := w.(newrelic.Transaction); ok {
		txn.AddAttribute("myString", "hello")
		txn.AddAttribute("myInt", 123)
	}
}

func background(w http.ResponseWriter, r *http.Request) {
	// Transactions started without an http.Request are classified as
	// background transactions.
	txn := app.StartTransaction("background", nil, nil)
	defer txn.End()

	io.WriteString(w, "background transaction")
	time.Sleep(150 * time.Millisecond)
}

func ignore(w http.ResponseWriter, r *http.Request) {
	if coinFlip := (0 == rand.Intn(2)); coinFlip {
		if txn, ok := w.(newrelic.Transaction); ok {
			txn.Ignore()
		}
		io.WriteString(w, "ignoring the transaction")
	} else {
		io.WriteString(w, "not ignoring the transaction")
	}
}

func segments(w http.ResponseWriter, r *http.Request) {
	func() {
		if txn, ok := w.(newrelic.Transaction); ok {
			defer txn.EndSegment(txn.StartSegment(), "f1")
		}
		func() {
			if txn, ok := w.(newrelic.Transaction); ok {
				defer txn.EndSegment(txn.StartSegment(), "f2")
			}
			io.WriteString(w, "segments!")
			time.Sleep(10 * time.Millisecond)
		}()
		time.Sleep(15 * time.Millisecond)
	}()
	time.Sleep(20 * time.Millisecond)
}

func mysql(w http.ResponseWriter, r *http.Request) {
	if txn, ok := w.(newrelic.Transaction); ok {
		defer txn.EndDatastore(txn.StartSegment(), datastore.Segment{
			Product:    datastore.MySQL,
			Collection: "my_table",
			Operation:  "SELECT",
		})
	}

	time.Sleep(20 * time.Millisecond)
	io.WriteString(w, `performing fake query "SELECT * from my_table"`)
}

func external(w http.ResponseWriter, r *http.Request) {
	url := "http://example.com/"
	if txn, ok := w.(newrelic.Transaction); ok {
		defer txn.EndExternal(txn.StartSegment(), url)
	}

	resp, err := http.Get(url)
	if nil != err {
		io.WriteString(w, err.Error())
		return
	}
	defer resp.Body.Close()
	io.Copy(w, resp.Body)
}

func roundtripper(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{}
	if txn, ok := w.(newrelic.Transaction); ok {
		client.Transport = newrelic.NewRoundTripper(txn, nil)
	}
	resp, err := client.Get("http://example.com/")
	if nil != err {
		io.WriteString(w, err.Error())
		return
	}
	defer resp.Body.Close()
	io.Copy(w, resp.Body)
}

const (
	licenseVar = "NEW_RELIC_LICENSE_KEY"
	appname    = "My Go Application"
)

func main() {
	lic := os.Getenv(licenseVar)
	if "" == lic {
		fmt.Printf("environment variable %s unset\n", licenseVar)
		os.Exit(1)
	}

	cfg := newrelic.NewConfig(appname, lic)

	var err error
	app, err = newrelic.NewApplication(cfg)
	if nil != err {
		fmt.Println(err)
		os.Exit(1)
	}

	http.HandleFunc(newrelic.WrapHandleFunc(app, "/", index))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/notice_error", noticeError))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/custom_event", customEvent))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/set_name", setName))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/add_attribute", addAttribute))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/ignore", ignore))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/segments", segments))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/mysql", mysql))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/external", external))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/roundtripper", roundtripper))
	http.HandleFunc("/background", background)

	http.ListenAndServe(":8000", nil)
}
