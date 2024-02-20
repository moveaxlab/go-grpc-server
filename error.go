package grpc_server

import (
	"context"
	"errors"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type ApplicationError interface {
	error
	GRPCStatus() *status.Status
	Trailer() metadata.MD
}

// NewErrorInterceptor creates an interceptor that serializes application errors.
//
// Adding this interceptor adds a prometheus metric that counts application errors.
//
// Your application code should return errors implementing the ApplicationError
// interface.
func NewErrorInterceptor() grpc.UnaryServerInterceptor {
	if applicationErrorCounter == nil {
		applicationErrorCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "grpc",
			Name:      "request_application_error_count_total",
			Help:      "Counter for failed gRPC requests with application errors",
		}, []string{"endpoint"})
	}

	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		resp, err = handler(ctx, req)

		if err == nil {
			return resp, nil
		}

		var applicationError ApplicationError

		if errors.As(err, &applicationError) {
			applicationErrorCounter.With(prometheus.Labels{"endpoint": info.FullMethod}).Inc()

			err = grpc.SetTrailer(ctx, applicationError.Trailer())
			if err != nil {
				panic(fmt.Errorf("failed to encode error info: %w", err))
			}

			return nil, applicationError.GRPCStatus().Err()
		}

		return nil, err
	}
}
