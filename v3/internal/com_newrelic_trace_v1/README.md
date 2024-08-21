# com_newrelic_trace_v1

To generate the `v1.pb.go` code, run the following from the top level
`github.com/newrelic/go-agent/v3` package:

```
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative v3/internal/com_newrelic_trace_v1/v1.proto
```

Be mindful which version of `protoc-gen-go` you are using. Upgrade
`protoc-gen-go` to the latest with:

```
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

https://grpc.io/docs/languages/go/quickstart/
