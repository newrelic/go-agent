module github.com/newrelic/go-agent/v3/integrations/nramqp

go 1.25

require (
	github.com/newrelic/go-agent/v3 v3.42.0
	github.com/rabbitmq/amqp091-go v1.9.0
)
replace github.com/newrelic/go-agent/v3 => ../..
