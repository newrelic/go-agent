# New Relic Go SDK

## Requirements

Go 1.3+ is required, due to the use of http.Client's Timeout field.

## Use

First, create a `Config`:

```
cfg := newrelic.NewConfig("Your Application Name", "YOUR_LICENSE_KEY")
```

Config has many public fields which can be changed to modify behavior.  Take a
look in [config.go](api/config.go).  Then, create an `Application`:

```
app, err := newrelic.NewApplication(cfg)
```

`Application` is an interface described in [application.go](api/application.go).
If the `Config` is invalid, `NewApplication` will return `nil` and an error.
Using the `Application`, you can add custom events:

```
app.RecordCustomEvent("my_event_type", map[string]interface{}{
	"myString": "hello",
	"myFloat":  0.603,
	"myInt":    123,
	"myBool":   true,
})
```

and record transactions:

```
txn := app.StartTransaction("my_transaction", nil, nil)
defer txn.End()
```

`Transaction` is an interface described in [transaction.go](api/transaction.go).
Since instrumentation of standard library http handlers is common, two helper
functions, `WrapHandle` and `WrapHandleFunc` are located in
[instrumentation.go](instrumentation.go).

## Example

An example web server lives in: [example/main.go](./example/main.go)

To run it:

```
NRLICENSE=YOUR_LICENSE_HERE go run example/main.go
```

Then access:
* [http://localhost:8000/](http://localhost:8000/)
* [http://localhost:8000/notice_error](http://localhost:8000/notice_error)
* [http://localhost:8000/custom_event](http://localhost:8000/custom_event)
