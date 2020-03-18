# com_newrelic_trace_v1

To generate the `v1.pb.go` code, I ran the following from the `internal`
package:

```
protoc -I com_newrelic_trace_v1/ com_newrelic_trace_v1/v1.proto --go_out=plugins=grpc:com_newrelic_trace_v1
```

## When you regenerate the file 
Once you have generated the code, you will need to add a build tag to
the file:
 
 ```go
// +build go1.9
```

This is because the GRPC/Protocol Buffer libraries only support Go 1.9 and
above.