package mx

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

type RegisterOption struct {
	Method              string
	Conn                *grpc.ClientConn
	Impl                interface{}
	Client              interface{}
	ClientCtor          ClientCtor
	ServiceHandleClient ServiceHandleClient
	Handler             ConnServiceHandler
}

type (
	ClientCtor          any
	ServiceHandleClient any
)

type (
	RegisterOptFunc    func(*RegisterOption) error
	ConnServiceHandler func(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error
)

const (
	// DefaultServicePattern default service pattern
	ServiceMethodImpl    = "impl"
	ServiceMethodHandler = "handler"
	ServiceMethodClient  = "client"
	ServiceMethodConn    = "conn"
)

func WithConnString(connString string) RegisterOptFunc {
	return func(o *RegisterOption) (err error) {
		o.Conn, err = grpc.Dial(connString, grpc.WithInsecure())
		if err != nil {
			return err
		}
		return nil
	}
}

func WithConn(conn *grpc.ClientConn) RegisterOptFunc {
	return func(o *RegisterOption) error {
		o.Conn = conn
		return nil
	}
}

func WithServiceHandler(handler ConnServiceHandler) RegisterOptFunc {
	return func(o *RegisterOption) error {
		o.Method = ServiceMethodHandler
		o.Handler = handler
		return nil
	}
}
