# New Relic Go Agent

## Requirements

Go 1.3+ is required, due to the use of http.Client's Timeout field.

## Getting Started

There are three types exposed by this agent.

| Entity  | See | Created By |
| ------------- | ------------- | ------------- |
| `Config`       | [config.go](api/config.go)  | `NewConfig`  |
| `Application`  | [application.go](api/application.go)  | `NewApplication`  |
| `Transaction`  | [transaction.go](api/transaction.go)  | `application.StartTransaction` or implicitly by `WrapHandle` and `WrapHandleFunc`  |

The public interface is contained in the top-level `newrelic` package and
the `newrelic/api` package.

## Example

Here is a barebones web server before and after New Relic Go Agent instrumentation.

### Before Instrumentation

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

### After Instrumentation

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

## Let's Go!

An example web server lives in: [example/main.go](./example/main.go).  To run it:

```
env NEW_RELIC_LICENSE_KEY=__YOUR_LICENSE_HERE__ go run example/main.go
```

Some endpoints exposed are:
* [http://localhost:8000/](http://localhost:8000/)
* [http://localhost:8000/notice_error](http://localhost:8000/notice_error)

This example will appear as "My Go Application" in your New Relic applications list.
