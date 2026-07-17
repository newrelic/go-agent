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

replace github.com/newrelic/go-agent/v3 => ../..
