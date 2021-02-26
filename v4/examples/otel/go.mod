module github.com/newrelic/go-agent/v4/examples/otel

go 1.14

require (
	github.com/newrelic/go-agent/v4 v4.0.0
	github.com/newrelic/newrelic-telemetry-sdk-go v0.3.0
	github.com/newrelic/opentelemetry-exporter-go v0.15.1
	go.opentelemetry.io/otel v0.16.0
	go.opentelemetry.io/otel/sdk v0.16.0
	gopkg.in/yaml.v2 v2.2.7 // indirect
)

replace github.com/newrelic/go-agent/v4 v4.0.0 => ../../
