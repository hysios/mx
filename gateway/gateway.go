package gateway

import (
	"github.com/hysios/mx"
	_ "github.com/hysios/mx/internal/gateway"
	"github.com/hysios/mx/middleware"
)

// GatewayOptFunc is a function that configures a gateway
type GatewayOptFunc func(*GatewayOption) error

// New creates a new gateway with various options
// Example:
//
// Create a gateway with default options:
//
//	var gw = gw.New()
//
// Create a gateway with custom logger:
//
//	var gw = gw.New(
//		gw.WithLogger(logger),
//	)
//
// Create a gateway with custom middlewares:
//
//	var gw = gw.New(
//			gw.WithMiddleware(
//				   middleware.Recover(),
//				   middleware.Logger(),
//		   ),
//	 )
//
// Create a gateway with custom client interceptors:
//
//	var gw = gw.New(
//		gw.WithClientUnaryInterceptor(
//			grpc_opentracing.UnaryClientInterceptor(),
//		),
//		gw.WithClientStreamInterceptor(
//			grpc_opentracing.StreamClientInterceptor(),
//		),
//	)
func New(optfns ...GatewayOptFunc) *mx.Gateway {
	var (
		gw   = &mx.Gateway{}
		opts = evaluteOption(optfns...)
	)

	if opts.Logger != nil {
		gw.Logger = opts.Logger
	}

	gw.Use(middleware.Defaults...)

	if opts.Middlewares != nil {
		gw.Use(opts.Middlewares...)
	}

	if opts.MiddlewareMakes != nil {
		for _, fn := range opts.MiddlewareMakes {
			gw.Use(fn(gw))
		}
	}

	if opts.ClientUnaryInterceptors != nil {
		gw.AddClientUnaryInterceptor(opts.ClientUnaryInterceptors...)
	}

	if opts.ClientStreamInterceptors != nil {
		gw.AddClientStreamInterceptor(opts.ClientStreamInterceptors...)
	}

	if len(opts.MuxOptions) > 0 {
		gw.WithMuxOption(opts.MuxOptions...)
	}

	// gw.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	// 	_, _ = w.Write([]byte("hello world"))
	// })

	return gw
}
