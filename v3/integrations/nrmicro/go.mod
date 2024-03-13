module github.com/newrelic/go-agent/v3/integrations/nrmicro

// As of Dec 2019, the go-micro go.mod file uses 1.13:
// https://github.com/micro/go-micro/blob/master/go.mod
go 1.19

require (
	github.com/golang/protobuf v1.5.4
	github.com/micro/go-micro v1.8.0
	github.com/newrelic/go-agent/v3 v3.30.0
	google.golang.org/protobuf v1.33.0
)

require (
	github.com/armon/go-metrics v0.0.0-20190430140413-ec5e00d3c878 // indirect
	github.com/go-log/log v0.1.0 // indirect
	github.com/google/btree v1.0.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/consul/api v1.1.0 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.1 // indirect
	github.com/hashicorp/go-immutable-radix v1.1.0 // indirect
	github.com/hashicorp/go-msgpack v0.5.5 // indirect
	github.com/hashicorp/go-multierror v1.0.0 // indirect
	github.com/hashicorp/go-rootcerts v1.0.1 // indirect
	github.com/hashicorp/go-sockaddr v1.0.2 // indirect
	github.com/hashicorp/golang-lru v0.5.1 // indirect
	github.com/hashicorp/memberlist v0.1.4 // indirect
	github.com/hashicorp/serf v0.8.3 // indirect
	github.com/json-iterator/go v1.1.6 // indirect
	github.com/klauspost/compress v1.17.7 // indirect
	github.com/micro/cli v0.2.0 // indirect
	github.com/micro/mdns v0.1.1-0.20190729112526-ef68c9635478 // indirect
	github.com/miekg/dns v1.1.15 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/hashstructure v1.0.0 // indirect
	github.com/mitchellh/mapstructure v1.1.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/nats-io/nats.go v1.33.1 // indirect
	github.com/nats-io/nkeys v0.4.7 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pkg/errors v0.8.1 // indirect
	github.com/sean-/seed v0.0.0-20170313163322-e2103e2c3529 // indirect
	golang.org/x/crypto v0.21.0 // indirect
	golang.org/x/net v0.21.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240227224415-6ceb2ff114de // indirect
	google.golang.org/grpc v1.62.0 // indirect
	google.golang.org/grpc/examples v0.0.0-20240313164337-4f43d2e91d2d // indirect
)

replace github.com/newrelic/go-agent/v3 => ../..
