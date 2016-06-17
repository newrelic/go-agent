package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/log"

	// "github.com/Sirupsen/logrus"
	// _ "github.com/newrelic/go-agent/log/_nrlogrus"
)

var (
	app newrelic.Application
)

func init() {
	log.SetLogFile("stdout", log.LevelDebug)
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

	io.WriteString(w, "background txn")
	time.Sleep(150 * time.Millisecond)
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
	http.HandleFunc("/background", background)

	http.ListenAndServe(":8000", nil)
}
