package grpc_server

import (
	"context"

	"github.com/getsentry/sentry-go"
	"google.golang.org/grpc"
)

func NewSentryInterceptor(getUserId func(ctx context.Context) (string, bool)) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		resp, err = handler(ctx, req)
		if err == nil {
			return resp, nil
		}

		// capture internal errors on sentry
		sentry.WithScope(func(scope *sentry.Scope) {
			if userId, hasUser := getUserId(ctx); hasUser {
				scope.SetUser(sentry.User{
					ID: userId,
				})
			}

			scope.SetExtras(map[string]interface{}{
				"endpoint": info.FullMethod,
				"request":  req,
			})

			sentry.CaptureException(err)
		})

		return nil, err
	}
}
