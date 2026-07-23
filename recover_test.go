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

	t.Run("recover interceptor redacts the logged request", func(t *testing.T) {
		hook := logrustest.NewGlobal()
		defer hook.Reset()

		info := &grpc.UnaryServerInfo{FullMethod: "/internal.TestService/Endpoint"}
		req := &internal.SensitiveInput{Username: "alice", Password: "hunter2"}
		handler := func(context.Context, interface{}) (interface{}, error) {
			panic(fmt.Errorf("boom"))
		}

		_, err := RecoverInterceptor(context.Background(), req, info, handler)

		assert.NotNil(t, err)
		logged := loggedRequest(t, hook)
		assert.Equal(t, "alice", logged["username"])
		assert.Equal(t, redactedPlaceholder, logged["password"])
	})
}
