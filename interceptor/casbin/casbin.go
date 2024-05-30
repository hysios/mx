package casbin

import (
	"context"

	"github.com/casbin/casbin/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Enforcer = casbin.SyncedEnforcer

type EnforceFunc func(enforcer *Enforcer, ctx context.Context, method string) (bool, error)

func UnaryClientInterceptor(enforcer *Enforcer, en EnforceFunc) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if en != nil {
			ok, err := en(enforcer, ctx, method)
			if err != nil {
				return err
			}
			if !ok {
				return status.Errorf(codes.PermissionDenied, "permission denied")
			}
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func NewEnforcer(params ...interface{}) (*Enforcer, error) {
	return casbin.NewSyncedEnforcer(params...)
}

func StreamClientInterceptor(enforcer *Enforcer, en EnforceFunc) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		if en != nil {
			ok, err := en(enforcer, ctx, method)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, status.Errorf(codes.PermissionDenied, "permission denied")
			}
		}

		return streamer(ctx, desc, cc, method, opts...)
	}
}
