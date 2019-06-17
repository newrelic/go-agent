package testapp

import (
	"context"
	"encoding/json"
	"io"

	"google.golang.org/grpc/metadata"
)

// Server is a gRPC server.
type Server struct{}

// DoUnaryUnary is a unary request, unary response method.
func (s *Server) DoUnaryUnary(ctx context.Context, msg *Message) (*Message, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	js, _ := json.Marshal(md)
	return &Message{Text: string(js)}, nil
}

// DoUnaryStream is a unary request, stream response method.
func (s *Server) DoUnaryStream(msg *Message, stream TestApplication_DoUnaryStreamServer) error {
	for i := 0; i < 3; i++ {
		if err := stream.Send(&Message{Text: "Hello from DoUnaryStream"}); nil != err {
			return err
		}
	}
	return nil
}

// DoStreamUnary is a stream request, unary response method.
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

// DoStreamStream is a stream request, stream response method.
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
