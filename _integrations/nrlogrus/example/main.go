package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/_integrations/nrlogrus"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("Logrus App"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		nrlogrus.ConfigStandardLogger(),
	)

	if nil != err {
		fmt.Println(err)
		os.Exit(1)
	}

	http.HandleFunc(newrelic.WrapHandleFunc(app, "/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello world")
	}))

	http.ListenAndServe(":8000", nil)
}
