# New Relic Go Agent [![GoDoc](https://godoc.org/github.com/newrelic/go-agent?status.svg)](https://godoc.org/github.com/newrelic/go-agent)

## Description

The New Relic Go Agent allows you to monitor your Go applications with New
Relic.  It helps you track transactions, outbound requests, database calls, and
other parts of your Go application's behavior and provides a running overview of
garbage collection, goroutine activity, and memory use.

All pull requests will be reviewed by the New Relic product team. Any questions or issues should be directed to our [support
site](http://support.newrelic.com/) or our [community
forum](https://discuss.newrelic.com).

## Requirements

Go 1.3+ is required, due to the use of http.Client's Timeout field.

Linux, OS X, and Windows (Vista, Server 2008 and later) are supported.

## Integrations

The following [_integration packages](https://godoc.org/github.com/newrelic/go-agent/_integrations)
extend the base [newrelic](https://godoc.org/github.com/newrelic/go-agent) package
to support the following frameworks and libraries.
Frameworks and databases which don't have an integration package may still be
instrumented using the [newrelic](https://godoc.org/github.com/newrelic/go-agent)
package primitives.  Specifically, more information about instrumenting your database using
these primitives can be found
[here](https://github.com/newrelic/go-agent/blob/master/GUIDE.md#datastore-segments).

| Project | Integration Package |  |
| ------------- | ------------- | - |
| [aws/aws-sdk-go](https://github.com/aws/aws-sdk-go) | [_integrations/nrawssdk/v1](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrawssdk/v1) | Instrument outbound calls made using Go AWS SDK |
| [aws/aws-sdk-go-v2](https://github.com/aws/aws-sdk-go-v2) | [_integrations/nrawssdk/v2](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrawssdk/v2) | Instrument outbound calls made using Go AWS SDK v2 |
| [labstack/echo](https://github.com/labstack/echo) | [_integrations/nrecho](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrecho) | Instrument inbound requests through the Echo framework |
| [gin-gonic/gin](https://github.com/gin-gonic/gin) | [_integrations/nrgin/v1](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrgin/v1) | Instrument inbound requests through the Gin framework |
| [gorilla/mux](https://github.com/gorilla/mux) | [_integrations/nrgorilla/v1](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrgorilla/v1) | Instrument inbound requests through the Gorilla framework |
| [aws/aws-lambda-go](https://github.com/aws/aws-lambda-go) | [_integrations/nrlambda](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrlambda) | Instrument AWS Lambda applications |
| [sirupsen/logrus](https://github.com/sirupsen/logrus) | [_integrations/nrlogrus](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrlogrus) | Send agent log messages to Logrus |
| [mgutz/logxi](https://github.com/mgutz/logxi) | [_integrations/nrlogxi/v1](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrlogxi/v1) | Send agent log messages to Logxi |
| [pkg/errors](https://github.com/pkg/errors) | [_integrations/nrpkgerrors](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrpkgerrors) | Wrap pkg/errors errors to improve stack traces and error class information |

These integration packages must be imported along
with the [newrelic](https://godoc.org/github.com/newrelic/go-agent) package, as shown in this
[nrgin example](https://github.com/newrelic/go-agent/blob/master/_integrations/nrgin/v1/example/main.go).

## Getting Started

Here are the basic steps to instrumenting your application.  For more
information, see [GUIDE.md](GUIDE.md).

#### Step 0: Installation

Installing the Go Agent is the same as installing any other Go library.  The
simplest way is to run:

```
go get github.com/newrelic/go-agent
```

Then import the `github.com/newrelic/go-agent` package in your application.

#### Step 1: Create a Config and an Application

In your `main` function or an `init` block:

```go
config := newrelic.NewConfig("Your Application Name", "__YOUR_NEW_RELIC_LICENSE_KEY__")
app, err := newrelic.NewApplication(config)
```

[more info](GUIDE.md#config-and-application), [application.go](application.go),
[config.go](config.go)

#### Step 2: Add Transactions

Transactions time requests and background tasks.  Use `WrapHandle` and
`WrapHandleFunc` to create transactions for requests handled by the `http`
standard library package.

```go
http.HandleFunc(newrelic.WrapHandleFunc(app, "/users", usersHandler))
```

Alternatively, create transactions directly using the application's
`StartTransaction` method:

```go
txn := app.StartTransaction("myTxn", optionalResponseWriter, optionalRequest)
defer txn.End()
```

[more info](GUIDE.md#transactions), [transaction.go](transaction.go)

#### Step 3: Instrument Segments

Segments show you where time in your transactions is being spent.  At the
beginning of important functions, add:

```go
defer newrelic.StartSegment(txn, "mySegmentName").End()
```

[more info](GUIDE.md#segments), [segments.go](segments.go)

## Runnable Example

[examples/server/main.go](./examples/server/main.go) is an example that will
appear as "Example App" in your New Relic applications list.  To run it:

```
env NEW_RELIC_LICENSE_KEY=__YOUR_NEW_RELIC_LICENSE_KEY__LICENSE__ \
    go run examples/server/main.go
```

Some endpoints exposed are [http://localhost:8000/](http://localhost:8000/)
and [http://localhost:8000/notice_error](http://localhost:8000/notice_error)


## Basic Example

Before Instrumentation

```go
package main

import (
	"io"
	"net/http"
)

func helloHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "hello, world")
}

func main() {
	http.HandleFunc("/", helloHandler)
	http.ListenAndServe(":8000", nil)
}
```

After Instrumentation

```go
package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/newrelic/go-agent"
)

func helloHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "hello, world")
}

func main() {
	// Create a config.  You need to provide the desired application name
	// and your New Relic license key.
	cfg := newrelic.NewConfig("Example App", "__YOUR_NEW_RELIC_LICENSE_KEY__")

	// Create an application.  This represents an application in the New
	// Relic UI.
	app, err := newrelic.NewApplication(cfg)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Wrap helloHandler.  The performance of this handler will be recorded.
	http.HandleFunc(newrelic.WrapHandleFunc(app, "/", helloHandler))
	http.ListenAndServe(":8000", nil)
}
```

## Support

You can find more detailed documentation [in the guide](GUIDE.md) and on
[the New Relic Documentation site](https://docs.newrelic.com/docs/agents/go-agent).

If you can't find what you're looking for there, reach out to us on our [support
site](http://support.newrelic.com/) or our [community
forum](https://discuss.newrelic.com) and we'll be happy to help you.

Find a bug?  Contact us via [support.newrelic.com](http://support.newrelic.com/),
or email support@newrelic.com.
