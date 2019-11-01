package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/julienschmidt/httprouter"
	newrelic "github.com/newrelic/go-agent"
	"github.com/newrelic/go-agent/_integrations/nrhttprouter"
)

func index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Write([]byte("welcome\n"))
}

func hello(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Write([]byte(fmt.Sprintf("hello %s\n", ps.ByName("name"))))
}

func main() {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("httprouter App"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDebugLogger(os.Stdout),
	)
	if nil != err {
		fmt.Println(err)
		os.Exit(1)
	}

	// Use an *nrhttprouter.Router in place of an *httprouter.Router.
	router := nrhttprouter.New(app)

	router.GET("/", index)
	router.GET("/hello/:name", hello)

	http.ListenAndServe(":8000", router)
}
