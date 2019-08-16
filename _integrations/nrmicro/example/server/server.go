package main

import (
	"context"
	"fmt"
	"log"

	proto "github.com/micro/examples/helloworld/proto"
	"github.com/micro/go-micro"
)

type Greeter struct{}

func (g *Greeter) Hello(ctx context.Context, req *proto.HelloRequest, rsp *proto.HelloResponse) error {
	name := req.GetName()
	fmt.Println("Request received from", name)
	rsp.Greeting = "Hello " + name
	return nil
}

func main() {
	service := micro.NewService(
		micro.Name("greeter"),
	)

	service.Init()

	proto.RegisterGreeterHandler(service.Server(), new(Greeter))

	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
