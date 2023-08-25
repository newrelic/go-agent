package main

import (
	"fmt"
	"os"
	"time"

	"github.com/newrelic/go-agent/v3/integrations/nrfasthttp"

	newrelic "github.com/newrelic/go-agent/v3/newrelic"

	"github.com/valyala/fasthttp"
)

func main() {
	// Initialize New Relic
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("httprouter App"),
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

	// Define your handler
	handler := nrfasthttp.NRHandler(app, func(ctx *fasthttp.RequestCtx) {
		txn := nrfasthttp.GetTransaction(ctx)
		client := &fasthttp.Client{}

		req := fasthttp.AcquireRequest()
		defer fasthttp.ReleaseRequest(req)
		req.SetRequestURI("http://example.com")

		res := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseResponse(res)

		// Call nrfasthttp.Do instead of fasthttp.Do
		err := nrfasthttp.Do(client, txn, req, res)

		if err != nil {
			fmt.Println("Request failed: ", err)
			return
		}
		// Your handler logic here...
		ctx.WriteString("Hello World")
	})

	// Start the server with the instrumented handler
	fasthttp.ListenAndServe(":8080", handler)
}
