package grpc_server

import (
	"context"
	"fmt"
	"runtime/debug"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func RecoverInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp interface{}, err error) {
	return func() (finalResponse interface{}, finalError error) {
		defer func() {
			recoveredErr := recover()

			if recoveredErr != nil {
				log.
					WithField("request", req).
					WithField("stack_trace", string(debug.Stack())).
					Errorf("recovered a panic on %s: %v", info.FullMethod, recoveredErr)

				if actualError, recoveredAnError := recoveredErr.(error); recoveredAnError {
					finalError = fmt.Errorf("%s panicked: %w", info.FullMethod, actualError)
				} else {
					finalError = fmt.Errorf("%s panicked: %v", info.FullMethod, recoveredErr)
				}
			}
		}()

		return handler(ctx, req)
	}()
}
