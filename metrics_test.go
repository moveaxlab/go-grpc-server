package grpc_server

import (
	"context"
	"fmt"
	"testing"

	"github.com/moveaxlab/go-grpc-server/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMetrics(t *testing.T) {
	t.Run("metric interceptor works", func(t *testing.T) {
		client, mockServer, cleanup := setupTestServer(t, NewMetricsInterceptor())
		defer cleanup()

		ctx := context.Background()

		mockServer.On("Endpoint", mock.Anything, mock.Anything).Return(&internal.Output{Value: "World"}, nil)

		res, err := client.Endpoint(ctx, &internal.Input{Value: "Hello"})

		assert.Nil(t, err)
		assert.Equal(t, "World", res.Value)
	})

	t.Run("metric interceptor works on error too", func(t *testing.T) {
		client, mockServer, cleanup := setupTestServer(t, NewMetricsInterceptor())
		defer cleanup()

		ctx := context.Background()

		mockServer.On("Endpoint", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("random error"))

		_, err := client.Endpoint(ctx, &internal.Input{Value: "Hello"})

		assert.NotNil(t, err)
	})
}
