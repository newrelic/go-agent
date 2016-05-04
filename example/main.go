package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	newrelic "github.com/newrelic/go-sdk"
	"github.com/newrelic/go-sdk/log"

	// "github.com/Sirupsen/logrus"
	// _ "github.com/newrelic/go-sdk/log/_nrlogrus"
)

var (
	app newrelic.Application
)

func init() {
	log.SetLogFile("stdout", log.LevelDebug)
	// logrus.SetOutput(os.Stdout)
	// logrus.SetLevel(logrus.DebugLevel)
}

func myHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "hello world")
	time.Sleep(50 * time.Millisecond)
}

type myError struct{}

func (m myError) Error() string { return "my error message" }

func noticeError(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "noticing an error")

	if txn, ok := w.(newrelic.Transaction); ok {
		txn.NoticeError(myError{})
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

const (
	appname = "My Golang Application"
	// licenseVar must be set to your New Relic license to run this example.
	licenseVar   = "NRLICENSE"
	collectorVar = "NRCOLLECTOR"
)

func background(w http.ResponseWriter, r *http.Request) {
	// Transactions started without an http.Request are classified as
	// background (Non-Web) transactions.
	txn := app.StartTransaction("background", nil, nil)
	defer txn.End()

	io.WriteString(w, "background txn")
	time.Sleep(150 * time.Millisecond)
}

func main() {
	lic := os.Getenv(licenseVar)
	if "" == lic {
		fmt.Printf("environment variable %s unset\n", licenseVar)
		os.Exit(1)
	}

	cfg := newrelic.NewConfig(appname, lic)

	if ctr := os.Getenv(collectorVar); "" != ctr {
		cfg.Collector = ctr
	}

	var err error
	app, err = newrelic.NewApplication(cfg)
	if nil != err {
		fmt.Println(err)
		os.Exit(1)
	}

	http.HandleFunc(newrelic.WrapHandleFunc(app, "/", myHandler))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/notice_error", noticeError))
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/custom_event", customEvent))
	http.HandleFunc("/background", background)

	http.ListenAndServe(":8000", nil)
}
