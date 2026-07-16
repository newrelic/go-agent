module github.com/newrelic/go-agent/v3/integrations/nrotelhybrid

go 1.25.0

require (
	github.com/newrelic/go-agent/v3 v3.44.1
	go.opentelemetry.io/otel v1.44.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.44.0
	go.opentelemetry.io/otel/sdk v1.44.0
	go.opentelemetry.io/otel/sdk/metric v1.44.0
	go.opentelemetry.io/otel/trace v1.44.0
)

require github.com/felixge/httpsnoop v1.0.4 // indirect

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.69.0
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
)

replace github.com/newrelic/go-agent/v3 => ../..
