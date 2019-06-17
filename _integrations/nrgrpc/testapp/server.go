package testapp

import (
	"context"
	"encoding/json"
	"io"

	"google.golang.org/grpc/metadata"
)

type Server struct{}

func (s *Server) DoUnaryUnary(ctx context.Context, msg *Message) (*Message, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	js, _ := json.Marshal(md)
	return &Message{Text: string(js)}, nil
}

func (s *Server) DoUnaryStream(msg *Message, stream TestApplication_DoUnaryStreamServer) error {
	for i := 0; i < 3; i++ {
		if err := stream.Send(&Message{Text: "Hello from DoUnaryStream"}); nil != err {
			return err
		}
	}
	return nil
}

func (s *Server) DoStreamUnary(stream TestApplication_DoStreamUnaryServer) error {
	for {
		_, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&Message{Text: "Hello from DoStreamUnary"})
		} else if nil != err {
			return err
		}
	}
}

func (s *Server) DoStreamStream(stream TestApplication_DoStreamStreamServer) error {
	for {
		_, err := stream.Recv()
		if err == io.EOF {
			return nil
		} else if nil != err {
			return err
		}
		if err := stream.Send(&Message{Text: "Hello from DoStreamStream"}); nil != err {
			return err
		}
	}
}
