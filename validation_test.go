package grpc_server

import (
	"context"
	"testing"

	"github.com/moveaxlab/go-grpc-server/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
)

func TestValidation(t *testing.T) {
	t.Run("returns a validation error if input is invalid", func(t *testing.T) {
		client, _, cleanup := setupTestServer(t, ValidationInterceptor)
		defer cleanup()

		ctx := context.Background()

		_, err := client.Endpoint(ctx, &internal.Input{Value: "Hel"})

		assert.NotNil(t, err)
		grpcErr, ok := err.(GRPCStatus)
		assert.True(t, ok)
		assert.Equal(t, "value is too short", grpcErr.GRPCStatus().Message())
		assert.Equal(t, codes.InvalidArgument, grpcErr.GRPCStatus().Code())
	})

	t.Run("everything goes fine if input is valid", func(t *testing.T) {
		client, mockServer, cleanup := setupTestServer(t, ValidationInterceptor)
		defer cleanup()

		ctx := context.Background()

		mockServer.On("Endpoint", mock.Anything, mock.Anything).Return(&internal.Output{Value: "World"}, nil)

		res, err := client.Endpoint(ctx, &internal.Input{Value: "Helloooo"})

		assert.Nil(t, err)
		assert.Equal(t, "World", res.Value)
	})
}
