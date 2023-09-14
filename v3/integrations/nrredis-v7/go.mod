module github.com/newrelic/go-agent/v3/integrations/nrredis-v7

// https://github.com/go-redis/redis/blob/master/go.mod
go 1.19

require (
	github.com/go-redis/redis/v7 v7.0.0-beta.5
	github.com/newrelic/go-agent/v3 v3.24.1
)


replace github.com/newrelic/go-agent/v3 => ../..
