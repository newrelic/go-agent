# New Relic Go Agent

## Description

The New Relic Go Agent allows you to monitor your Go applications with New
Relic.  It helps you track transactions, outbound requests, database calls, and
other parts of your Go application's behavior while automatically providing a
running overview of garbage collection events, goroutine activity, and memory
use.

## Requirements

Go 1.3+ is required, due to the use of http.Client's Timeout field.

Linux and OS X are supported.

## Getting Started

Here are the basic steps to instrumenting your application.  For more
information, see [GUIDE.md](GUIDE.md).

#### Step 1: Create a Config and an Application

In your `main` function or an `init` block:

```go
config := newrelic.NewConfig("Your Application Name", "__YOUR_NEW_RELIC_LICENSE_KEY__")
app, err := newrelic.NewApplication(config)
```

[more info](GUIDE.md#config-and-application), [application.go](api/application.go),
[config.go](api/config.go)

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

[more info](GUIDE.md#transactions), [transaction.go](api/transaction.go)

#### Step 3: Instrument Segments

Segments show you where time in your transactions is being spent.  At the
beginning of important functions, add:

```go
defer txn.EndSegment(txn.StartSegment(), "mySegmentName")
```

[more info](GUIDE.md#segments), [segments.go](api/segments.go)

## Runnable Example

[example/main.go](./example/main.go) is an example that will appear as "My Go
Application" in your New Relic applications list.  To run it:

```
env NEW_RELIC_LICENSE_KEY=__YOUR_LICENSE_HERE__ go run example/main.go
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
	cfg := newrelic.NewConfig("My Go Application", "__YOUR_NEW_RELIC_LICENSE_KEY__")

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
