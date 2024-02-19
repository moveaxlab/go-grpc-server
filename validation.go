package grpc_server

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ValidationInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp interface{}, err error) {
	if v, ok := req.(interface{ Validate(bool) error }); ok {
		validationError := v.Validate(false)

		if validationError != nil {
			log.
				WithField("request", req).
				Errorf("validation failed on %s: %v", info.FullMethod, validationError)

			if errorCounter != nil {
				errorCounter.With(prometheus.Labels{"endpoint": info.FullMethod}).Inc()
			}

			st := status.Convert(validationError)

			return nil, status.New(codes.InvalidArgument, st.Message()).Err()
		}
	}

	return handler(ctx, req)
}
