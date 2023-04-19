package mx

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"

	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/hysios/mx/discovery"
	"github.com/hysios/mx/logger"
	"github.com/hysios/mx/provisioning"
	"github.com/hysios/utils"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// Gateway grpc gateway
type Gateway struct {
	ApiPrefix string
	Logger    *zap.Logger // logger

	// middleware chain
	middlewares              []Middleware                   // middleware chain
	muxOptions               []runtime.ServeMuxOption       // grpc-gateway mux options
	gwmux                    *runtime.ServeMux              // grpc-gateway mux instance
	muxpool                  *MuxPool                       // mux pool
	serve                    *http.Server                   // http server
	prevAddr                 string                         // previous listen address
	discovery                *discovery.ServiceDiscovery    // service discovery registry
	routers                  []map[string]routeHandler      // custom routers
	notFounder               http.Handler                   // not found handler
	services                 utils.Map[string, Service]     // services
	ctx                      context.Context                // context
	closefn                  context.CancelFunc             // close function
	clientUnaryInterceptors  []grpc.UnaryClientInterceptor  // client unary interceptors
	clientStreamInterceptors []grpc.StreamClientInterceptor // client stream interceptors
	run                      runqueue
}

type routeMethod int

const (
	route_Prefix routeMethod = iota
	route_Route
)

type routeHandler struct {
	Path    string
	Method  routeMethod
	Handler http.Handler
}

type serviceConn struct {
	ServiceID string
	Conn      *grpc.ClientConn
}

type (
	ServiceHandler       func(ctx context.Context, mux *runtime.ServeMux, conn grpc.ClientConnInterface) error
	ServiceDirectHandler func(ctx context.Context, mux *runtime.ServeMux, impl any) error
)

// Middleware grpc gateway middleware
type Middleware func(http.Handler) http.Handler

func (gw *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	gw.serve.Handler.ServeHTTP(w, r)
}

// Use add middleware
func (gw *Gateway) Use(middlewares ...Middleware) {
	gw.middlewares = append(gw.middlewares, middlewares...)
}

func (gw *Gateway) HandleFunc(path string, handler func(http.ResponseWriter, *http.Request)) {
	// gw.routers = append(gw.routers, map[string]http.Handler{path: http.HandlerFunc(handler)})
	gw.routers = append(gw.routers, map[string]routeHandler{path: routeHandler{Path: path, Method: route_Route, Handler: http.HandlerFunc(handler)}})
}

func (gw *Gateway) NotFoundFunc(handler http.Handler) {
	gw.notFounder = handler
}

func (gw *Gateway) AddClientUnaryInterceptor(interceptors ...grpc.UnaryClientInterceptor) {
	gw.clientUnaryInterceptors = append(gw.clientUnaryInterceptors, interceptors...)
}

func (gw *Gateway) AddClientStreamInterceptor(interceptors ...grpc.StreamClientInterceptor) {
	gw.clientStreamInterceptors = append(gw.clientStreamInterceptors, interceptors...)
}

func (gw *Gateway) RegisterService(service Service) error {
	if _, ok := gw.services.Load(service.ServiceName()); ok {
		return fmt.Errorf("service %s already registered", service.ServiceName())
	}
	gw.services.Store(service.ServiceName(), service)

	return gw.run.call(Setup, func() {
		if err := service.Register(gw.ctx, gw); err != nil {
			panic(err)
		}
	})
}

func (gw *Gateway) GetService(name string) (Service, bool) {
	return gw.services.Load(name)
}

func (gw *Gateway) setup() error {
	gw.init()
	gw.gwmux = runtime.NewServeMux(
		gw.buildMuxOptions()...,
	)
	gw.serve = gw.createServer(gw.gwmux)
	gw.discovery.Discovery(gw.discoveryService)

	go gw.discovery.Start(gw.ctx)
	provisioning.Init(gw)

	gw.muxpool = NewMuxPool(gw.createMuxs(2)...)

	gw.run.do(Setup)
	return nil
}

func (gw *Gateway) createMux() *runtime.ServeMux {
	return runtime.NewServeMux(
		gw.buildMuxOptions()...,
	)
}

func (gw *Gateway) createMuxs(n int) []*runtime.ServeMux {
	muxs := make([]*runtime.ServeMux, n)
	for i := 0; i < n; i++ {
		muxs[i] = gw.createMux()
	}

	return muxs
}

func (gw *Gateway) Serve(addr string) error {
	gw.prevAddr = addr

	if err := gw.setup(); err != nil {
		return err
	}

	gw.Logger.Info("gateway start", zap.String("addr", addr))
	return http.ListenAndServe(addr, gw)
}

func (gw *Gateway) ServeTLS(addr string, certFile, keyFile string) error {
	gw.prevAddr = addr
	if err := gw.setup(); err != nil {
		return err
	}

	return http.ListenAndServeTLS(addr, certFile, keyFile, gw)
}

func (gw *Gateway) createServer(gwmux *runtime.ServeMux) *http.Server {

	// build router and initial middlewares
	r := gw.buildRouter()

	r.PathPrefix(gw.ApiPrefix).Handler(gw.gwmux)

	httpServer := &http.Server{
		// Addr:    gw.prevAddr,
		// Handler: gw.gwmux,
		Handler: r,
	}

	return httpServer
}

func (gw *Gateway) buildRouter() *mux.Router {
	r := mux.NewRouter()
	// use middlewares
	gw.addMetrics()
	gw.addPprof()
	gw.buildMiddlewares(r)
	gw.setupRouters(r)

	r.NotFoundHandler = gw.notFounder
	return r
}

func (gw *Gateway) buildMiddlewares(r *mux.Router) {
	for _, m := range gw.middlewares {
		r.Use(mux.MiddlewareFunc(m))
	}
}

func (gw *Gateway) setupRouters(r *mux.Router) {
	for _, router := range gw.routers {
		for path, handler := range router {
			switch handler.Method {
			case route_Route:
				r.Handle(path, handler.Handler)
			case route_Prefix:
				r.PathPrefix(path).Handler(handler.Handler)
			}
			// r.PathPrefix(path).Handler(handler)
		}
	}
}

func (gw *Gateway) addRouter(path string, handler http.Handler) {
	// gw.routers = append(gw.routers, map[string]http.Handler{path: handler})
	gw.routers = append(gw.routers, map[string]routeHandler{path: routeHandler{Path: path, Method: route_Route, Handler: handler}})
}

func (gw *Gateway) addPrefixRoute(path string, handler http.Handler) {
	gw.routers = append(gw.routers, map[string]routeHandler{path: routeHandler{Path: path, Method: route_Prefix, Handler: handler}})
}

func (gw *Gateway) addMetrics() {
	gw.addRouter("/metrics", promhttp.Handler())
}

// addPprof
func (gw *Gateway) addPprof() {
	gw.addPrefixRoute("/debug/pprof/", http.HandlerFunc(pprof.Index))
	gw.addPrefixRoute("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	gw.addPrefixRoute("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	gw.addPrefixRoute("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	gw.addPrefixRoute("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
}

func (gw *Gateway) initGWServer() {
	if gw.gwmux == nil {
		gw.gwmux = runtime.NewServeMux(
			gw.buildMuxOptions()...,
		)
	}
}

func (gw *Gateway) buildMuxOptions() []runtime.ServeMuxOption {
	return gw.muxOptions
}

func (gw *Gateway) init() {
	var (
		ctx = context.Background()
	)

	gw.ctx, gw.closefn = context.WithCancel(ctx)
	if gw.ApiPrefix == "" {
		gw.ApiPrefix = "/api"
	}

	if gw.Logger == nil {
		gw.Logger = logger.Logger
	}

	if gw.discovery == nil {
		gw.discovery = discovery.Default
	}

	gw.run.do(Init)
}

func (gw *Gateway) discoveryService(desc discovery.RegistryMessage) {
	switch desc.Method {
	case discovery.ServiceJoin:
		gw.Logger.Debug("service join", zap.String("service", desc.Desc.Service), zap.String("id", desc.Desc.ID), zap.String("target", desc.Desc.TargetURI))
		if desc.Desc.FileDescriptor == nil { // no file descriptor
			gw.getDynamicService(desc.Desc.Service, func(dynservice DynamicService) {
				conn, err := grpc.Dial(desc.Desc.TargetURI, grpc.WithInsecure(), grpc.WithBlock())
				if err != nil {
					gw.Logger.Warn("dial failed", zap.String("service", desc.Desc.Service), zap.String("id", desc.Desc.ID), zap.String("target", desc.Desc.TargetURI), zap.Error(err))
					return
				}

				if err := dynservice.AddConn(desc.Desc.ID, conn); err != nil {
					gw.Logger.Warn("add conn failed", zap.String("service", desc.Desc.Service), zap.String("id", desc.Desc.ID), zap.String("target", desc.Desc.TargetURI), zap.Error(err))
				}
			})
		} else {
			if _, ok := gw.GetService(desc.Desc.Service); ok {
				gw.getDynamicService(desc.Desc.Service, func(dynservice DynamicService) {
					conn, err := grpc.Dial(desc.Desc.TargetURI, grpc.WithInsecure(), grpc.WithBlock())
					if err != nil {
						gw.Logger.Warn("dial failed", zap.String("service", desc.Desc.Service), zap.String("id", desc.Desc.ID), zap.String("target", desc.Desc.TargetURI), zap.Error(err))
						return
					}

					if err := dynservice.AddConn(desc.Desc.ID, conn); err != nil {
						gw.Logger.Warn("add conn failed", zap.String("service", desc.Desc.Service), zap.String("id", desc.Desc.ID), zap.String("target", desc.Desc.TargetURI), zap.Error(err))
					}
				})
			} else {
				service := NewDescriptorBuilderService(desc.Desc.Service, desc.Desc.FileDescriptor)
				service.SetLogger(gw.Logger)

				if err := gw.RegisterService(service); err != nil {
					gw.Logger.Warn("register service failed", zap.String("service", desc.Desc.Service), zap.String("id", desc.Desc.ID), zap.String("target", desc.Desc.TargetURI), zap.Error(err))
					return
				}

				gw.getDynamicService(desc.Desc.Service, func(dynservice DynamicService) {
					conn, err := grpc.Dial(desc.Desc.TargetURI, grpc.WithInsecure(), grpc.WithBlock())
					if err != nil {
						gw.Logger.Warn("dial failed", zap.String("service", desc.Desc.Service), zap.String("id", desc.Desc.ID), zap.String("target", desc.Desc.TargetURI), zap.Error(err))
						return
					}

					if err := dynservice.AddConn(desc.Desc.ID, conn); err != nil {
						gw.Logger.Warn("add conn failed", zap.String("service", desc.Desc.Service), zap.String("id", desc.Desc.ID), zap.String("target", desc.Desc.TargetURI), zap.Error(err))
					}
				})
			}
		}

		// g.addService(desc.Service, desc.Callback, desc.Conn)
	case discovery.ServiceLeave:
		gw.Logger.Debug("service leave", zap.String("service", desc.Desc.Service), zap.String("id", desc.Desc.ID), zap.String("target", desc.Desc.TargetURI))
		gw.getDynamicService(desc.Desc.Service, func(dynservice DynamicService) {
			if err := dynservice.RemoveConn(desc.Desc.ID); err != nil {
				gw.Logger.Warn("remove conn failed", zap.String("service", desc.Desc.Service), zap.String("id", desc.Desc.ID), zap.String("target", desc.Desc.TargetURI), zap.Error(err))
			}
		})
	}
}

func (gw *Gateway) dynamicService(service Service, fn func(dynamicService DynamicService)) DynamicService {
	var a any = service
	dynservice, ok := a.(DynamicService)
	if !ok {
		gw.Logger.Warn("service not dynamic", zap.String("service", service.ServiceName()))
		return nil
	}

	fn(dynservice)
	return dynservice

}

func (gw *Gateway) getDynamicService(serviceName string, fn func(dynamicService DynamicService)) (DynamicService, bool) {
	var service, ok = gw.GetService(serviceName)
	if !ok {
		gw.Logger.Warn("service not found", zap.String("service", serviceName))
		return nil, false
	}

	return gw.dynamicService(service, fn), true
}
