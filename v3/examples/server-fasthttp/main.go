package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	newrelic "github.com/newrelic/go-agent/v3/newrelic"

	"github.com/valyala/fasthttp"
)

func index(ctx *fasthttp.RequestCtx) {
	ctx.WriteString("Hello World")
}

func noticeError(ctx *fasthttp.RequestCtx) {
	ctx.WriteString("noticing an error")
	txn := ctx.UserValue("transaction").(*newrelic.Transaction)
	txn.NoticeError(errors.New("my error message"))
}

func main() {
	// Initialize New Relic
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("FastHTTP App"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDebugLogger(os.Stdout),
		newrelic.ConfigDistributedTracerEnabled(true),
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	if err := app.WaitForConnection(5 * time.Second); nil != err {
		fmt.Println(err)
	}
	_, helloRoute := newrelic.WrapHandleFuncFastHTTP(app, "/hello", index)
	_, errorRoute := newrelic.WrapHandleFuncFastHTTP(app, "/error", noticeError)
	handler := func(ctx *fasthttp.RequestCtx) {
		path := string(ctx.Path())
		method := string(ctx.Method())

		switch {
		case method == "GET" && path == "/hello":
			helloRoute(ctx)
		case method == "GET" && path == "/error":
			errorRoute(ctx)
		}
	}

	// Start the server with the instrumented handler
	fasthttp.ListenAndServe(":8080", handler)
}
