module github.com/newrelic/go-agent/v3/integrations/nrotelhybrid

go 1.25.0

require (
	github.com/lib/pq v1.12.3
	github.com/newrelic/go-agent/v3 v3.44.1
	github.com/rabbitmq/amqp091-go v1.14.0
	github.com/uptrace/opentelemetry-go-extra/otelsql v0.3.2
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.71.0
	go.opentelemetry.io/otel v1.46.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.44.0
	go.opentelemetry.io/otel/sdk v1.46.0
	go.opentelemetry.io/otel/trace v1.46.0
)

replace github.com/newrelic/go-agent/v3 => ../..
