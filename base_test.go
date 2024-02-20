package grpc_server

import (
	"context"
	"net"
	"testing"

	"github.com/moveaxlab/go-grpc-server/internal"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func setupTestServer(
	t *testing.T,
	interceptors ...grpc.UnaryServerInterceptor,
) (
	client internal.TestServiceClient,
	mockServer *internal.MockTestServiceServer,
	cleanup func(),
) {
	server := grpc.NewServer(grpc.ChainUnaryInterceptor(interceptors...))
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

	mockServer = &internal.MockTestServiceServer{}

	internal.RegisterTestServiceServer(server, mockServer)

	go func() {
		err := server.Serve(listener)
		assert.Nil(t, err)
	}()

	cleanup = func() {
		server.GracefulStop()
		err := cc.Close()
		assert.Nil(t, err)
	}

	client = internal.NewTestServiceClient(cc)

	return client, mockServer, cleanup
}
