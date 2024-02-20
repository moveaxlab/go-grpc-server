package grpc_server

import (
	"context"
	"fmt"
	"testing"

	"github.com/moveaxlab/go-grpc-server/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func TestStatusInterceptor(t *testing.T) {
	t.Run("works when there is no error", func(t *testing.T) {
		client, mockServer, cleanup := setupTestServer(t, StatusInterceptor)
		defer cleanup()

		ctx := context.Background()

		mockServer.On("Endpoint", mock.Anything, mock.Anything).Return(&internal.Output{Value: "World"}, nil)

		res, err := client.Endpoint(ctx, &internal.Input{Value: "Hello"})

		assert.Nil(t, err)

		assert.Equal(t, "World", res.Value)
	})

	t.Run("does nothing to application errors", func(t *testing.T) {
		client, mockServer, cleanup := setupTestServer(t, StatusInterceptor, NewErrorInterceptor())
		defer cleanup()

		ctx := context.Background()

		applicationError := &testApplicationError{
			message:  "Failed",
			grpcCode: codes.InvalidArgument,
			code:     "APPLICATION_ERROR",
		}

		mockServer.On("Endpoint", mock.Anything, mock.Anything).Return(nil, applicationError)

		var md metadata.MD

		_, err := client.Endpoint(ctx, &internal.Input{Value: "Hello"}, grpc.Trailer(&md))

		assert.NotNil(t, err)
		grpcErr, ok := err.(GRPCStatus)
		assert.True(t, ok)
		assert.Equal(t, "Failed", grpcErr.GRPCStatus().Message())
		assert.Equal(t, codes.InvalidArgument, grpcErr.GRPCStatus().Code())
		assert.Equal(t, []string{"APPLICATION_ERROR"}, md["code"])
	})

	t.Run("wraps other errors", func(t *testing.T) {
		client, mockServer, cleanup := setupTestServer(t, StatusInterceptor)
		defer cleanup()

		ctx := context.Background()

		mockServer.On("Endpoint", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("random error"))

		var md metadata.MD

		_, err := client.Endpoint(ctx, &internal.Input{Value: "Hello"}, grpc.Trailer(&md))

		assert.NotNil(t, err)
		grpcErr, ok := err.(GRPCStatus)
		assert.True(t, ok)
		assert.Equal(t, "random error", grpcErr.GRPCStatus().Message())
		assert.Equal(t, codes.Internal, grpcErr.GRPCStatus().Code())
		assert.Nil(t, md["code"])
	})
}
