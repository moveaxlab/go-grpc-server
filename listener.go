package grpc_server

import (
	"fmt"
	"net"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
)

type GrpcServer interface {
	Start()
	GetServer() *grpc.Server
	GetMetrics() []prometheus.Collector
	Stop() error
}

var requestTimesMonitor *prometheus.HistogramVec

var (
	requestCounter          *prometheus.CounterVec
	errorCounter            *prometheus.CounterVec
	applicationErrorCounter *prometheus.CounterVec
)

type listener struct {
	server      *grpc.Server
	healthcheck *health.Server
	listener    net.Listener
	port        int
}

func NewGrpcServer(port int, interceptors ...grpc.UnaryServerInterceptor) GrpcServer {
	grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(interceptors...), grpc.MaxHeaderListSize(8*1024*1024))

	healthcheck := health.NewServer()
	healthgrpc.RegisterHealthServer(grpcServer, healthcheck)

	return &listener{
		server:      grpcServer,
		port:        port,
		healthcheck: healthcheck,
	}
}

func (l *listener) GetServer() *grpc.Server {
	return l.server
}

func (l *listener) GetMetrics() []prometheus.Collector {
	res := make([]prometheus.Collector, 0)
	if requestTimesMonitor != nil {
		res = append(res, requestTimesMonitor)
	}
	if requestCounter != nil {
		res = append(res, requestCounter)
	}
	if applicationErrorCounter != nil {
		res = append(res, applicationErrorCounter)
	}
	if errorCounter != nil {
		res = append(res, errorCounter)
	}
	return res
}

func (l *listener) Start() {
	go func() {
		grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", l.port))
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}

		l.listener = grpcListener

		l.healthcheck.SetServingStatus("", healthgrpc.HealthCheckResponse_SERVING)

		err = l.server.Serve(l.listener)
		if err != nil {
			log.Fatalf("gRPC server failed: %v", err)
		}
	}()

	log.Infof("gRPC server listening on port %d", l.port)
}

func (l *listener) Stop() error {
	l.healthcheck.SetServingStatus("", healthgrpc.HealthCheckResponse_NOT_SERVING)

	log.Debugf("stopping gRPC server gracefully...")
	l.server.GracefulStop()

	time.Sleep(1 * time.Second)

	log.Debugf("stopping gRPC server...")
	l.server.Stop()

	return nil
}
