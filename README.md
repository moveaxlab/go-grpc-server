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
	"os"
	"os/signal"
	"sync"
	"syscall"

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
The suggested order of registration is the following:

- `StatusInterceptor` handles non-application errors
- `NewMetricsInterceptor()` tracks prometheus metrics for your application
- `ValidationInterceptor` validates requests using `protoc-gen-validate`
- `NewErrorInterceptor()` handles application errors
- `RecoverInterceptor` recovers from panics occurring in the application

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

### Validating requests

If you are using [`protoc-gen-validate`](https://github.com/bufbuild/protoc-gen-validate)
on incoming requests, you can add the `grpc_server.ValidationInterceptor` to your gRPC server.
This interceptor will validate incoming requests, and return an error if validation does not pass.

### Collecting metrics

You can use the `grpc_server.NewMetricsInterceptor` function to create an interceptor
that collects metrics on requests you received using prometheus.
Initializing this interceptor will register the following metrics:

- `grpc_request_time_ms` tracks time taken by all requests
- `grpc_request_count_total` tracks the number of requests received
- `grpc_request_error_count_total` tracks the number of requests that failed for any reason

The function that creates the interceptor takes in input
a list of endpoints that will be ignored.
This can be used if you don't want to track metrics on certain endpoints,
e.g. for the health check endpoint.

### Unhandled errors

The `grpc_server.StatusInterceptor` adds the internal status code to responses
if the service returned an error and it was not handled by other interceptors.

This should be your first interceptor.

## Custom interceptors

You can implement custom interceptors that use your custom application logic.

This interceptor extracts user info from the request metadata,
and stores them in the request context:

```go
import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func SecurityInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp interface{}, err error) {
	md, hasMetadata := metadata.FromIncomingContext(ctx)

	if hasMetadata {
		ctx = AddUserToContext(ctx, md)
	}

	return handler(ctx, req)
}
```

This is what a Sentry interceptor would look like,
that sends non-application errors to Sentry and tracks user info:

```go
import (
	"context"

	"github.com/moveaxlab/go-grpc-server"
	"github.com/getsentry/sentry-go"
	"google.golang.org/grpc"
)

func SentryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp interface{}, err error) {
	resp, err = handler(ctx, req)

	// skip reporting application errors to sentry
	if err == nil || grpc_server.IsApplicationError(err) {
		return resp, nil
	}

	sentry.WithScope(func(scope *sentry.Scope) {
		// your custom logic to retrieve the user from the context
		if user, hasUser := GetUser(ctx); hasUser {
			scope.SetUser(sentry.User{
				ID: user.Id(),
			})
		}

		scope.SetExtras(map[string]interface{}{
			"endpoint": info.FullMethod,
			"request":  req,
		})

		sentry.CaptureException(err)
	})

	return nil, err
}
```

This interceptor should be registered after the `SecurityInterceptor` example given above.
