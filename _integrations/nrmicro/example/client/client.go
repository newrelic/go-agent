package main

import (
	"context"
	"fmt"

	proto "github.com/micro/examples/helloworld/proto"
	"github.com/micro/go-micro"
)

func main() {
	service := micro.NewService()
	c := proto.NewGreeterService("greeter", service.Client())

	rsp, err := c.Hello(context.Background(), &proto.HelloRequest{
		Name: "John",
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(rsp.Greeting)
}
