module github.com/newrelic/go-agent/v4/integrations/nrredis-v7

// As of Jan 2020, go 1.11 is in the go-redis go.mod file:
// https://github.com/go-redis/redis/blob/master/go.mod
go 1.11

require (
	github.com/go-redis/redis/v7 v7.0.0-beta.5
	github.com/newrelic/go-agent/v4 v4.0.0
)
