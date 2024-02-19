package grpc_server

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type ApplicationError interface {
	GRPCStatus() *status.Status
	Trailer() metadata.MD
}

func ErrorInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	resp, err = handler(ctx, req)

	if err == nil {
		return resp, nil
	}

	if applicationError, ok := err.(ApplicationError); ok {
		err = grpc.SetTrailer(ctx, applicationError.Trailer())
		if err != nil {
			panic(fmt.Errorf("failed to encode error info: %w", err))
		}
		return nil, applicationError.GRPCStatus().Err()
	}

	if errorCounter != nil {
		errorCounter.With(prometheus.Labels{"endpoint": info.FullMethod}).Inc()
	}

	return nil, err
}
