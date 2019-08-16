package main

import (
	"context"
	"fmt"
	"log"
	"time"

	proto "github.com/micro/examples/helloworld/proto"
	"github.com/micro/go-micro"
)

func subEv(ctx context.Context, msg *proto.HelloRequest) error {
	fmt.Println("Message received from", msg.GetName())
	return nil
}

func publish(s micro.Service) {
	c := s.Client()

	for range time.NewTicker(time.Second).C {
		msg := c.NewMessage("example.topic.pubsub", &proto.HelloRequest{Name: "Sally"})
		ctx := context.Background()
		fmt.Println("Sending message")
		if err := c.Publish(ctx, msg); nil != err {
			log.Fatal(err)
		}
	}
}

func main() {
	s := micro.NewService(
		micro.Name("go.micro.srv.pubsub"),
	)
	s.Init()

	go publish(s)

	micro.RegisterSubscriber("example.topic.pubsub", s.Server(), subEv)

	if err := s.Run(); err != nil {
		log.Fatal(err)
	}
}
