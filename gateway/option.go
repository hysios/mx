package gateway

import (
	"github.com/hysios/mx"
	"github.com/hysios/mx/logger"
	"github.com/hysios/mx/provisioning"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type GatewayOption struct {
	Logger                   *zap.Logger
	Middlewares              []mx.Middleware
	ClientUnaryInterceptors  []grpc.UnaryClientInterceptor
	ClientStreamInterceptors []grpc.StreamClientInterceptor
}

func WithLogger(logger *zap.Logger) GatewayOptFunc {
	return func(o *GatewayOption) error {
		o.Logger = logger
		return nil
	}
}

func WithMiddleware(mws ...mx.Middleware) GatewayOptFunc {
	return func(o *GatewayOption) error {
		o.Middlewares = mws
		return nil
	}
}

func WithClientUnaryInterceptor(interceptors ...grpc.UnaryClientInterceptor) GatewayOptFunc {
	return func(o *GatewayOption) error {
		o.ClientUnaryInterceptors = interceptors
		return nil
	}
}

func WithClientStreamInterceptor(interceptors ...grpc.StreamClientInterceptor) GatewayOptFunc {
	return func(o *GatewayOption) error {
		o.ClientStreamInterceptors = interceptors
		return nil
	}
}

func evaluteOption(optfns ...GatewayOptFunc) *GatewayOption {
	var opts = &GatewayOption{}
	provisioning.Init(opts)

	for _, fn := range optfns {
		if err := fn(opts); err != nil {
			panic(err)
		}
	}

	return opts
}

func init() {
	provisioning.Provision(func(opts *GatewayOption) {
		if opts.Logger == nil {
			opts.Logger = logger.Logger
		}
	})
}
