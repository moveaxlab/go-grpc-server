package grpc_server

import (
	"context"
	"fmt"
	"testing"

	"github.com/moveaxlab/go-grpc-server/internal"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
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

	t.Run("metric interceptor redacts the logged request on error", func(t *testing.T) {
		hook := logrustest.NewGlobal()
		defer hook.Reset()

		interceptor := NewMetricsInterceptor()
		info := &grpc.UnaryServerInfo{FullMethod: "/internal.TestService/Endpoint"}
		req := &internal.SensitiveInput{Username: "alice", Password: "hunter2"}
		handler := func(context.Context, interface{}) (interface{}, error) {
			return nil, fmt.Errorf("boom")
		}

		_, err := interceptor(context.Background(), req, info, handler)

		assert.NotNil(t, err)
		logged := loggedRequest(t, hook)
		assert.Equal(t, "alice", logged["username"])
		assert.Equal(t, redactedPlaceholder, logged["password"])
	})
}
