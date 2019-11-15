# Testing gRPC Application

This directory contains a testing application for validating the New Relic gRPC
integration.  The code in `testapp.pb.go` is generated using the following
command (to be run from the `_integrations/nrgrpc` directory).  This command
should be rerun every time the `testapp.proto` file has changed for any reason.

```bash
$ protoc -I testapp/ testapp/testapp.proto --go_out=plugins=grpc:testapp
```

To install required dependencies:

```bash
go get -u google.golang.org/grpc
go get -u github.com/golang/protobuf/protoc-gen-go
```
