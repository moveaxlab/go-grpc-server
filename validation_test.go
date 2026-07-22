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
	"google.golang.org/grpc/codes"
)

type validatingSensitiveInput struct {
	*internal.SensitiveInput
}

func (v *validatingSensitiveInput) Validate(bool) error {
	return fmt.Errorf("invalid")
}

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

	t.Run("validation interceptor redacts the logged request", func(t *testing.T) {
		hook := logrustest.NewGlobal()
		defer hook.Reset()

		info := &grpc.UnaryServerInfo{FullMethod: "/internal.TestService/Endpoint"}
		req := &validatingSensitiveInput{
			SensitiveInput: &internal.SensitiveInput{Username: "alice", Password: "hunter2"},
		}
		handler := func(context.Context, interface{}) (interface{}, error) { return nil, nil }

		_, err := ValidationInterceptor(context.Background(), req, info, handler)

		assert.NotNil(t, err)
		logged := loggedRequest(t, hook)
		assert.Equal(t, "alice", logged["username"])
		assert.Equal(t, redactedPlaceholder, logged["password"])
	})
}
