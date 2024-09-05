# com_newrelic_trace_v1

To generate the `v1.pb.go` code, run the following from the top level
`github.com/newrelic/go-agent` package:

```
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    v3/internal/com_newrelic_trace_v1/v1.proto
```

Be mindful which version of `protoc-gen-go` and `protoc-gen-go-grpc` you are using.
Upgrade both of these tools to the latest with:

```
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

## When you regenerate the file

Once you have generated the code, you will need to add a build tag to the file:

 ```go
// +build go1.9
```

This is because the gRPC/Protocol Buffer libraries only support Go 1.9 and
above.
