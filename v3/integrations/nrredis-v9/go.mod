module github.com/newrelic/go-agent/v3/integrations/nrredis-v9

// https://github.com/redis/go-redis/blob/a38f75b640398bd709ee46c778a23e80e09d48b5/go.mod#L3
go 1.20

require (
	github.com/newrelic/go-agent/v3 v3.32.0
	github.com/redis/go-redis/v9 v9.0.2
)


replace github.com/newrelic/go-agent/v3 => ../..
