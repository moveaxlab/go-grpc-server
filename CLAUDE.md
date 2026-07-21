# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`go-grpc-server` is a small Go library (not a standalone service) that provides reusable building
blocks for gRPC servers built by moveax: server start/stop lifecycle, health checking, Prometheus
metrics, and a set of `grpc.UnaryServerInterceptor`s for error handling, panic recovery, request
validation, and metrics collection. Consumers `go get` this module and wire the pieces into their
own gRPC server.

## Commands

Tooling versions (Go 1.20, protoc, protoc-gen-go, protoc-gen-go-grpc, mockery v2.38.0) are pinned in
`.mise.toml` and installed via `mise install`.

- Run all tests: `go test ./...`
- Run a single test: `go test ./... -run TestName`
- Vet: `go vet ./...`
- Format check (must be clean, this is enforced in CI): `gofmt -l .`
- Regenerate protobuf code from `internal/server.proto`: `mise run generate`
- Regenerate mocks (config in `.mockery.yaml`): `mockery`

CI (`.github/workflows/lint.yml`) runs on push/PR to `main` and does exactly: `go vet ./...`,
`gofmt -l .` (must produce no output), and `go test ./...`.

## Architecture

All public code lives in the root `grpc_server` package (module root, package name `grpc_server`,
imported as `github.com/moveaxlab/go-grpc-server`). The `internal/` package holds a test-only gRPC
service (`server.proto` → generated `server.pb.go` / `server_grpc.pb.go` / mocked
`mock_TestServiceServer.go` via mockery) used purely as a fixture for the interceptor tests — it is
not part of the public API.

Key files and their responsibility:

- `listener.go` — `GrpcServer` interface and its `listener` implementation. `NewGrpcServer(port, interceptors...)`
  builds the `*grpc.Server` with a chained unary interceptor list, registers the standard gRPC health
  service, and exposes `Start()`/`Stop()`/`GetServer()`/`GetMetrics()`. `Start()` listens and serves in
  a goroutine and marks the health check as `SERVING`; `Stop()` flips health to `NOT_SERVING`, does a
  `GracefulStop()`, sleeps 1s, then force `Stop()`s.
- `status.go` — `StatusInterceptor`: normalizes any error with gRPC code `Unknown` to `Internal`.
  Intended as the *first* interceptor in the chain.
- `metrics.go` — `NewMetricsInterceptor(excluded ...string)`: builds/reuses package-level Prometheus
  collectors (`grpc_request_time_ms` histogram, `grpc_request_count_total` and
  `grpc_request_error_count_total` counters, labeled by endpoint) and returns an interceptor that
  records them, skipping endpoints passed in `excluded`.
- `validation.go` — `ValidationInterceptor`: if the request implements `Validate(bool) error` (the
  `protoc-gen-validate` convention), runs validation and converts failures to `InvalidArgument`.
- `error.go` — `ApplicationError` interface (`error` + `GRPCStatus() *status.Status` +
  `Trailer() metadata.MD`) that application code can implement for domain errors.
  `NewErrorInterceptor()` unwraps `ApplicationError`s with `errors.As`, sets gRPC trailer metadata,
  converts them to their declared gRPC status, and increments the
  `grpc_request_application_error_count_total` counter (labeled by endpoint).
- `recover.go` — `RecoverInterceptor`: recovers panics from downstream handlers, logs request +
  stack trace via logrus, and turns the panic into a returned `error` instead of crashing the process.
  Intended as the *last* interceptor in the chain.

These interceptors are designed to be chained in a specific order (see README for the full rationale):
`StatusInterceptor` → `NewMetricsInterceptor(...)` → `ValidationInterceptor` → `NewErrorInterceptor()` →
`RecoverInterceptor`. Each interceptor calls `handler(ctx, req)` and inspects/transforms the
resulting `(resp, err)`, so ordering determines what each subsequent interceptor sees.

Several Prometheus collectors (`requestTimesMonitor`, `requestCounter`, `errorCounter`,
`applicationErrorCounter`) are package-level singletons lazily initialized inside their respective
interceptor constructors (`NewMetricsInterceptor`, `NewErrorInterceptor`) — calling a constructor
twice reuses the existing collector rather than re-registering it. `listener.GetMetrics()` returns
whichever of these have been initialized, so metrics only appear if the corresponding interceptor
was constructed.

## Testing conventions

Interceptor tests spin up a real in-memory gRPC server over `bufconn` rather than mocking at the
interceptor level. `base_test.go` provides `setupTestServer(t, interceptors...)`, which starts a
`grpc.Server` wired with the given interceptor chain and the generated `internal.TestServiceServer`
backed by `internal.MockTestServiceServer` (mockery-generated, expecter-style: `mockServer.EXPECT()...`),
and returns a real `internal.TestServiceClient` connected over an in-memory `bufconn` listener plus a
`cleanup func()` to gracefully stop the server and close the connection. Tests configure mock
expectations on `mockServer`, make real RPCs through `client`, and assert on the response/error/status
returned to the client — this exercises the interceptor exactly as it would run in production request
handling.
