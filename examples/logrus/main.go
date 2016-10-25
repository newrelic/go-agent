package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/Sirupsen/logrus"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/_integrations/nrlogrus"
)

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

	logrus.SetLevel(logrus.DebugLevel)
	cfg.Logger = nrlogrus.StandardLogger()

	app, err := newrelic.NewApplication(cfg)
	if nil != err {
		fmt.Println(err)
		os.Exit(1)
	}

	http.HandleFunc(newrelic.WrapHandleFunc(app, "/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello world")
	}))

	http.ListenAndServe(":8000", nil)
}
