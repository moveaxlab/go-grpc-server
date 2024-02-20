package grpc_server

import (
	"context"
	"fmt"
	"testing"

	"github.com/moveaxlab/go-grpc-server/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRecover(t *testing.T) {
	t.Run("server crashes without recover interceptor", func(t *testing.T) {
		client, mockServer, cleanup := setupTestServer(t, RecoverInterceptor)
		defer cleanup()

		ctx := context.Background()

		mockServer.On("Endpoint", mock.Anything, mock.Anything).Run(func(_ mock.Arguments) {
			panic(fmt.Errorf("panic"))
		})

		_, err := client.Endpoint(ctx, &internal.Input{Value: "Hello"})

		assert.NotNil(t, err)
	})
}
