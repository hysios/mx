package gateway

import (
	"net/http"

	"github.com/hysios/mx"
	"github.com/hysios/mx/logger"
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
		opts GatewayOption
	)

	for _, fn := range optfns {
		if err := fn(&opts); err != nil {
			panic(err)
		}
	}

	if opts.Logger != nil {
		gw.Logger = opts.Logger
	} else {
		gw.Logger = logger.Cli
	}

	if opts.Middlewares != nil {
		gw.Use(opts.Middlewares...)
	}

	gw.Use(middleware.Defaults...)

	if opts.ClientUnaryInterceptors != nil {
		gw.AddClientUnaryInterceptor(opts.ClientUnaryInterceptors...)
	}

	if opts.ClientStreamInterceptors != nil {
		gw.AddClientStreamInterceptor(opts.ClientStreamInterceptors...)
	}

	gw.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	})

	return gw
}
