module github.com/newrelic/go-agent/v3/integrations/nrredis-v9

// As of Mar 2023, go 1.17 is in the go-redis go.mod file:
// https://github.com/redis/go-redis/blob/a38f75b640398bd709ee46c778a23e80e09d48b5/go.mod#L3
go 1.17

require (
	github.com/newrelic/go-agent/v3 v3.20.4
	github.com/redis/go-redis/v9 v9.0.2
)

require (
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	golang.org/x/net v0.0.0-20201021035429-f5854403a974 // indirect
	golang.org/x/sys v0.0.0-20210119212857-b64e53b001e4 // indirect
	golang.org/x/text v0.3.3 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	google.golang.org/grpc v1.49.0 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
)
