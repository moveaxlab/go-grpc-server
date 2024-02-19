package grpc_server

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func NewSecurityInterceptor(fn func(context.Context, metadata.MD) context.Context) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		md, hasMetadata := metadata.FromIncomingContext(ctx)
		if !hasMetadata {
			md = metadata.MD{}
		}

		ctx = fn(ctx, md)

		return handler(ctx, req)
	}
}
