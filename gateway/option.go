package gateway

import (
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/hysios/mx"
	"github.com/hysios/mx/logger"
	"github.com/hysios/mx/provisioning"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type GatewayOption struct {
	Logger                   *zap.Logger
	Middlewares              []mx.Middleware
	MiddlewareMakes          []MiddlewareMaker
	ClientUnaryInterceptors  []grpc.UnaryClientInterceptor
	ClientStreamInterceptors []grpc.StreamClientInterceptor
	MuxOptions               []runtime.ServeMuxOption
	CustomMetricsPath        string
	CustomDebugPath          string
	CustomMetricsHander      http.Handler
}

type MiddlewareMaker func(gateway *mx.Gateway) mx.Middleware

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

func WithMiddlewareMaker(mws ...MiddlewareMaker) GatewayOptFunc {
	return func(o *GatewayOption) error {
		o.MiddlewareMakes = mws
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

func WithMuxOptions(opts ...runtime.ServeMuxOption) GatewayOptFunc {
	return func(o *GatewayOption) error {
		o.MuxOptions = opts
		return nil
	}
}

// WithCustomMetricsPath sets the path for the metrics handler.
func WithCustomMetricsPath(path string) GatewayOptFunc {
	return func(o *GatewayOption) error {
		o.CustomMetricsPath = path
		return nil
	}
}

// WithCustomMetricsHandler sets the handler for the metrics handler.
func WithCustomMetricsHandler(path string, handler http.Handler) GatewayOptFunc {
	return func(o *GatewayOption) error {
		o.CustomMetricsPath = path
		o.CustomMetricsHander = handler
		return nil
	}
}

// WithCustomDebugPath sets the path for the debug handler.
func WithCustomDebugPath(path string) GatewayOptFunc {
	return func(o *GatewayOption) error {
		o.CustomDebugPath = path
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
