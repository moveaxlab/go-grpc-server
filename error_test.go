package grpc_server

import (
	"context"
	"net"
	"testing"

	"github.com/moveaxlab/go-grpc-server/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestErrorInterceptor(t *testing.T) {
	server := grpc.NewServer(grpc.UnaryInterceptor(ErrorInterceptor))
	listener := bufconn.Listen(1024 * 1024)
	cc, err := grpc.DialContext(
		context.Background(),
		"",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.Dial()
		}),
	)
	assert.Nil(t, err)

	mockServer := &internal.MockTestServiceServer{}

	internal.RegisterTestServiceServer(server, mockServer)

	go func() {
		err := server.Serve(listener)
		assert.Nil(t, err)
	}()

	defer func() {
		server.GracefulStop()
		err := cc.Close()
		assert.Nil(t, err)
	}()

	client := internal.NewTestServiceClient(cc)

	ctx := context.Background()

	mockServer.On("Endpoint", mock.Anything, mock.Anything).Return(&internal.Output{Value: "World"}, nil)

	res, err := client.Endpoint(ctx, &internal.Input{Value: "Hello"})

	assert.Nil(t, err)

	assert.Equal(t, "World", res.Value)
}
