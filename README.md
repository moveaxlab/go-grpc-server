# Go gRPC server utilities

This repo contains utilities to build gRPC server.
These include useful interceptors, health checks, metrics,
and functions to start and stop the gRPC server.

## Installation

```bash
go get github.com/moveaxlab/go-grpc-server
```

## Create the server

This package provides utilities to create, start, and stop gRPC servers.

```go
package main

import (
    "fmt"
    "github.com/moveaxlab/go-grpc-server"
)

func main() {
    server := grpc_server.NewGrpcServer(40051)
    wg := &sync.WaitGroup{}
    channel := make(chan os.Signal, 1)

    // register your gRPC services
    var myService mypackage.MyServiceServer
    // initialize the myService variable
    mypackage.RegisterMyServiceServer(server.GetServer(), myService)

    // start the server
    server.Start()

    // stop the server gracefully
    signal.Notify(channel, syscall.SIGINT)
	signal.Notify(channel, syscall.SIGTERM)

    wg.Add(1)
    go func() {
        defer wg.Done()
        select {
		case <-channel:
            err := server.Stop()
            if err != nil {
                fmt.Printf("server stop returned an error: %v", err)
            }
		}

    }()
    wg.Wait()
}
```

Implementing a new gRPC service requires the following steps:

1. generate code from your proto files, using `protoc`, `protoc-gen-go`, and `protoc-gen-go-grpc`
2. implement the service with a struct that embeds the `mypackage.MyServiceServer` interface
3. register it before the gRPC server start with `mypackage.RegisterMyServiceServer`,
   passing it the result of `server.GetServer()` as first argument

The server registers automatically the [standard health service](https://grpc.io/docs/guides/health-checking/),
and sets its status to running as soon as the gRPC server is started.

### Metrics

The gRPC server provides metrics with the `GetMetrics()` method,
which returns a list of `github.com/prometheus/client_golang/prometheus.Collector`.

Using interceptors will enable different metrics.

## Interceptors

The gRPC server constructor function accepts a list of `google.golang.org/grpc.UnaryServerInterceptor`
that will be registered in the order they are passed.

We provide a few useful interceptors.

### Recovering from panics

The `grpc_server.RecoverInterceptor` recovers from panics downstream in your application.
Add it at the end of your interceptor list, and you can `panic` as much as you want
inside your application code.

The recovered errors will be returned as the second result from the `handler` function
in upstream interceptors.

### Handling application errors

This package provides the `grpc_server.ApplicationError` interface that can be implemented
by your application errors. This interface requires you to implement the `GRPCStatus()`
method, that converts the error to a `google.golang.org/grpc/status.Status` object,
and the `Trailer()` method that returns the trailing metadata that must be returned to the caller.

Once you have defined your application errors, call the `grpc_server.NewErrorInterceptor()`
to initialize the interceptor, and add it to your gRPC server.

Initializing this interceptor adds the `grpc_request_application_error_count_total` prometheus metric
to the gRPC server, which counts application errors.
