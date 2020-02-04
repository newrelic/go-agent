# com_newrelic_trace_v1

To generate the `v1.pb.go` code, I ran the following from the `internal`
package:

```
protoc -I com_newrelic_trace_v1/ com_newrelic_trace_v1/v1.proto --go_out=plugins=grpc:com_newrelic_trace_v1
```
