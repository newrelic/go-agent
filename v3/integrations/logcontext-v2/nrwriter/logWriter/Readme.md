# New Relic Log Writer

The logWriter library is an `io.Writer` that automatically integrates the latest New Relic Logs in Context features into the go standard library logger. When used as the `io.Writer` for log, this tool will collect log metrics, forward logs, and enrich logs depending on how your New Relic application is configured. This is the most complete and convenient way to to capture log data with New Relic in log.

## Usage

Once your New Relic application has been created, create a logWriter instance. It must be passed an io.Writer, which is where the final log content will be written to, and a pointer to New Relic application.

```go
writer := logWriter.New(os.Stdout, app)
```

If any errors occor while trying to decorate your log with New Relic metadata, it will fail silently and print your log message in its original, unedited form. If you want to see the error messages, then enable debug logging. This will print an error message in a new line after the original log message is printed.

```go
writer.DebugLogging(true)
```

To capture log data in the context of a transaction, make a new logWriter with the `WithTransaction` or `WithContext` methods.

If you have a pointer to a transaction, use the `WithTransaction()` function. 

```go
txn := app.StartTransaction("my transaction")
defer txn.End()
txnWriter := writer.WithTransaction(txn)
```

If you have a context with a transaction pointer in it, use the `WithContext()` function. 

```go
func ExampleHandler(w http.ResponseWriter, r *http.Request) {
    txnWriter := writer.WithContext(r.Context())
}
```
