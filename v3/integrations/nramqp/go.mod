module github.com/newrelic/go-agent/v3/integrations/nramqp

go 1.21

require (
	github.com/newrelic/go-agent/v3 v3.36.0
	github.com/rabbitmq/amqp091-go v1.9.0
)
replace github.com/newrelic/go-agent/v3 => ../..
