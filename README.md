# New Relic Go Agent [![GoDoc](https://godoc.org/github.com/newrelic/go-agent?status.svg)](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic/) [![Go Report Card](https://goreportcard.com/badge/github.com/newrelic/go-agent)](https://goreportcard.com/report/github.com/newrelic/go-agent)

## Description

The New Relic Go Agent allows you to monitor your Go applications with New
Relic.  It helps you track transactions, outbound requests, database calls, and
other parts of your Go application's behavior and provides a running overview of
garbage collection, goroutine activity, and memory use.

All pull requests will be reviewed by the New Relic product team. Any questions or issues should be directed to our [support
site](http://support.newrelic.com/) or our [community
forum](https://discuss.newrelic.com).

## Upgrading
If you have already been using version 2.X of the agent and are upgrading to
version 3.0, see our [Migration Guide](MIGRATION.md) for details.

## Requirements

For the latest version of the agent, Go 1.7+ is required, due to the use of `context.Context`.
(For versions 2.X and earlier of the Go agent, Go 1.3+ is required.)

Linux, OS X, and Windows (Vista, Server 2008 and later) are supported.

## Integrations

The following [integration packages](https://godoc.org/github.com/newrelic/go-agent/v3/integrations)
extend the base [newrelic](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic/) package
to support the following frameworks and libraries.
Frameworks and databases which don't have an integration package may still be
instrumented using the [newrelic](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic/)
package primitives.  Specifically, more information about instrumenting your database using
these primitives can be found [here](GUIDE.md#datastore-segments).

<!---
NOTE!  When updating the table below, be sure to update the docs site version too:
https://docs.newrelic.com/docs/agents/go-agent/get-started/go-agent-compatibility-requirements
-->

| Project | Integration Package |  |
| ------------- | ------------- | - |
| [aws/aws-sdk-go](https://github.com/aws/aws-sdk-go) | [v3/integrations/nrawssdk-v1](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrawssdk-v1) | Instrument outbound calls made using Go AWS SDK |
| [aws/aws-sdk-go-v2](https://github.com/aws/aws-sdk-go-v2) | [v3/integrations/nrawssdk-v2](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrawssdk-v2) | Instrument outbound calls made using Go AWS SDK v2 |
| [labstack/echo](https://github.com/labstack/echo) | [v3/integrations/nrecho-v3](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrecho-v3) | Instrument inbound requests through version 3 of the Echo framework |
| [labstack/echo](https://github.com/labstack/echo) | [v3/integrations/nrecho-v4](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrecho-v4) | Instrument inbound requests through version 4 of the Echo framework |
| [gin-gonic/gin](https://github.com/gin-gonic/gin) | [v3/integrations/nrgin](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrgin) | Instrument inbound requests through the Gin framework |
| [gorilla/mux](https://github.com/gorilla/mux) | [v3/integrations/nrgorilla](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrgorilla) | Instrument inbound requests through the Gorilla framework |
| [julienschmidt/httprouter](https://github.com/julienschmidt/httprouter) | [v3/integrations/nrhttprouter](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrhttprouter) | Instrument inbound requests through the HttpRouter framework |
| [aws/aws-lambda-go](https://github.com/aws/aws-lambda-go) | [v3/integrations/nrlambda](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrlambda) | Instrument AWS Lambda applications |
| [sirupsen/logrus](https://github.com/sirupsen/logrus) | [v3/integrations/nrlogrus](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrlogrus) | Send agent log messages to Logrus |
| [mgutz/logxi](https://github.com/mgutz/logxi) | [v3/integrations/nrlogxi](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrlogxi) | Send agent log messages to Logxi |
| [uber-go/zap](https://github.com/uber-go/zap) | [v3/integrations/nrzap](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrzap) | Send agent log messages to Zap |
| [pkg/errors](https://github.com/pkg/errors) | [v3/integrations/nrpkgerrors](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrpkgerrors) | Wrap pkg/errors errors to improve stack traces and error class information |
| [openzipkin/b3-propagation](https://github.com/openzipkin/b3-propagation) | [v3/integrations/nrb3](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrb3) | Add B3 headers to outgoing requests |
| [database/sql](https://godoc.org/database/sql) | Use a supported database driver or [builtin instrumentation](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#InstrumentSQLConnector) | Instrument database calls with SQL |
| [jmoiron/sqlx](https://github.com/jmoiron/sqlx) | Use a supported [database driver](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrpq/example/sqlx) or [builtin instrumentation](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#InstrumentSQLConnector) | Instrument database calls with SQLx |
| [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql) | [v3/integrations/nrmysql](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrmysql) | Instrument MySQL driver |
| [lib/pq](https://github.com/lib/pq) | [v3/integrations/nrpq](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrpq) | Instrument PostgreSQL driver |
| [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3) | [v3/integrations/nrsqlite3](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrsqlite3) | Instrument SQLite driver |
| [mongodb/mongo-go-driver](https://github.com/mongodb/mongo-go-driver) | [v3/integrations/nrmongo](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrmongo) | Instrument MongoDB calls |
| [google.golang.org/grpc](https://github.com/grpc/grpc-go) | [v3/integrations/nrgrpc](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrgrpc) | Instrument gRPC servers and clients |
| [micro/go-micro](https://github.com/micro/go-micro) | [v3/integrations/nrmicro](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrmicro) | Instrument servers, clients, publishers, and subscribers through the Micro framework |
| [nats-io/nats.go](https://github.com/nats-io/nats.go) | [v3/integrations/nrnats](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrnats) | Instrument publishers and subscribers using the NATS client |
| [nats-io/stan.go](https://github.com/nats-io/stan.go) | [v3/integrations/nrstan](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrstan) | Instrument publishers and subscribers using the NATS streaming client |


These integration packages must be imported along
with the [newrelic](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic/) package, as shown in this
[nrgin example](https://github.com/newrelic/go-agent/blob/master/v3/integrations/nrgin/example/main.go).

## Getting Started

Follow the steps in [GETTING_STARTED.md](GETTING_STARTED.md) to instrument your
application.

## Runnable Example

[v3/examples/server/main.go](v3/examples/server/main.go) is an example that
will appear as "Example App" in your New Relic applications list.  To run it:

```
env NEW_RELIC_LICENSE_KEY=__YOUR_NEW_RELIC_LICENSE_KEY__LICENSE__ \
    go run examples/server/main.go
```

Some endpoints exposed are [http://localhost:8000/](http://localhost:8000/)
and [http://localhost:8000/notice_error](http://localhost:8000/notice_error)

## Support

You can find more detailed documentation [in the guide](GUIDE.md) and on
[the New Relic Documentation site](https://docs.newrelic.com/docs/agents/go-agent).

If you can't find what you're looking for there, reach out to us on our [support
site](http://support.newrelic.com/) or our [community
forum](https://discuss.newrelic.com) and we'll be happy to help you.

Find a bug?  Contact us via [support.newrelic.com](http://support.newrelic.com/),
or email support@newrelic.com.
