module github.com/newrelic/go-agent/v3/integrations/nrsarama

go 1.19

require (
	github.com/Shopify/sarama v1.38.1
	github.com/newrelic/go-agent/v3 v3.24.1
	github.com/stretchr/testify v1.8.1
)


replace github.com/newrelic/go-agent/v3 => ../..
