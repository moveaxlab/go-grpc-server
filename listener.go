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

// requestTimesMonitor is a global histogram used by gRPC endpoints
var requestTimesMonitor *prometheus.HistogramVec

var (
	requestCounter *prometheus.CounterVec
	failureCounter *prometheus.CounterVec
	errorCounter   *prometheus.CounterVec
)

type listener struct {
	server              *grpc.Server
	healthcheck         *health.Server
	requestTimesMonitor *prometheus.HistogramVec
	requestCounter      *prometheus.CounterVec
	failureCounter      *prometheus.CounterVec
	errorCounter        *prometheus.CounterVec
	listener            net.Listener
	port                int
}

func NewGrpcServer(port int, interceptors ...grpc.UnaryServerInterceptor) GrpcServer {
	grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(interceptors...), grpc.MaxHeaderListSize(8*1024*1024))

	requestTimesMonitor = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Subsystem: "grpc",
		Name:      "request_time_ms",
		Help:      "Time to serve gRPC requests in milliseconds",
		Buckets:   prometheus.ExponentialBuckets(16, 2, 10),
	}, []string{"endpoint"})

	requestCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "grpc",
		Name:      "request_count_total",
		Help:      "Counter for received gRPC requests",
	}, []string{"endpoint"})

	failureCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "grpc",
		Name:      "request_failure_count_total",
		Help:      "Counter for failed gRPC requests",
	}, []string{"endpoint"})

	errorCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "grpc",
		Name:      "request_error_count_total",
		Help:      "Counter for failed gRPC requests with internal errors",
	}, []string{"endpoint"})

	healthcheck := health.NewServer()
	healthgrpc.RegisterHealthServer(grpcServer, healthcheck)

	return &listener{
		server:              grpcServer,
		requestTimesMonitor: requestTimesMonitor,
		requestCounter:      requestCounter,
		failureCounter:      failureCounter,
		errorCounter:        errorCounter,
		port:                port,
	}
}

func (l *listener) GetServer() *grpc.Server {
	return l.server
}

func (l *listener) GetMetrics() []prometheus.Collector {
	return []prometheus.Collector{l.requestTimesMonitor, l.requestCounter, l.errorCounter, l.failureCounter}
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
