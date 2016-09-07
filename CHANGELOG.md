## ChangeLog

## 1.1.0

* Added support for Transaction Traces.

* Stack trace filenames have been shortened: Any thing preceding the first
  `/src/` is now removed.

## 1.0.0

* Removed `BetaToken` from the `Config` structure.

* Breaking Datastore Change:  `datastore` package contents moved to top level
  `newrelic` package.  `datastore.MySQL` has become `newrelic.DatastoreMySQL`.

* Breaking Attributes Change:  `attributes` package contents moved to top
  level `newrelic` package.  `attributes.ResponseCode` has become
  `newrelic.AttributeResponseCode`.  Some attribute name constants have been
  shortened.

* Added "runtime.NumCPU" to the environment tab.  Thanks sergeylanzman for the
  contribution.

* Prefixed the environment tab values "Compiler", "GOARCH", "GOOS", and
  "Version" with "runtime.".

## 0.8.0

* Breaking Segments API Changes:  The segments API has been rewritten with the
  goal of being easier to use and to avoid nil Transaction checks.  See:

  * [segments.go](segments.go)
  * [example/main.go](example/main.go)
  * [GUIDE.md#segments](GUIDE.md#segments)

* Updated LICENSE.txt with contribution information.

## 0.7.1

* Fixed a bug causing the `Config` to fail to serialize into JSON when the
  `Transport` field was populated.

## 0.7.0

* Eliminated `api`, `version`, and `log` packages.  `Version`, `Config`,
  `Application`, and `Transaction` now live in the top level `newrelic` package.
  If you imported the  `attributes` or `datastore` packages then you will need
  to remove `api` from the import path.

* Breaking Logging Changes

Logging is no longer controlled though a single global.  Instead, logging is
configured on a per-application basis with the new `Config.Logger` field.  The
logger is an interface described in [log.go](log.go).  See
[GUIDE.md#logging](GUIDE.md#logging).

## 0.6.1

* No longer create "GC/System/Pauses" metric if no GC pauses happened.

## 0.6.0

* Introduced beta token to support our beta program.

* Rename `Config.Development` to `Config.Enabled` (and change boolean
  direction).

* Fixed a bug where exclusive time could be incorrect if segments were not
  ended.

* Fix unit tests broken in 1.6.

* In `Config.Enabled = false` mode, the license must be the proper length or empty.

* Added runtime statistics for CPU/memory usage, garbage collection, and number
  of goroutines.

## 0.5.0

* Added segment timing methods to `Transaction`.  These methods must only be
  used in a single goroutine.

* The license length check will not be performed in `Development` mode.

* Rename `SetLogFile` to `SetFile` to reduce redundancy.

* Added `DebugEnabled` logging guard to reduce overhead.

* `Transaction` now implements an `Ignore` method which will prevent
  any of the transaction's data from being recorded.

* `Transaction` now implements a subset of the interfaces
  `http.CloseNotifier`, `http.Flusher`, `http.Hijacker`, and `io.ReaderFrom`
  to match the behavior of its wrapped `http.ResponseWriter`.

* Changed project name from `go-sdk` to `go-agent`.

## 0.4.0

* Queue time support added: if the inbound request contains an
`"X-Request-Start"` or `"X-Queue-Start"` header with a unix timestamp, the
agent will report queue time metrics.  Queue time will appear on the
application overview chart.  The timestamp may fractional seconds,
milliseconds, or microseconds: the agent will deduce the correct units.
