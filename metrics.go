package grpc_server

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func NewMetricsInterceptor(excluded ...string) grpc.UnaryServerInterceptor {
	ignoredEndpoints := make(map[string]bool)
	for _, endpoint := range excluded {
		ignoredEndpoints[endpoint] = true
	}

	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		if ignoredEndpoints[info.FullMethod] {
			return handler(ctx, req)
		}

		start := time.Now()

		requestCounter.With(prometheus.Labels{"endpoint": info.FullMethod}).Inc()

		defer func() {
			requestTimesMonitor.
				With(prometheus.Labels{"endpoint": info.FullMethod}).
				Observe(float64(time.Now().Sub(start) / time.Millisecond))
		}()

		resp, err = handler(ctx, req)

		if err == nil {
			return resp, nil
		}

		failureCounter.With(prometheus.Labels{"endpoint": info.FullMethod}).Inc()

		log.
			WithField("request", req).
			Errorf("request failed on %s: %v", info.FullMethod, err)

		return nil, err
	}
}
