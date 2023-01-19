package mx

import (
	"context"

	"google.golang.org/grpc"
)

type Service interface {
	ServiceName() string
	Register(ctx context.Context, gw *Gateway) error
	// Invoke(ctx context.Context, method string, args, reply interface{}) error
}

type Initer interface {
	Init() error
}

type nopConn struct {
}

func (*nopConn) Invoke(ctx context.Context, method string, args, reply interface{}, opts ...grpc.CallOption) error {
	return nil
}

func (*nopConn) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, nil
}

func NopConn() grpc.ClientConnInterface {
	return &nopConn{}
}
