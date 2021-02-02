module github.com/newrelic/go-agent/v4

go 1.13

require (
	github.com/newrelic/opentelemetry-exporter-go v0.15.1
	go.opentelemetry.io/otel v0.16.0
	google.golang.org/grpc v1.30.0
)

replace github.com/newrelic/go-agent/v4/internal => ./internal