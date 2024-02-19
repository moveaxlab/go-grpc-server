package grpc_server

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func StatusInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	resp, err = handler(ctx, req)

	if err == nil {
		return resp, nil
	}

	st := status.Convert(err)

	if st.Code() == codes.Unknown {
		return nil, status.New(codes.Internal, st.Message()).Err()
	} else {
		return nil, st.Err()
	}
}
