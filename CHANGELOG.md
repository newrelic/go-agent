# ChangeLog

## 3.9.0

### Changes
* When sending Serverless telemetry using the `nrlambda` integration, support an externally-managed named pipe.

## 3.8.1

### Bug Fixes

* Fixed an issue that could cause orphaned Distributed Trace spans when using
  SQL instrumentation like `nrmysql`.

## 3.8.0

### Changes
* When marking a transaction as a web transaction using 
[Transaction.SetWebRequest](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Transaction.SetWebRequest), 
it is now possible to include a `Host` field in the 
[WebRequest](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#WebRequest)
 struct, which defaults to the empty string.

### Bug Fixes

* The `Host` header is now being correctly captured and recorded in the 
 `request.headers.host` attribute, as described 
 [here](https://docs.newrelic.com/docs/agents/go-agent/instrumentation/go-agent-attributes#requestHeadersHost).
*  Previously, the timestamps on Spans and Transactions were being written
   using different data types, which sometimes caused rounding errors that
   could cause spans to be offset incorrectly in the UI. This has been fixed.

## 3.7.0

### Changes

* When `Config.Transport` is nil, no longer use the `http.DefaultTransport`
  when communicating with the New Relic backend.  This addresses an issue with
  shared transports as described in https://github.com/golang/go/issues/33006.

* If a timeout occurs when attempting to send data to the New Relic backend,
  instead of dropping the data, we save it and attempt to send it with the
  next harvest.  Note data retention limits still apply and the agent will
  still start to drop data when these limits are reached. We attempt to keep
  the highest priority events and traces.

## 3.6.0

### New Features

* Added support for [adding custom attributes directly to
  spans](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Segment.AddAttribute).
  These attributes will be visible when looking at spans in the Distributed
  Tracing UI.

  Example:
  ```go
  txn := newrelic.FromContext(r.Context())
  sgmt := txn.StartSegment("segment1")
  defer sgmt.End()
  sgmt.AddAttribute("mySpanString", "hello")
  sgmt.AddAttribute("mySpanInt", 123)
  ```

* Custom attributes added to the transaction with `txn.AddAttribute` are now
  also added to the root Span Event and will be visible when looking at the
  span in the Distributed Tracing UI. These custom attributes can be disabled
  from all destinations using `Config.Attributes.Exclude` or disabled from Span
  Events specifically using `Config.SpanEvents.Attributes.Exclude`.

* Agent attributes added to the transaction are now also added to the root Span
  Event and will be visible when looking at the span in the Distributed Tracing
  UI. These attributes include the `request.uri` and the `request.method` along
  with all other attributes listed in the [attributes section of our
  godocs](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#pkg-constants).
  These agent attributes can be disabled from all destinations using
  `Config.Attributes.Exclude` or disabled from Span Events specifically using
  `Config.SpanEvents.Attributes.Exclude`.

### Bug Fixes

* Fixed an issue where it was impossible to exclude the attributes
  `error.class` and `error.message` from the root Span Event. This issue has
  now been fixed. These attributes can now be excluded from all Span Events
  using `Config.Attributes.Exclude` or `Config.SpanEvents.Attributes.Exclude`.
  
* Fixed an issue that caused Go's data race warnings to trigger in certain situations 
  when using the `newrelic.NewRoundTripper`. There were no reports of actual data corruption, 
  but now the warnings should be resolved. Thank you to @blixt for bringing this to our 
  attention!

## 3.5.0

### New Features

* Added support for [Infinite Tracing on New Relic
  Edge](https://docs.newrelic.com/docs/understand-dependencies/distributed-tracing/enable-configure/enable-distributed-tracing).

  Infinite Tracing observes 100% of your distributed traces and provides
  visualizations for the most actionable data so you have the examples of errors
  and long-running traces so you can better diagnose and troubleshoot your systems.

  You [configure your
  agent](https://docs.newrelic.com/docs/agents/go-agent/configuration/go-agent-configuration#infinite-tracing)
  to send traces to a trace observer in New Relic Edge.  You view your
  distributed traces through the New Relic’s UI. There is no need to install a
  collector on your network.

  Infinite Tracing is currently available on a sign-up basis. If you would like to
  participate, please contact your sales representative.
  
  **As part of this change, the Go Agent now has an added dependency on gRPC.** 
  This is true whether or not you enable the Infinite Tracing feature. The gRPC dependencies include these two libraries:
  * [github.com/golang/protobuf](https://github.com/golang/protobuf) v1.3.3
  * [google.golang.org/grpc](https://github.com/grpc/grpc-go) v1.27.0

  You can see the changes in the [go.mod file](v3/go.mod) 

  **As part of this change, the Go Agent now has an added dependency on gRPC.** 
  This is true whether or not you enable the Infinite Tracing feature. The gRPC dependencies include these two libraries:
  * [github.com/golang/protobuf](https://github.com/golang/protobuf) v1.3.3
  * [google.golang.org/grpc](https://github.com/grpc/grpc-go) v1.27.0

  You can see the changes in the [go.mod file](v3/go.mod) 

### Changes

* [`nrgin.Middleware`](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrgin#Middleware)
  uses
  [`Context.FullPath()`](https://godoc.org/github.com/gin-gonic/gin#Context.FullPath)
  for transaction names when using Gin version 1.5.0 or greater.  Gin
  transactions were formerly named after the
  [`Context.HandlerName()`](https://godoc.org/github.com/gin-gonic/gin#Context.HandlerName),
  which uses reflection.  This change improves transaction naming and reduces
  overhead.  Please note that because your transaction names will change, you
  may have to update any related dashboards and alerts to match the new name.
  If you wish to continue using `Context.HandlerName()` for your transaction
  names, use
  [`nrgin.MiddlewareHandlerTxnNames`](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrgin#MiddlewareHandlerTxnNames)
  instead.

  ```go
  // Transactions previously named
  "GET main.handleGetUsers"
  // will be change to something like this match the full path
  "GET /user/:id"
  ```

  Note: As part of agent release v3.4.0, a v2.0.0 tag was added to the nrgin
  package.  When using go modules however, it was impossible to install this
  latest version of nrgin.  The v2.0.0 tag has been removed and replaced with
  v1.1.0.

## 3.4.0

### New Features

* Attribute `http.statusCode` has been added to external span events
  representing the status code on an http response.  This attribute will be
  included when added to an ExternalSegment in one of these three ways:

  1. Using
     [`NewRoundTripper`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#NewRoundTripper)
     with your http.Client
  2. Including the http.Response as a field on your
     [`ExternalSegment`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#ExternalSegment)
  3. Using the new
     [`ExternalSegment.SetStatusCode`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#ExternalSegment.SetStatusCode)
     API to set the status code directly

  To exclude the `http.statusCode` attribute from span events, update your
  agent configuration like so, where `cfg` is your [`newrelic.Config`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Config) object.

  ```go
  cfg.SpanEvents.Attributes.Exclude = append(cfg.SpanEvents.Attributes.Exclude, newrelic.SpanAttributeHTTPStatusCode)
  ```

* Error attributes `error.class` and `error.message` are now included on the
 span event in which the error was noticed, or on the root span if an error
 occurs in a transaction with no segments (no chid spans). Only the most recent error
 information is added to the attributes; prior errors on the same span are
 overwritten.

  To exclude the `error.class` and/or `error.message` attributes from span events, update your
  agent configuration like so, where `cfg` is your [`newrelic.Config`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Config) object.

  ```go
  cfg.SpanEvents.Attributes.Exclude = append(cfg.SpanEvents.Attributes.Exclude, newrelic.newrelic.SpanAttributeErrorClass, newrelic.SpanAttributeErrorMessage)
  ```

### Changes

* Use
  [`Context.FullPath()`](https://godoc.org/github.com/gin-gonic/gin#Context.FullPath)
  for transaction names when using Gin version 1.5.0 or greater.  Gin
  transactions were formerly named after the
  [`Context.HandlerName()`](https://godoc.org/github.com/gin-gonic/gin#Context.HandlerName),
  which uses reflection.  This change improves transaction naming and reduces
  overhead.  Please note that because your transaction names will change, you
  may have to update any related dashboards and alerts to match the new name.

  ```go
  // Transactions previously named
  "GET main.handleGetUsers"
  // will be change to something like this match the full path
  "GET /user/:id"
  ```
* If you are using any of these integrations, you must upgrade them when you
 upgrade the agent:
    * [nrlambda v1.1.0](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrlambda)
    * [nrmicro v1.1.0](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrmicro)
    * [nrnats v1.1.0](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrnats)
    * [nrstan v1.1.0](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrstan)
    
### Known Issues and Workarounds

* If a .NET agent is initiating distributed traces as the root service, you must 
  update that .NET agent to version 8.24 or later before upgrading your downstream 
  Go New Relic agents to this agent release.

## 3.3.0

### New Features

* Added support for GraphQL in two new integrations:
  * [graph-gophers/graphql-go](https://github.com/graph-gophers/graphql-go)
  with
  [v3/integrations/nrgraphgophers](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrgraphgophers).
    * [Documentation](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrgraphgophers)
    * [Example](v3/integrations/nrgraphgophers/example/main.go)
  * [graphql-go/graphql](https://github.com/graphql-go/graphql)
  with
  [v3/integrations/nrgraphqlgo](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrgraphqlgo).
    * [Documentation](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrgraphqlgo)
    * [Example](v3/integrations/nrgraphqlgo/example/main.go)

* Added database instrumentation support for
  [snowflakedb/gosnowflake](https://github.com/snowflakedb/gosnowflake).
  * [Documentation](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrsnowflake)
  * [Example](v3/integrations/nrsnowflake/example/main.go)

### Changes

* When using
  [`newrelic.StartExternalSegment`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#StartExternalSegment)
  or
  [`newrelic.NewRoundTripper`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#NewRoundTripper),
  if existing cross application tracing or distributed tracing headers are
  present on the request, they will be replaced instead of added.

* The
  [`FromContext`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#FromContext)
  API which allows you to pull a Transaction from a context.Context will no
  longer panic if the provided context is nil.  In this case, a nil is
  returned.
  
### Known Issues and Workarounds

* If a .NET agent is initiating distributed traces as the root service, you must 
  update that .NET agent to version 8.24 or later before upgrading your downstream 
  Go New Relic agents to this agent release.

## 3.2.0

### New Features

* Added support for `v7` of [go-redis/redis](https://github.com/go-redis/redis)
  in the new [v3/integrations/nrredis-v7](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrredis-v7)
  package.
  * [Documentation](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrredis-v7)
  * [Example](v3/integrations/nrredis-v7/example/main.go)

### Changes

* Updated Gorilla instrumentation to include request time spent in middlewares.
  Added new `nrgorilla.Middleware` and deprecated `nrgorilla.InstrumentRoutes`.
  Register the new middleware as your first middleware using
  [`Router.Use`](https://godoc.org/github.com/gorilla/mux#Router.Use). See the
  [godocs
  examples](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrgorilla)
  for more details.

  ```go
  r := mux.NewRouter()
  // Always register the nrgorilla.Middleware first.
  r.Use(nrgorilla.Middleware(app))

  // All handlers and custom middlewares will be instrumented.  The
  // transaction will be available in the Request's context.
  r.Use(MyCustomMiddleware)
  r.Handle("/", makeHandler("index"))

  // The NotFoundHandler and MethodNotAllowedHandler must be instrumented
  // separately using newrelic.WrapHandle.  The second argument to
  // newrelic.WrapHandle is used as the transaction name; the string returned
  // from newrelic.WrapHandle should be ignored.
  _, r.NotFoundHandler = newrelic.WrapHandle(app, "NotFoundHandler", makeHandler("not found"))
  _, r.MethodNotAllowedHandler = newrelic.WrapHandle(app, "MethodNotAllowedHandler", makeHandler("method not allowed"))

  http.ListenAndServe(":8000", r)
  ```

### Known Issues and Workarounds

* If a .NET agent is initiating distributed traces as the root service, you must 
  update that .NET agent to version 8.24 or later before upgrading your downstream 
  Go New Relic agents to this agent release.

## 3.1.0

### New Features

* Support for W3C Trace Context, with easy upgrade from New Relic trace context.

  Distributed Tracing now supports W3C Trace Context headers for HTTP and
  gRPC protocols when distributed tracing is enabled.  Our implementation can
  accept and emit both W3C trace header format and New Relic trace header
  format.  This simplifies agent upgrades, allowing trace context to be
  propagated between services with older and newer releases of New Relic
  agents.  W3C trace header format will always be accepted and emitted.  New
  Relic trace header format will be accepted, and you can optionally disable
  emission of the New Relic trace header format.

  When distributed tracing is enabled with
  `Config.DistributedTracer.Enabled = true`, the Go agent will now accept
  W3C's `traceparent` and `tracestate` headers when calling
  [`Transaction.AcceptDistributedTraceHeaders`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Transaction.AcceptDistributedTraceHeaders).  When calling
  [`Transaction.InsertDistributedTraceHeaders`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#Transaction.InsertDistributedTraceHeaders), the Go agent will include the
  W3C headers along with the New Relic distributed tracing header, unless
  the New Relic trace header format is disabled using
  `Config.DistributedTracer.ExcludeNewRelicHeader = true`.

* Added support for [elastic/go-elasticsearch](https://github.com/elastic/go-elasticsearch)
  in the new [v3/integrations/nrelasticsearch-v7](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrelasticsearch-v7)
  package.

* At this time, the New Relic backend has enabled support for real time
  streaming.  Versions 2.8 and above will now send data to New Relic every five
  seconds, instead of every minute.  As a result, transaction, error, and custom
  events will now be available in New Relic One and Insights dashboards in near
  real time.
  
### Known Issues and Workarounds

* If a .NET agent is initiating distributed traces as the root service, you must update 
  that .NET agent to version 8.24 or later before upgrading your downstream 
  Go New Relic agents to this agent release.

## 3.0.0

We are pleased to announce the release of Go Agent v3.0.0!  This is a major release
that includes some breaking changes that will simplify your future use of the Go
Agent.

Please pay close attention to the list of Changes.

### Changes

* A full list of changes and a step by step checklist on how to upgrade can
  be found in the [v3 Migration Guide](MIGRATION.md).

### New Features

* Support for Go Modules.  Our Go agent integration packages support frameworks
  and libraries which are changing over time. With support for Go Modules, we
  are now able to release instrumentation packages for multiple versions of
  frameworks and libraries with a single agent release; and support operation
  of the Go agent in Go Modules environments.   This affects naming of our
  integration packages, as described in the v3 Migration Guide (see under
  "Changes" above).

* Detect and set hostnames based on Heroku dyno names.  When deploying an
  application in Heroku, the hostnames collected will now match the dyno name.
  This serves to greatly improve the usability of the servers list in APM since
  dyno names are often sporadic or fleeting in nature.  The feature is
  controlled by two new configuration options `Config.Heroku.UseDynoNames` and
  `Config.Heroku.DynoNamePrefixesToShorten`.

## 2.16.3

### New Relic's Go agent v3.0 is currently available for review and beta testing.  Your use of this pre-release is at your own risk. New Relic disclaims all warranties, express or implied, regarding the beta release.

### If you do not manually take steps to use the new v3 folder you will not see any changes in your agent.

This is the third release of the pre-release of Go agent v3.0.  It includes
changes due to user feedback during the pre-release. The existing agent in
`"github.com/newrelic/go-agent"` is unchanged.  The Go agent v3.0 code in the v3
folder has the following changes:

* A [ConfigFromEnvironment](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#ConfigFromEnvironment)
  bug has been fixed.

## 2.16.2

### New Relic's Go agent v3.0 is currently available for review and beta testing. Your use of this pre-release is at your own risk. New Relic disclaims all warranties, express or implied, regarding the beta release.

### If you do not manually take steps to use the new v3 folder, as described below, you will not see any changes in your agent.

This is the second release of the pre-release of Go agent v3.0.  It includes changes due to user feedback during the pre-release. The existing
agent in `"github.com/newrelic/go-agent"` is unchanged.  The Go agent v3.0 code
in the v3 folder has the following changes:

* Transaction names created by [`WrapHandle`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#WrapHandle),
[`WrapHandleFunc`](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#WrapHandleFunc),
[nrecho-v3](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrecho-v3),
[nrecho-v4](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrecho-v4),
[nrgorilla](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrgorilla), and
[nrgin](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrgin) now
include the HTTP method.  For example, the following code:

  ```go
  http.HandleFunc(newrelic.WrapHandleFunc(app, "/users", usersHandler))
  ```

  now creates a metric called `WebTransaction/Go/GET /users` instead of
  `WebTransaction/Go/users`.  As a result of this change, you may need to update
  your alerts and dashboards.

* The [ConfigFromEnvironment](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#ConfigFromEnvironment)
  config option is now strict.  If one of the environment variables, such as
  `NEW_RELIC_DISTRIBUTED_TRACING_ENABLED`, cannot be parsed, then `Config.Error`
  will be populated and [NewApplication](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#NewApplication)
  will return an error.

* [ConfigFromEnvironment](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#ConfigFromEnvironment)
  now processes `NEW_RELIC_ATTRIBUTES_EXCLUDE` and `NEW_RELIC_ATTRIBUTES_INCLUDE`.

## 2.16.1

### New Relic's Go agent v3.0 is currently available for review and beta testing. Your use of this pre-release is at your own risk. New Relic disclaims all warranties, express or implied, regarding the beta release.

### If you do not manually take steps to use the new v3 folder, as described below, you will not see any changes in your agent.

This 2.16.1 release includes a new v3.0 folder which contains the pre-release of
Go agent v3.0; Go agent v3.0 includes breaking changes. We are seeking
feedback and hope that you will look this over and test out the changes prior
to the official release.

**This is not an official 3.0 release, it is just a vehicle to gather feedback
on proposed changes**. It is not tagged as 3.0 in Github and the 3.0 release is
not yet available to update in your Go mod file. In order to test out these
changes, you will need to clone this repo in your Go source directory, under
`[go-src-dir]/src/github.com/newrelic/go-agent`. Once you have the source
checked out, you will need to follow the steps in the second section of
[v3/MIGRATION.md](v3/MIGRATION.md).

A list of changes and installation instructions is included in the v3 folder
and can be found [here](v3/MIGRATION.md)

For this pre-release (beta) version of Go agent v3.0, please note:
* The changes in the v3 folder represent what we expect to release in ~2 weeks
as our major 3.0 release. However, as we are soliciting feedback on the changes
and there is the possibility of some breaking changes before the official
release.
* This is not an official 3.0 release; it is not tagged as 3.0 in Github and
the 3.0 release is not yet available to update in your Go mod file.
* If you test out these changes and encounter issues, questions, or have
feedback that you would like to pass along, please open up an issue
[here](https://github.com/newrelic/go-agent/issues/new) and be sure to include
the label `3.0`.
  * For normal (non-3.0) issues/questions we request that you report them via
   our [support site](http://support.newrelic.com/) or our
   [community forum](https://discuss.newrelic.com). Please only report
   questions related to the 3.0 pre-release directly via GitHub.


### New Features

* V3 will add support for Go Modules. The go.mod files exist in the v3 folder,
but they will not be usable until we have fully tagged the 3.0 release
officially. Examples of version tags we plan to use for different modules
include:
  * `v3.0.0`
  * `v3/integrations/nrecho-v3/v1.0.0`
  * `v3/integrations/nrecho-v4/v1.0.0`

### Changes

* The changes are the ones that we have requested feedback previously in
[this issue](https://github.com/newrelic/go-agent/issues/106).  
* A full list of changes that are included, along with a checklist for
 upgrading, is available in [v3/MIGRATION.md](v3/MIGRATION.md).

## 2.16.0

### Upcoming

* The next release of the Go Agent is expected to be a major version release
  to improve the API and incorporate Go modules.
  Details available here: https://github.com/newrelic/go-agent/issues/106
  We would love your feedback!

### Bug Fixes

* Fixed an issue in the
  [`nrhttprouter`](http://godoc.org/github.com/newrelic/go-agent/_integrations/nrhttprouter)
  integration where the transaction was not being added to the requests
  context.  This resulted in an inability to access the transaction from within
  an
  [`httprouter.Handle`](https://godoc.org/github.com/julienschmidt/httprouter#Handle)
  function.  This issue has now been fixed.

## 2.15.0

### New Features

* Added support for monitoring [MongoDB](https://github.com/mongodb/mongo-go-driver/) queries with the new
[_integrations/nrmongo](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrmongo)
package.

  * [Example application](https://github.com/newrelic/go-agent/blob/master/_integrations/nrmongo/example/main.go)
  * [Full godocs Documentation](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrmongo)

* Added new method `Transaction.IsSampled()` that returns a boolean that
  indicates if the transaction is sampled.  A sampled transaction records a
  span event for each segment.  Distributed tracing must be enabled for
  transactions to be sampled.  `false` is returned if the transaction has
  finished.  This sampling flag is needed for B3 trace propagation and
  future support of W3C Trace Context.

* Added support for adding [B3
  Headers](https://github.com/openzipkin/b3-propagation) to outgoing requests.
  This is helpful if the service you are calling uses B3 for trace state
  propagation (for example, it uses Zipkin instrumentation).  You can use the
  new
  [_integrations/nrb3](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrb3)
  package's
  [`nrb3.NewRoundTripper`](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrb3#NewRoundTripper)
  like this:

  ```go
  // When defining the client, set the Transport to the NewRoundTripper. This
  // will create ExternalSegments and add B3 headers for each request.
  client := &http.Client{
      Transport: nrb3.NewRoundTripper(nil),
  }

  // Distributed Tracing must be enabled for this application.
  // (see https://docs.newrelic.com/docs/understand-dependencies/distributed-tracing/enable-configure/enable-distributed-tracing)
  txn := currentTxn()

  req, err := http.NewRequest("GET", "http://example.com", nil)
  if nil != err {
      log.Fatalln(err)
  }

  // Be sure to add the transaction to the request context.  This step is
  // required.
  req = newrelic.RequestWithTransactionContext(req, txn)
  resp, err := client.Do(req)
  if nil != err {
      log.Fatalln(err)
  }

  defer resp.Body.Close()
  fmt.Println(resp.StatusCode)
  ```

### Bug Fixes

* Fixed an issue where the
  [`nrgin`](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrgin/v1)
  integration was not capturing the correct response code in the case where no
  response body was sent.  This issue has now been fixed but requires Gin
  greater than v1.4.0.

## 2.14.1

### Bug Fixes

* Removed the hidden `"NEW_RELIC_DEBUG_LOGGING"` environment variable setting
  which was broken in release 2.14.0.

## 2.14.0

### New Features

* Added support for a new segment type,
  [`MessageProducerSegment`](https://godoc.org/github.com/newrelic/go-agent#MessageProducerSegment),
  to be used to track time spent adding messages to message queuing systems like
  RabbitMQ or Kafka.

  ```go
  seg := &newrelic.MessageProducerSegment{
      StartTime:       newrelic.StartSegmentNow(txn),
      Library:         "RabbitMQ",
      DestinationType: newrelic.MessageExchange,
      DestinationName: "myExchange",
  }
  // add message to queue here
  seg.End()
  ```

* Added new attribute constants for use with message consumer transactions.
  These attributes can be used to add more detail to a transaction that tracks
  time spent consuming a message off a message queuing system like RabbitMQ or Kafka.
  They can be added using
  [`txn.AddAttribute`](https://godoc.org/github.com/newrelic/go-agent#Transaction).

  ```go
  // The routing key of the consumed message.
  txn.AddAttribute(newrelic.AttributeMessageRoutingKey, "myRoutingKey")
  // The name of the queue the message was consumed from.
  txn.AddAttribute(newrelic.AttributeMessageQueueName, "myQueueName")
  // The type of exchange used for the consumed message (direct, fanout,
  // topic, or headers).
  txn.AddAttribute(newrelic.AttributeMessageExchangeType, "myExchangeType")
  // The callback queue used in RPC configurations.
  txn.AddAttribute(newrelic.AttributeMessageReplyTo, "myReplyTo")
  // The application-generated identifier used in RPC configurations.
  txn.AddAttribute(newrelic.AttributeMessageCorrelationID, "myCorrelationID")
  ```

  It is recommended that at most one message is consumed per transaction.

* Added support for [Go 1.13's Error wrapping](https://golang.org/doc/go1.13#error_wrapping).
  `Transaction.NoticeError` now uses [Unwrap](https://golang.org/pkg/errors/#Unwrap)
  recursively to identify the error's cause (the deepest wrapped error) when generating
  the error's class field.  This functionality will help group your errors usefully.

  For example, when using Go 1.13, the following code:

  ```go
  type socketError struct{}

  func (e socketError) Error() string { return "socket error" }

  func gamma() error { return socketError{} }
  func beta() error  { return fmt.Errorf("problem in beta: %w", gamma()) }
  func alpha() error { return fmt.Errorf("problem in alpha: %w", beta()) }

  func execute(txn newrelic.Transaction) {
  	err := alpha()
  	txn.NoticeError(err)
  }
  ```
  captures an error with message `"problem in alpha: problem in beta: socket error"`
  and class `"main.socketError"`.  Previously, the class was recorded as `"*fmt.wrapError"`.

* A `Stack` field has been added to [Error](https://godoc.org/github.com/newrelic/go-agent#Error),
  which can be assigned using the new
  [NewStackTrace](https://godoc.org/github.com/newrelic/go-agent#NewStackTrace) function.
  This allows your error stack trace to show where the error happened, rather
  than the location of the `NoticeError` call.

  `Transaction.NoticeError` not only checks for a stack trace (using
  [StackTracer](https://godoc.org/github.com/newrelic/go-agent#StackTracer)) in
  the error parameter, but in the error's cause as well.  This means that you
  can create an [Error](https://godoc.org/github.com/newrelic/go-agent#Error)
  where your error occurred, wrap it multiple times to add information, notice it
  with `NoticeError`, and still have a useful stack trace. Take a look!

  ```go
  func gamma() error {
  	return newrelic.Error{
  		Message: "something went very wrong",
  		Class:   "socketError",
  		Stack:   newrelic.NewStackTrace(),
  	}
  }

  func beta() error  { return fmt.Errorf("problem in beta: %w", gamma()) }
  func alpha() error { return fmt.Errorf("problem in alpha: %w", beta()) }

  func execute(txn newrelic.Transaction) {
  	err := alpha()
  	txn.NoticeError(err)
  }
  ```

  In this example, the topmost stack trace frame recorded is `"gamma"`,
  rather than `"execute"`.

* Added support for configuring a maximum number of transaction events per minute to be sent to New Relic.
It can be configured as follows:

  ```go
  config := newrelic.NewConfig("Application Name", os.Getenv("NEW_RELIC_LICENSE_KEY"))  
  config.TransactionEvents.MaxSamplesStored = 100
  ```
    * For additional configuration information, see our [documentation](https://docs.newrelic.com/docs/agents/go-agent/configuration/go-agent-configuration)


### Miscellaneous

* Updated the
  [`nrmicro`](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrmicro)
  package to use the new segment type
  [`MessageProducerSegment`](https://godoc.org/github.com/newrelic/go-agent#MessageProducerSegment)
  and the new attribute constants:
  * [`nrmicro.ClientWrapper`](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrmicro#ClientWrapper)
    now uses `newrelic.MessageProducerSegment`s instead of
    `newrelic.ExternalSegment`s for calls to
    [`Client.Publish`](https://godoc.org/github.com/micro/go-micro/client#Client).
  * [`nrmicro.SubscriberWrapper`](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrmicro#SubscriberWrapper)
    updates transaction names and adds the attribute `message.routingKey`.

* Updated the
  [`nrnats`](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrnats)
  and
  [`nrstan`](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrstan)
  packages to use the new segment type
  [`MessageProducerSegment`](https://godoc.org/github.com/newrelic/go-agent#MessageProducerSegment)
  and the new attribute constants:
  * [`nrnats.StartPublishSegment`](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrnats#StartPublishSegment)
    now starts and returns a `newrelic.MessageProducerSegment` type.
  * [`nrnats.SubWrapper`](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrnats#SubWrapper)
    and
    [`nrstan.StreamingSubWrapper`](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrstan#StreamingSubWrapper)
    updates transaction names and adds the attributes `message.routingKey`,
    `message.queueName`, and `message.replyTo`.

## 2.13.0

### New Features

* Added support for [HttpRouter](https://github.com/julienschmidt/httprouter) in
  the new [_integrations/nrhttprouter](http://godoc.org/github.com/newrelic/go-agent/_integrations/nrhttprouter) package.  This package allows you to easily instrument inbound requests through the HttpRouter framework.

  * [Documentation](http://godoc.org/github.com/newrelic/go-agent/_integrations/nrhttprouter)
  * [Example](_integrations/nrhttprouter/example/main.go)

* Added support for [github.com/uber-go/zap](https://github.com/uber-go/zap) in
  the new
  [_integrations/nrzap](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrzap)
  package.  This package allows you to send agent log messages to `zap`.

## 2.12.0

### New Features

* Added new methods to expose `Transaction` details:

  * `Transaction.GetTraceMetadata()` returns a
    [TraceMetadata](https://godoc.org/github.com/newrelic/go-agent#TraceMetadata)
    which contains distributed tracing identifiers.

  * `Transaction.GetLinkingMetadata()` returns a
    [LinkingMetadata](https://godoc.org/github.com/newrelic/go-agent#LinkingMetadata)
    which contains the fields needed to link data to a trace or entity.

* Added a new plugin for the [Logrus logging
  framework](https://github.com/sirupsen/logrus) with the new
  [_integrations/logcontext/nrlogrusplugin](https://github.com/newrelic/go-agent/go-agent/tree/master/_integrations/logcontext/nrlogrusplugin)
  package. This plugin leverages the new `GetTraceMetadata` and
  `GetLinkingMetadata` above to decorate logs.

  To enable, set your log's formatter to the `nrlogrusplugin.ContextFormatter{}`

  ```go
  logger := logrus.New()
  logger.SetFormatter(nrlogrusplugin.ContextFormatter{})
  ```

  The logger will now look for a `newrelic.Transaction` inside its context and
  decorate logs accordingly.  Therefore, the Transaction must be added to the
  context and passed to the logger.  For example, this logging call

  ```go
  logger.Info("Hello New Relic!")
  ```

  must be transformed to include the context, such as:

  ```go
  ctx := newrelic.NewContext(context.Background(), txn)
  logger.WithContext(ctx).Info("Hello New Relic!")
  ```

  For full documentation see the
  [godocs](https://godoc.org/github.com/newrelic/go-agent/_integrations/logcontext/nrlogrusplugin)
  or view the
  [example](https://github.com/newrelic/go-agent/blob/master/_integrations/logcontext/nrlogrusplugin/example/main.go).

* Added support for [NATS](https://github.com/nats-io/nats.go) and [NATS Streaming](https://github.com/nats-io/stan.go)
monitoring with the new [_integrations/nrnats](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrnats) and
[_integrations/nrstan](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrstan) packages.  These packages
support instrumentation of publishers and subscribers.

  * [NATS Example](https://github.com/newrelic/go-agent/blob/master/_integrations/nrnats/examples/main.go)
  * [NATS Streaming Example](https://github.com/newrelic/go-agent/blob/master/_integrations/nrstan/examples/main.go)

* Enables ability to migrate to [Configurable Security Policies (CSP)](https://docs.newrelic.com/docs/agents/manage-apm-agents/configuration/enable-configurable-security-policies) on a per agent basis for accounts already using [High Security Mode (HSM)](https://docs.newrelic.com/docs/agents/manage-apm-agents/configuration/high-security-mode).
  * Previously, if CSP was configured for an account, New Relic would not allow an agent to connect without the `security_policies_token`. This led to agents not being able to connect during the period between when CSP was enabled for an account and when each agent is configured with the correct token.
  * With this change, when both HSM and CSP are enabled for an account, an agent (this version or later) can successfully connect with either `high_security: true` or the appropriate `security_policies_token` configured - allowing the agent to continue to connect after CSP is configured on the account but before the appropriate `security_policies_token` is configured for each agent.

## 2.11.0

### New Features

* Added support for [Micro](https://github.com/micro/go-micro) monitoring with the new
[_integrations/nrmicro](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrmicro)
package.  This package supports instrumentation for servers, clients, publishers, and subscribers.

  * [Server Example](https://github.com/newrelic/go-agent/blob/master/_integrations/nrmicro/example/server/server.go)
  * [Client Example](https://github.com/newrelic/go-agent/blob/master/_integrations/nrmicro/example/client/client.go)
  * [Publisher and Subscriber Example](https://github.com/newrelic/go-agent/blob/master/_integrations/nrmicro/example/pubsub/main.go)
  * [Full godocs Documentation](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrmicro)

* Added support for creating static `WebRequest` instances manually via the `NewStaticWebRequest` function. This can be useful when you want to create a web transaction but don't have an `http.Request` object. Here's an example of creating a static `WebRequest` and using it to mark a transaction as a web transaction:
  ```go
  hdrs := http.Headers{}
  u, _ := url.Parse("http://example.com")
  webReq := newrelic.NewStaticWebRequest(hdrs, u, "GET", newrelic.TransportHTTP)
  txn := app.StartTransaction("My-Transaction", nil, nil)
  txn.SetWebRequest(webReq)
  ```

## 2.10.0

### New Features

* Added support for custom events when using
  [nrlambda](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrlambda).
  Example Lambda handler which creates custom event:

   ```go
   func handler(ctx context.Context) {
		if txn := newrelic.FromContext(ctx); nil != txn {
			txn.Application().RecordCustomEvent("myEvent", map[string]interface{}{
				"zip": "zap",
			})
		}
		fmt.Println("hello world!")
   }
   ```

## 2.9.0

### New Features

* Added support for [gRPC](https://github.com/grpc/grpc-go) monitoring with the new
[_integrations/nrgrpc](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrgrpc)
package.  This package supports instrumentation for servers and clients.

  * [Server Example](https://github.com/newrelic/go-agent/blob/master/_integrations/nrgrpc/example/server/server.go)
  * [Client Example](https://github.com/newrelic/go-agent/blob/master/_integrations/nrgrpc/example/client/client.go)

* Added new
  [ExternalSegment](https://godoc.org/github.com/newrelic/go-agent#ExternalSegment)
  fields `Host`, `Procedure`, and `Library`.  These optional fields are
  automatically populated from the segment's `URL` or `Request` if unset.  Use
  them if you don't have access to a request or URL but still want useful external
  metrics, transaction segment attributes, and span attributes.
  * `Host` is used for external metrics, transaction trace segment names, and
    span event names.  The host of segment's `Request` or `URL` is the default.
  * `Procedure` is used for transaction breakdown metrics.  If set, it should be
    set to the remote procedure being called.  The HTTP method of the segment's `Request` is the default.
  * `Library` is used for external metrics and the `"component"` span attribute.
    If set, it should be set to the framework making the call. `"http"` is the default.

  With the addition of these new fields, external transaction breakdown metrics
  are changed: `External/myhost.com/all` will now report as
  `External/myhost.com/http/GET` (provided the HTTP method is `GET`).

* HTTP Response codes below `100`, except `0` and `5`, are now recorded as
  errors.  This is to support `gRPC` status codes.  If you start seeing
  new status code errors that you would like to ignore, add them to
  `Config.ErrorCollector.IgnoreStatusCodes` or your server side configuration
  settings.

* Improve [logrus](https://github.com/sirupsen/logrus) support by introducing
  [nrlogrus.Transform](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrlogrus#Transform),
  a function which allows you to turn a
  [logrus.Logger](https://godoc.org/github.com/sirupsen/logrus#Logger) instance into a
  [newrelic.Logger](https://godoc.org/github.com/newrelic/go-agent#Logger).
  Example use:

  ```go
  l := logrus.New()
  l.SetLevel(logrus.DebugLevel)
  cfg := newrelic.NewConfig("Your Application Name", "__YOUR_NEW_RELIC_LICENSE_KEY__")
  cfg.Logger = nrlogrus.Transform(l)
  ```

  As a result of this change, the
  [nrlogrus](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrlogrus)
  package requires [logrus](https://github.com/sirupsen/logrus) version `v1.1.0`
  and above.

## 2.8.1

### Bug Fixes

* Removed `nrmysql.NewConnector` since
  [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql) has not yet
  released `mysql.NewConnector`.

## 2.8.0

### New Features

* Support for Real Time Streaming

  * The agent now has support for sending event data to New Relic every five
    seconds, instead of every minute.  As a result, transaction, error, and
    custom events will now be available in New Relic One and Insights dashboards
    in near real time. For more information on how to view your events with a
    five-second refresh, see the documentation.

  * Note that the overall limits on how many events can be sent per minute have
    not changed. Also, span events, metrics, and trace data is unaffected, and
    will still be sent every minute.

* Introduce support for databases using
  [database/sql](https://golang.org/pkg/database/sql/).  This new functionality
  allows you to instrument MySQL, PostgreSQL, and SQLite calls without manually
  creating
  [DatastoreSegment](https://godoc.org/github.com/newrelic/go-agent#DatastoreSegment)s.

  | Database Library Supported | Integration Package |
  | ------------- | ------------- |
  | [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql) | [_integrations/nrmysql](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrmysql) |
  | [lib/pq](https://github.com/lib/pq) | [_integrations/nrpq](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrpq) |
  | [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3) | [_integrations/nrsqlite3](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrsqlite3) |

  Using these database integration packages is easy!  First replace the driver
  with our integration version:

  ```go
  import (
  	// import our integration package in place of "github.com/go-sql-driver/mysql"
  	_ "github.com/newrelic/go-agent/_integrations/nrmysql"
  )

  func main() {
  	// open "nrmysql" in place of "mysql"
  	db, err := sql.Open("nrmysql", "user@unix(/path/to/socket)/dbname")
  }
  ```

  Second, use the `ExecContext`, `QueryContext`, and `QueryRowContext` methods of
  [sql.DB](https://golang.org/pkg/database/sql/#DB),
  [sql.Conn](https://golang.org/pkg/database/sql/#Conn),
  [sql.Tx](https://golang.org/pkg/database/sql/#Tx), and
  [sql.Stmt](https://golang.org/pkg/database/sql/#Stmt) and provide a
  transaction-containing context.  Calls to `Exec`, `Query`, and `QueryRow` do not
  get instrumented.

  ```go
  ctx := newrelic.NewContext(context.Background(), txn)
  row := db.QueryRowContext(ctx, "SELECT count(*) from tables")
  ```

  If you are using a [database/sql](https://golang.org/pkg/database/sql/) database
  not listed above, you can write your own instrumentation for it using
  [InstrumentSQLConnector](https://godoc.org/github.com/newrelic/go-agent#InstrumentSQLConnector),
  [InstrumentSQLDriver](https://godoc.org/github.com/newrelic/go-agent#InstrumentSQLDriver),
  and
  [SQLDriverSegmentBuilder](https://godoc.org/github.com/newrelic/go-agent#SQLDriverSegmentBuilder).
  The integration packages act as examples of how to do this.

  For more information, see the [Go agent documentation on instrumenting datastore segments](https://docs.newrelic.com/docs/agents/go-agent/instrumentation/instrument-go-segments#go-datastore-segments).

### Bug Fixes

* The [http.RoundTripper](https://golang.org/pkg/net/http/#RoundTripper) returned
  by [NewRoundTripper](https://godoc.org/github.com/newrelic/go-agent#NewRoundTripper)
  no longer modifies the request.  Our thanks to @jlordiales for the contribution.

## 2.7.0

### New Features

* Added support for server side configuration.  Server side configuration allows
 you to set the following configuration settings in the New Relic APM UI:

  * `Config.TransactionTracer.Enabled`
  * `Config.ErrorCollector.Enabled`
  * `Config.CrossApplicationTracer.Enabled`
  * `Config.TransactionTracer.Threshold`
  * `Config.TransactionTracer.StackTraceThreshold`
  * `Config.ErrorCollector.IgnoreStatusCodes`

  For more information see the [server side configuration documentation](https://docs.newrelic.com/docs/agents/manage-apm-agents/configuration/server-side-agent-configuration).

* Added support for AWS Lambda functions in the new
  [nrlambda](_integrations/nrlambda)
  package.  Please email <lambda_preview@newrelic.com> if you are interested in
  learning more or previewing New Relic Lambda monitoring.  This instrumentation
  package requires `aws-lambda-go` version
  [v1.9.0](https://github.com/aws/aws-lambda-go/releases) and above.

  * [documentation](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrlambda)
  * [working example](_integrations/nrlambda/example/main.go)

## 2.6.0

### New Features

* Added support for async: the ability to instrument multiple concurrent
  goroutines, or goroutines that access or manipulate the same Transaction.

  The new `Transaction.NewGoroutine() Transaction` method allows
  transactions to create segments in multiple goroutines!

  `NewGoroutine` returns a new reference to the `Transaction`.  This must be
  called any time you are passing the `Transaction` to another goroutine which
  makes segments.  Each segment-creating goroutine must have its own `Transaction`
  reference.  It does not matter if you call this before or after the other
  goroutine has started.

  All `Transaction` methods can be used in any `Transaction` reference.  The
  `Transaction` will end when `End()` is called in any goroutine.

  Example passing a new `Transaction` reference directly to another goroutine:

  ```go
  	go func(txn newrelic.Transaction) {
  		defer newrelic.StartSegment(txn, "async").End()
  		time.Sleep(100 * time.Millisecond)
  	}(txn.NewGoroutine())
  ```

  Example passing a new `Transaction` reference on a channel to another
  goroutine:

  ```go
  	ch := make(chan newrelic.Transaction)
  	go func() {
  		txn := <-ch
  		defer newrelic.StartSegment(txn, "async").End()
  		time.Sleep(100 * time.Millisecond)
  	}()
  	ch <- txn.NewGoroutine()
  ```

* Added integration support for
  [`aws-sdk-go`](https://github.com/aws/aws-sdk-go) and
  [`aws-sdk-go-v2`](https://github.com/aws/aws-sdk-go-v2).

  When using these SDKs, a segment will be created for each out going request.
  For DynamoDB calls, these will be Datastore segments and for all others they
  will be External segments.
  * [v1 Documentation](http://godoc.org/github.com/newrelic/go-agent/_integrations/nrawssdk/v1)
  * [v2 Documentation](http://godoc.org/github.com/newrelic/go-agent/_integrations/nrawssdk/v2)

* Added span event and transaction trace segment attribute configuration.  You
  may control which attributes are captured in span events and transaction trace
  segments using the `Config.SpanEvents.Attributes` and
  `Config.TransactionTracer.Segments.Attributes` settings. For example, if you
  want to disable the collection of `"db.statement"` in your span events, modify
  your config like this:

  ```go
  cfg.SpanEvents.Attributes.Exclude = append(cfg.SpanEvents.Attributes.Exclude,
  	newrelic.SpanAttributeDBStatement)
  ```

  To disable the collection of all attributes from your transaction trace
  segments, modify your config like this:

  ```go
  cfg.TransactionTracer.Segments.Attributes.Enabled = false
  ```

### Bug Fixes

* Fixed a bug that would prevent External Segments from being created under
  certain error conditions related to Cross Application Tracing.

### Miscellaneous

* Improved linking between Cross Application Transaction Traces in the APM UI.
  When `Config.CrossApplicationTracer.Enabled = true`, External segments in the
  Transaction Traces details will now link to the downstream Transaction Trace
  if there is one. Additionally, the segment name will now include the name of
  the downstream application and the name of the downstream transaction.

* Update attribute names of Datastore and External segments on Transaction
  Traces to be in line with attribute names on Spans. Specifically:
    * `"uri"` => `"http.url"`
    * `"query"` => `"db.statement"`
    * `"database_name"` => `"db.instance"`
    * `"host"` => `"peer.hostname"`
    * `"port_path_or_id"` + `"host"` => `"peer.address"`

## 2.5.0

* Added support for [New Relic Browser](https://docs.newrelic.com/docs/browser)
  using the new `BrowserTimingHeader` method on the
  [`Transaction`](https://godoc.org/github.com/newrelic/go-agent#Transaction)
  which returns a
  [BrowserTimingHeader](https://godoc.org/github.com/newrelic/go-agent#BrowserTimingHeader).
  The New Relic Browser JavaScript code measures page load timing, also known as
  real user monitoring.  The Pro version of this feature measures AJAX requests,
  single-page applications, JavaScript errors, and much more!  Example use:

```go
func browser(w http.ResponseWriter, r *http.Request) {
	hdr, err := w.(newrelic.Transaction).BrowserTimingHeader()
	if nil != err {
		log.Printf("unable to create browser timing header: %v", err)
	}
	// BrowserTimingHeader() will always return a header whose methods can
	// be safely called.
	if js := hdr.WithTags(); js != nil {
		w.Write(js)
	}
	io.WriteString(w, "browser header page")
}
```

* The Go agent now collects an attribute named `request.uri` on Transaction
  Traces, Transaction Events, Error Traces, and Error Events.  `request.uri`
  will never contain user, password, query parameters, or fragment.  To prevent
  the request's URL from being collected in any data, modify your `Config` like
  this:

```go
cfg.Attributes.Exclude = append(cfg.Attributes.Exclude, newrelic.AttributeRequestURI)
```

## 2.4.0

* Introduced `Transaction.Application` method which returns the `Application`
  that started the `Transaction`.  This method is useful since it may prevent
  having to pass the `Application` to code that already has access to the
  `Transaction`.  Example use:

```go
txn.Application().RecordCustomEvent("customerOrder", map[string]interface{}{
	"numItems":   2,
	"totalPrice": 13.75,
})
```

* The `Transaction.AddAttribute` method no longer accepts `nil` values since
  our backend ignores them.

## 2.3.0

* Added support for [Echo](https://echo.labstack.com) in the new `nrecho`
  package.
  * [Documentation](http://godoc.org/github.com/newrelic/go-agent/_integrations/nrecho)
  * [Example](_integrations/nrecho/example/main.go)

* Introduced `Transaction.SetWebResponse(http.ResponseWriter)` method which sets
  the transaction's response writer.  After calling this method, the
  `Transaction` may be used in place of the `http.ResponseWriter` to intercept
  the response code.  This method is useful when the `http.ResponseWriter` is
  not available at the beginning of the transaction (if so, it can be given as a
  parameter to `Application.StartTransaction`).  This method will return a
  reference to the transaction which implements the combination of
  `http.CloseNotifier`, `http.Flusher`, `http.Hijacker`, and `io.ReaderFrom`
  implemented by the ResponseWriter.  Example:

```go
func setResponseDemo(txn newrelic.Transaction) {
	recorder := httptest.NewRecorder()
	txn = txn.SetWebResponse(recorder)
	txn.WriteHeader(200)
	fmt.Println("response code recorded:", recorder.Code)
}
```

* The `Transaction`'s `http.ResponseWriter` methods may now be called safely if
  a `http.ResponseWriter` has not been set.  This allows you to add a response code
  to the transaction without using a `http.ResponseWriter`.  Example:

```go
func transactionWithResponseCode(app newrelic.Application) {
       txn := app.StartTransaction("hasResponseCode", nil, nil)
       defer txn.End()
       txn.WriteHeader(200) // Safe!
}
```

* The agent now collects environment variables prefixed by
  `NEW_RELIC_METADATA_`.  Some of these may be added
  Transaction events to provide context between your Kubernetes cluster and your
  services. For details on the benefits (currently in beta) see [this blog
  post](https://blog.newrelic.com/engineering/monitoring-application-performance-in-kubernetes/)

* The agent now collects the `KUBERNETES_SERVICE_HOST` environment variable to
  detect when the application is running on Kubernetes.

* The agent now collects the fully qualified domain name of the host and
  local IP addresses for improved linking with our infrastructure product.

## 2.2.0

* The `Transaction` parameter to
[NewRoundTripper](https://godoc.org/github.com/newrelic/go-agent#NewRoundTripper)
and
[StartExternalSegment](https://godoc.org/github.com/newrelic/go-agent#StartExternalSegment)
is now optional:  If it is `nil`, then a `Transaction` will be looked for in the
request's context (using
[FromContext](https://godoc.org/github.com/newrelic/go-agent#FromContext)).
Passing a `nil` transaction is **STRONGLY** recommended when using
[NewRoundTripper](https://godoc.org/github.com/newrelic/go-agent#NewRoundTripper)
since it allows one `http.Client.Transport` to be used for multiple
transactions.  Example use:

```go
client := &http.Client{}
client.Transport = newrelic.NewRoundTripper(nil, client.Transport)
request, _ := http.NewRequest("GET", "http://example.com", nil)
request = newrelic.RequestWithTransactionContext(request, txn)
resp, err := client.Do(request)
```

* Introduced `Transaction.SetWebRequest(WebRequest)` method which marks the
transaction as a web transaction.  If the `WebRequest` parameter is non-nil,
`SetWebRequest` will collect details on request attributes, url, and method.
This method is useful if you don't have access to the request at the beginning
of the transaction, or if your request is not an `*http.Request` (just add
methods to your request that satisfy
[WebRequest](https://godoc.org/github.com/newrelic/go-agent#WebRequest)).  To
use an `*http.Request` as the parameter, use the
[NewWebRequest](https://godoc.org/github.com/newrelic/go-agent#NewWebRequest)
transformation function.  Example:

```go
var request *http.Request = getInboundRequest()
txn.SetWebRequest(newrelic.NewWebRequest(request))
```

* Fixed `Debug` in `nrlogrus` package.  Previous versions of the New Relic Go Agent incorrectly
logged to Info level instead of Debug.  This has now been fixed.  Thanks to @paddycarey for catching this.

* [nrgin.Transaction](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrgin/v1#Transaction)
may now be called with either a `context.Context` or a `*gin.Context`.  If you were passing a `*gin.Context`
around your functions as a `context.Context`, you may access the Transaction by calling either
[nrgin.Transaction](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrgin/v1#Transaction)
or [FromContext](https://godoc.org/github.com/newrelic/go-agent#FromContext).
These functions now work nicely together.
For example, [FromContext](https://godoc.org/github.com/newrelic/go-agent#FromContext) will return the `Transaction`
added by [nrgin.Middleware](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrgin/v1#Middleware).
Thanks to @rodriguezgustavo for the suggestion.  

## 2.1.0

* The Go Agent now supports distributed tracing.

  Distributed tracing lets you see the path that a request takes as it travels through your distributed system. By
  showing the distributed activity through a unified view, you can troubleshoot and understand a complex system better
  than ever before.

  Distributed tracing is available with an APM Pro or equivalent subscription. To see a complete distributed trace, you
  need to enable the feature on a set of neighboring services. Enabling distributed tracing changes the behavior of
  some New Relic features, so carefully consult the
  [transition guide](https://docs.newrelic.com/docs/transition-guide-distributed-tracing) before you enable this
  feature.

  To enable distributed tracing, set the following fields in your config.  Note that distributed tracing and cross
  application tracing cannot be used simultaneously.

```
  config := newrelic.NewConfig("Your Application Name", "__YOUR_NEW_RELIC_LICENSE_KEY__")
  config.CrossApplicationTracer.Enabled = false
  config.DistributedTracer.Enabled = true
```

  Please refer to the
  [distributed tracing section of the guide](GUIDE.md#distributed-tracing)
  for more detail on how to ensure you get the most out of the Go agent's distributed tracing support.

* Added functions [NewContext](https://godoc.org/github.com/newrelic/go-agent#NewContext)
  and [FromContext](https://godoc.org/github.com/newrelic/go-agent#FromContext)
  for adding and retrieving the Transaction from a Context.  Handlers
  instrumented by
  [WrapHandle](https://godoc.org/github.com/newrelic/go-agent#WrapHandle),
  [WrapHandleFunc](https://godoc.org/github.com/newrelic/go-agent#WrapHandleFunc),
  and [nrgorilla.InstrumentRoutes](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrgorilla/v1#InstrumentRoutes)
  may use [FromContext](https://godoc.org/github.com/newrelic/go-agent#FromContext)
  on the request's context to access the Transaction.
  Thanks to @caarlos0 for the contribution!  Though [NewContext](https://godoc.org/github.com/newrelic/go-agent#NewContext)
  and [FromContext](https://godoc.org/github.com/newrelic/go-agent#FromContext)
  require Go 1.7+ (when [context](https://golang.org/pkg/context/) was added),
  [RequestWithTransactionContext](https://godoc.org/github.com/newrelic/go-agent#RequestWithTransactionContext) is always exported so that it can be used in all framework and library
  instrumentation.

## 2.0.0

* The `End()` functions defined on the `Segment`, `DatastoreSegment`, and
  `ExternalSegment` types now receive the segment as a pointer, rather than as
  a value. This prevents unexpected behaviour when a call to `End()` is
  deferred before one or more fields are changed on the segment.

  In practice, this is likely to only affect this pattern:

    ```go
    defer newrelic.DatastoreSegment{
      // ...
    }.End()
    ```

  Instead, you will now need to separate the literal from the deferred call:

    ```go
    ds := newrelic.DatastoreSegment{
      // ...
    }
    defer ds.End()
    ```

  When creating custom and external segments, we recommend using
  [`newrelic.StartSegment()`](https://godoc.org/github.com/newrelic/go-agent#StartSegment)
  and
  [`newrelic.StartExternalSegment()`](https://godoc.org/github.com/newrelic/go-agent#StartExternalSegment),
  respectively.

* Added GoDoc badge to README.  Thanks to @mrhwick for the contribution!

* `Config.UseTLS` configuration setting has been removed to increase security.
   TLS will now always be used in communication with New Relic Servers.

## 1.11.0

* We've closed the Issues tab on GitHub. Please visit our
  [support site](https://support.newrelic.com) to get timely help with any
  problems you're having, or to report issues.

* Added support for Cross Application Tracing (CAT). Please refer to the
  [CAT section of the guide](GUIDE.md#cross-application-tracing)
  for more detail on how to ensure you get the most out of the Go agent's new
  CAT support.

* The agent now collects additional metadata when running within Amazon Web
  Services, Google Cloud Platform, Microsoft Azure, and Pivotal Cloud Foundry.
  This information is used to provide an enhanced experience when the agent is
  deployed on those platforms.

## 1.10.0

* Added new `RecordCustomMetric` method to [Application](https://godoc.org/github.com/newrelic/go-agent#Application).
  This functionality can be used to track averages or counters without using
  custom events.
  * [Custom Metric Documentation](https://docs.newrelic.com/docs/agents/manage-apm-agents/agent-data/collect-custom-metrics)

* Fixed import needed for logrus.  The import Sirupsen/logrus had been renamed to sirupsen/logrus.
  Thanks to @alfred-landrum for spotting this.

* Added [ErrorAttributer](https://godoc.org/github.com/newrelic/go-agent#ErrorAttributer),
  an optional interface that can be implemented by errors provided to
  `Transaction.NoticeError` to attach additional attributes.  These attributes are
  subject to attribute configuration.

* Added [Error](https://godoc.org/github.com/newrelic/go-agent#Error), a type
  that allows direct control of error fields.  Example use:

```go
txn.NoticeError(newrelic.Error{
	// Message is returned by the Error() method.
	Message: "error message: something went very wrong",
	Class:   "errors are aggregated by class",
	Attributes: map[string]interface{}{
		"important_number": 97232,
		"relevant_string":  "zap",
	},
})
```

* Updated license to address scope of usage.

## 1.9.0

* Added support for [github.com/gin-gonic/gin](https://github.com/gin-gonic/gin)
  in the new `nrgin` package.
  * [Documentation](http://godoc.org/github.com/newrelic/go-agent/_integrations/nrgin/v1)
  * [Example](examples/_gin/main.go)

## 1.8.0

* Fixed incorrect metric rule application when the metric rule is flagged to
  terminate and matches but the name is unchanged.

* `Segment.End()`, `DatastoreSegment.End()`, and `ExternalSegment.End()` methods now return an
  error which may be helpful in diagnosing situations where segment data is unexpectedly missing.

## 1.7.0

* Added support for [gorilla/mux](http://github.com/gorilla/mux) in the new `nrgorilla`
  package.
  * [Documentation](http://godoc.org/github.com/newrelic/go-agent/_integrations/nrgorilla/v1)
  * [Example](examples/_gorilla/main.go)

## 1.6.0

* Added support for custom error messages and stack traces.  Errors provided
  to `Transaction.NoticeError` will now be checked to see if
  they implement [ErrorClasser](https://godoc.org/github.com/newrelic/go-agent#ErrorClasser)
  and/or [StackTracer](https://godoc.org/github.com/newrelic/go-agent#StackTracer).
  Thanks to @fgrosse for this proposal.

* Added support for [pkg/errors](https://github.com/pkg/errors).  Thanks to
  @fgrosse for this work.
  * [documentation](https://godoc.org/github.com/newrelic/go-agent/_integrations/nrpkgerrors)
  * [example](https://github.com/newrelic/go-agent/blob/master/_integrations/nrpkgerrors/nrpkgerrors.go)

* Fixed tests for Go 1.8.

## 1.5.0

* Added support for Windows.  Thanks to @ianomad and @lvxv for the contributions.

* The number of heap objects allocated is recorded in the
  `Memory/Heap/AllocatedObjects` metric.  This will soon be displayed on the "Go
  runtime" page.

* If the [DatastoreSegment](https://godoc.org/github.com/newrelic/go-agent#DatastoreSegment)
  fields `Host` and `PortPathOrID` are not provided, they will no longer appear
  as `"unknown"` in transaction traces and slow query traces.

* Stack traces will now be nicely aligned in the APM UI.

## 1.4.0

* Added support for slow query traces.  Slow datastore segments will now
 generate slow query traces viewable on the datastore tab.  These traces include
 a stack trace and help you to debug slow datastore activity.
 [Slow Query Documentation](https://docs.newrelic.com/docs/apm/applications-menu/monitoring/viewing-slow-query-details)

* Added new
[DatastoreSegment](https://godoc.org/github.com/newrelic/go-agent#DatastoreSegment)
fields `ParameterizedQuery`, `QueryParameters`, `Host`, `PortPathOrID`, and
`DatabaseName`.  These fields will be shown in transaction traces and in slow
query traces.

## 1.3.0

* Breaking Change: Added a timeout parameter to the `Application.Shutdown` method.

## 1.2.0

* Added support for instrumenting short-lived processes:
  * The new `Application.Shutdown` method allows applications to report
    data to New Relic without waiting a full minute.
  * The new `Application.WaitForConnection` method allows your process to
    defer instrumentation until the application is connected and ready to
    gather data.
  * Full documentation here: [application.go](application.go)
  * Example short-lived process: [examples/short-lived-process/main.go](examples/short-lived-process/main.go)

* Error metrics are no longer created when `ErrorCollector.Enabled = false`.

* Added support for [github.com/mgutz/logxi](github.com/mgutz/logxi).  See
  [_integrations/nrlogxi/v1/nrlogxi.go](_integrations/nrlogxi/v1/nrlogxi.go).

* Fixed bug where Transaction Trace thresholds based upon Apdex were not being
  applied to background transactions.

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
  * [examples/server/main.go](examples/server/main.go)
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
