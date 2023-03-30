[![Community Plus header](https://github.com/newrelic/opensource-website/raw/main/src/images/categories/Community_Plus.png)](https://opensource.newrelic.com/oss-category/#community-plus)

# New Relic Go Agent [![GoDoc](https://godoc.org/github.com/newrelic/go-agent?status.svg)](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic/) [![Go Report Card](https://goreportcard.com/badge/github.com/newrelic/go-agent)](https://goreportcard.com/report/github.com/newrelic/go-agent)

The New Relic Go Agent allows you to monitor your Go applications with New
Relic. It helps you track transactions, outbound requests, database calls, and
other parts of your Go application's behavior and provides a running overview of
garbage collection, goroutine activity, and memory use.

Go is a compiled language, and doesn’t use a virtual machine. This means that setting up New Relic for your Golang app requires you to use our Go agent API and manually add New Relic methods to your source code. Our API provides exceptional flexibility and control over what gets instrumented.

## Installation

### Compatibility and Requirements

For the latest version of the agent, Go 1.17+ is required.

Linux, OS X, and Windows (Vista, Server 2008 and later) are supported.

### Installing and using the Go agent

To install the agent, follow the instructions in our [GETTING_STARTED](https://github.com/newrelic/go-agent/blob/master/GETTING_STARTED.md)
document or our [GUIDE](https://github.com/newrelic/go-agent/blob/master/GUIDE.md).

We recommend instrumenting your Go code to get the maximum benefits from the
New Relic Go agent. But we make it easy to get great data in couple of ways:

* Even without adding instrumentation, just importing the agent and creating an
application will provide useful runtime information about your number of goroutines,
garbage collection statistics, and memory and CPU usage.
* You can use our many [INTEGRATION packages](https://github.com/newrelic/go-agent/tree/master/v3/integrations)
for out-of-the box support for many popular Go web frameworks and libraries. We
continue to add integration packages based on your feedback. You can weigh in on
potential integrations by opening an `Issue` here in our New Relic Go agent GitHub project.

### Upgrading

If you have already been using version 2.X of the agent and are upgrading to
version 3.0, see our [MIGRATION guide](MIGRATION.md) for details.

## Getting Started

[v3/examples/server/main.go](v3/examples/server/main.go) is an example that
will appear as "Example App" in your New Relic applications list.  To run it:

```
env NEW_RELIC_LICENSE_KEY=__YOUR_NEW_RELIC_LICENSE_KEY__LICENSE__ \
    go run v3/examples/server/main.go
```

Some endpoints exposed are [http://localhost:8000/](http://localhost:8000/)
and [http://localhost:8000/notice_error](http://localhost:8000/notice_error)

## Usage

### Integration Packages

The following [integration packages](https://godoc.org/github.com/newrelic/go-agent/v3/integrations)
extend the base [newrelic](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic/) package
to support the following frameworks and libraries.
Frameworks and databases which don't have an integration package may still be
instrumented using the [newrelic](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic/)
package primitives.

<!---
NOTE! When updating the tables below, be sure to update the docs site version too:
https://docs.newrelic.com/docs/agents/go-agent/get-started/go-agent-compatibility-requirements
-->

#### Service Frameworks

| Project | Integration Package |  |
| ------------- | ------------- | - |
| [gin-gonic/gin](https://github.com/gin-gonic/gin) | [v3/integrations/nrgin](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrgin) | Instrument inbound requests through the Gin framework |
| [gorilla/mux](https://github.com/gorilla/mux) | [v3/integrations/nrgorilla](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrgorilla) | Instrument inbound requests through the Gorilla framework |
| [google.golang.org/grpc](https://github.com/grpc/grpc-go) | [v3/integrations/nrgrpc](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrgrpc) | Instrument gRPC servers and clients |
| [labstack/echo](https://github.com/labstack/echo) | [v3/integrations/nrecho-v3](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrecho-v3) | Instrument inbound requests through version 3 of the Echo framework |
| [labstack/echo](https://github.com/labstack/echo) | [v3/integrations/nrecho-v4](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrecho-v4) | Instrument inbound requests through version 4 of the Echo framework |
| [julienschmidt/httprouter](https://github.com/julienschmidt/httprouter) | [v3/integrations/nrhttprouter](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrhttprouter) | Instrument inbound requests through the HttpRouter framework |
| [micro/go-micro](https://github.com/micro/go-micro) | [v3/integrations/nrmicro](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrmicro) | Instrument servers, clients, publishers, and subscribers through the Micro framework |

#### Datastores

More information about instrumenting databases without an integration package
using [newrelic](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic/)
package primitives can be found [here](GUIDE.md#datastore-segments).

| Project | Integration Package |  |
| ------------- | ------------- | - |
| [lib/pq](https://github.com/lib/pq) | [v3/integrations/nrpq](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrpq) | Instrument PostgreSQL driver (`pq` driver for `database/sql`) |
| [jackc/pgx](https://github.com/jackc/pgx) | [v3/integrations/nrpgx](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrpgx) | Instrument PostgreSQL driver (`pgx` driver for `database/sql`)|
| [jackc/pgx/v5](https://github.com/jackc/pgx/v5) | [v3/integrations/nrpgx5](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrpgx5) | Instrument PostgreSQL driver (`pgx/v5` driver for `database/sql`)|
| [go-mssqldb](github.com/denisenkom/go-mssqldb) | [v3/integrations/nrmssql](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrmssql) | Instrument MS SQL driver |
| [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql) | [v3/integrations/nrmysql](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrmysql) | Instrument MySQL driver |
| [elastic/go-elasticsearch](https://github.com/elastic/go-elasticsearch) | [v3/integrations/nrelasticsearch-v7](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrelasticsearch-v7) | Instrument Elasticsearch datastore calls |
| [database/sql](https://godoc.org/database/sql) | Use a supported database driver or [builtin instrumentation](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#InstrumentSQLConnector) | Instrument database calls with SQL |
| [jmoiron/sqlx](https://github.com/jmoiron/sqlx) | Use a supported [database driver](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrpq/example/sqlx) or [builtin instrumentation](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic#InstrumentSQLConnector) | Instrument database calls with SQLx |
| [go-redis/redis](https://github.com/go-redis/redis) | [v3/integrations/nrredis-v7](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrredis-v7) | Instrument Redis 7 calls |
| [go-redis/redis](https://github.com/go-redis/redis) | [v3/integrations/nrredis-v8](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrredis-v8) | Instrument Redis 8 calls |
| [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3) | [v3/integrations/nrsqlite3](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrsqlite3) | Instrument SQLite driver |
| [snowflakedb/gosnowflake](https://github.com/snowflakedb/gosnowflake) | [v3/integrations/nrsnowflake](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrsnowflake) | Instrument Snowflake driver |
| [mongodb/mongo-go-driver](https://github.com/mongodb/mongo-go-driver) | [v3/integrations/nrmongo](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrmongo) | Instrument MongoDB calls |

#### Agent Logging

| Project | Integration Package |  |
| ------------- | ------------- | - |
| [sirupsen/logrus](https://github.com/sirupsen/logrus) | [v3/integrations/nrlogrus](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrlogrus) | Send agent log messages to Logrus |
| [mgutz/logxi](https://github.com/mgutz/logxi) | [v3/integrations/nrlogxi](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrlogxi) | Send agent log messages to Logxi |
| [uber-go/zap](https://github.com/uber-go/zap) | [v3/integrations/nrzap](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrzap) | Send agent log messages to Zap |

#### Logs in Context

| Project | Integration Package |  |
| ------------- | ------------- | - |
| [sirupsen/logrus](https://github.com/sirupsen/logrus) | [v3/integrations/logcontext-v2/nrlogrus](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/logcontext-v2/nrlogrus) | Send data collected from Logrus log messages to New Relic |
| [log](https://pkg.go.dev/log) | [v3/integrations/logcontext-v2/logWriter](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/logcontext-v2/logWriter) | Send data collected from the standard library logger log messages to New Relic |
| [rs/zerolog](https://github.com/rs/zerolog) | [v3/integrations/logcontext-v2/zerologWriter](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/logcontext-v2/zerologWriter) | Send data collected from zerolog log messages to New Relic |

#### AWS

| Project | Integration Package |  |
| ------------- | ------------- | - |
| [aws/aws-sdk-go](https://github.com/aws/aws-sdk-go) | [v3/integrations/nrawssdk-v1](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrawssdk-v1) | Instrument outbound calls made using Go AWS SDK |
| [aws/aws-sdk-go-v2](https://github.com/aws/aws-sdk-go-v2) | [v3/integrations/nrawssdk-v2](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrawssdk-v2) | Instrument outbound calls made using Go AWS SDK v2 |
| [aws/aws-lambda-go](https://github.com/aws/aws-lambda-go) | [v3/integrations/nrlambda](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrlambda) | Instrument AWS Lambda applications |

#### GraphQL

| Project | Integration Package |  |
| ------------- | ------------- | - |
| [graph-gophers/graphql-go](https://github.com/graph-gophers/graphql-go) | [v3/integrations/nrgraphgophers](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrgraphgophers) | Instrument inbound requests using graph-gophers/graphql-go |
| [graphql-go/graphql](https://github.com/graphql-go/graphql) | [v3/integrations/nrgraphqlgo](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrgraphqlgo) | Instrument inbound requests using graphql-go/graphql |

#### Misc

| Project | Integration Package |  |
| ------------- | ------------- | - |
| [pkg/errors](https://github.com/pkg/errors) | [v3/integrations/nrpkgerrors](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrpkgerrors) | Wrap pkg/errors errors to improve stack traces and error class information |
| [openzipkin/b3-propagation](https://github.com/openzipkin/b3-propagation) | [v3/integrations/nrb3](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrb3) | Add B3 headers to outgoing requests |
| [nats-io/nats.go](https://github.com/nats-io/nats.go) | [v3/integrations/nrnats](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrnats) | Instrument publishers and subscribers using the NATS client |
| [nats-io/stan.go](https://github.com/nats-io/stan.go) | [v3/integrations/nrstan](https://godoc.org/github.com/newrelic/go-agent/v3/integrations/nrstan) | Instrument publishers and subscribers using the NATS streaming client |


These integration packages must be imported along
with the [newrelic](https://godoc.org/github.com/newrelic/go-agent/v3/newrelic/) package, as shown in this
[nrgin example](https://github.com/newrelic/go-agent/blob/master/v3/integrations/nrgin/example/main.go).

### Alternatives

If you are already using another open source solution to gather telemetry data, you may find it easier to use one of our open source exporters to send this data to New Relic:

* OpenTelemetry: [github.com/newrelic/opentelemetry-exporter-go](https://github.com/newrelic/opentelemetry-exporter-go)
* OpenCensus: [github.com/newrelic/newrelic-opencensus-exporter-go](https://github.com/newrelic/newrelic-opencensus-exporter-go)
* Prometheus Exporter: [github.com/newrelic/nri-prometheus](https://github.com/newrelic/nri-prometheus)
* Istio Adapter: [github.com/newrelic/newrelic-istio-adapter](https://github.com/newrelic/newrelic-istio-adapter)
* Telemetry SDK: [github.com/newrelic/newrelic-telemetry-sdk-go](https://github.com/newrelic/newrelic-telemetry-sdk-go)


## Support

Should you need assistance with New Relic products, you are in good hands with several support channels.  

If the issue has been confirmed as a bug or is a Feature request, please file a Github issue.


* [Go Agent GUIDE](GUIDE.md): Step by step how-to for key agent features
* [New Relic Documentation](https://docs.newrelic.com/docs/agents/go-agent): Comprehensive guidance for using our platform
* [Troubleshooting framework](https://discuss.newrelic.com/t/troubleshooting-frameworks/108787): Steps you through common troubleshooting questions
* [New Relic Community](https://discuss.newrelic.com/tags/goagent): The best place to engage in troubleshooting questions
* [New Relic Developer](https://developer.newrelic.com/): Resources for building a custom observability applications
* [New Relic University](https://learn.newrelic.com/): A range of online training for New Relic users of every level

## Privacy

At New Relic we take your privacy and the security of your information seriously, and are committed to protecting your information. We must emphasize the importance of not sharing personal data in public forums, and ask all users to scrub logs and diagnostic information for sensitive information, whether personal, proprietary, or otherwise.

We define "Personal Data" as any information relating to an identified or identifiable individual, including, for example, your name, phone number, post code or zip code, Device ID, IP address and email address.

For more information, review [New Relic’s General Data Privacy Notice](https://newrelic.com/termsandconditions/privacy).

## Contribute

We encourage your contributions to improve the Go Agent!  Keep in mind when you submit your pull request, you'll need to sign the CLA via the click-through using CLA-Assistant.  You only have to sign the CLA one time per project.  

If you have any questions, or to execute our corporate CLA, required if your contribution is on behalf of a company, please drop us an email at opensource@newrelic.com.


**A note about vulnerabilities**

As noted in our [security policy](https://github.com/newrelic/go-agent/security/policy), New Relic is committed to the privacy and security of our customers and their data. We believe that providing coordinated disclosure by security researchers and engaging with the security community are important means to achieve our security goals.

If you believe you have found a security vulnerability in this project or any of New Relic's products or websites, we welcome and greatly appreciate you reporting it to New Relic through [HackerOne](https://hackerone.com/newrelic).

If you would like to contribute to this project, please review [these guidelines](./CONTRIBUTING.md).

To [all contributors](https://github.com/newrelic/go-agent/graphs/contributors), we thank you!  Without your contribution, this project would not be what it is today.  We also host a community project page dedicated to 
the [Go Agent](https://opensource.newrelic.com/projects/newrelic/go-agent).

## License
The New Relic Go agent is licensed under the [Apache 2.0](http://apache.org/licenses/LICENSE-2.0.txt) License.
