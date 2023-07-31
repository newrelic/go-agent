module github.com/newrelic/go-agent/v3/integrations/nrredis-v8

// As of Jan 2020, go 1.11 is in the go-redis go.mod file:
// https://github.com/go-redis/redis/blob/master/go.mod
go 1.11

require (
	github.com/go-redis/redis/v8 v8.4.0
	github.com/newrelic/go-agent/v3 v3.0.0
)
