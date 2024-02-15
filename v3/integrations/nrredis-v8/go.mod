module github.com/newrelic/go-agent/v3/integrations/nrredis-v8

// https://github.com/go-redis/redis/blob/master/go.mod
go 1.19

require (
	github.com/go-redis/redis/v8 v8.4.0
	github.com/newrelic/go-agent/v3 v3.30.0
)


replace github.com/newrelic/go-agent/v3 => ../..
