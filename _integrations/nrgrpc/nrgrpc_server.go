package nrgrpc

import (
	"context"

	newrelic "github.com/newrelic/go-agent"
	"google.golang.org/grpc"
)

func UnaryServerInterceptor(app newrelic.Application) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		txn := app.StartTransaction(info.FullMethod, nil, nil)
		defer txn.End()
		// TODO: Read incoming DT headers
		// TODO: Set proper attributes

		ctx = newrelic.NewContext(ctx, txn)
		resp, err = handler(ctx, req)
		if err != nil {
			// TODO: NoticeError
		}
		// TODO: Save response code
		return
	}
}
