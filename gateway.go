package mx

import (
	"context"
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/hashicorp/go-multierror"
	"github.com/hnhuaxi/platform/utils"
	"github.com/hysios/mx/internal/delegate"
	"github.com/hysios/mx/logger"
	"github.com/hysios/mx/registry"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// Gateway grpc gateway
type Gateway struct {
	ApiPrefix string

	// middleware chain
	middlewares              []Middleware                     // middleware chain
	muxOptions               []runtime.ServeMuxOption         // grpc-gateway mux options
	gwmux                    *runtime.ServeMux                // grpc-gateway mux instance
	serve                    *http.Server                     // http server
	prevAddr                 string                           // previous listen address
	registry                 *registry.ServiceRegistry        // service discovery registry
	routers                  []map[string]http.Handler        // custom routers
	notFounder               http.Handler                     // not found handler
	srvRegisters             utils.Map[string, *srvRegister]  // service register
	srvConnections           utils.Map[string, []serviceConn] // service connections
	srvMuxConns              utils.Map[string, *Muxer]        // service muxer connections
	srvImpls                 utils.Map[string, any]           // service implementations
	srvNameIdxs              []string                         // service name index
	Logger                   *zap.Logger                      // logger
	ctx                      context.Context                  // context
	closefn                  context.CancelFunc               // close function
	clientUnaryInterceptors  []grpc.UnaryClientInterceptor    // client unary interceptors
	clientStreamInterceptors []grpc.StreamClientInterceptor   // client stream interceptors
}

type serviceConn struct {
	ServiceID string
	Conn      *grpc.ClientConn
}

type srvRegister struct {
	ID                  string
	Name                string
	declare             bool
	remote              bool
	registerHandler     ConnServiceHandler
	directRegister      ServiceDirectHandler
	clientCtor          ClientCtor
	serviceHandleClient ServiceHandleClient
	isRegistred         bool
}

type (
	ServiceHandler       func(ctx context.Context, mux *runtime.ServeMux, conn grpc.ClientConnInterface) error
	ServiceDirectHandler func(ctx context.Context, mux *runtime.ServeMux, impl any) error
)

// Middleware grpc gateway middleware
type Middleware func(http.Handler) http.Handler

func (gw *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if gw.serve == nil {
		gw.serve = gw.createServer()
	}

	gw.serve.Handler.ServeHTTP(w, r)
}

// Use add middleware
func (gw *Gateway) Use(middlewares ...Middleware) {
	gw.middlewares = append(gw.middlewares, middlewares...)
}

func (gw *Gateway) HandleFunc(path string, handler func(http.ResponseWriter, *http.Request)) {
	gw.routers = append(gw.routers, map[string]http.Handler{path: http.HandlerFunc(handler)})
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

func (gw *Gateway) addService(name string, srvReg *srvRegister) {
	_, loaded := gw.srvRegisters.LoadOrStore(name, srvReg)

	if !loaded {
		gw.srvNameIdxs = append(gw.srvNameIdxs, name)
	}
}

func (gw *Gateway) addServiceConn(serviceName string, id string, conn *grpc.ClientConn) bool {
	_, ok := gw.srvRegisters.Load(serviceName)
	if !ok {
		return false
	}

	serverConns, ok := gw.srvConnections.Load(serviceName)
	if !ok {
		serverConns = []serviceConn{}
	}

	serverConns = append(serverConns, serviceConn{
		ServiceID: id,
		Conn:      conn,
	})

	gw.srvConnections.Store(serviceName, serverConns)

	muxConn, _ := gw.srvMuxConns.LoadOrStore(serviceName, &Muxer{})
	return muxConn.Add(id, conn)
}

func (gw *Gateway) hasConnected(serviceName string, id string) bool {
	serverConns, ok := gw.srvConnections.Load(serviceName)
	if !ok {
		return false
	}

	for _, conn := range serverConns {
		if conn.ServiceID == id {
			return true
		}
	}

	return false
}

func (gw *Gateway) addServiceImpl(serviceName string, impl any) bool {
	_, ok := gw.srvRegisters.Load(serviceName)
	if !ok {
		return false
	}

	gw.srvImpls.Store(serviceName, impl)
	return true
}

func (gw *Gateway) removeServiceConn(serviceName string, id string) bool {
	serverConns, ok := gw.srvConnections.Load(serviceName)
	if !ok {
		return false
	}

	var newServerConns []serviceConn
	for _, conn := range serverConns {
		if conn.ServiceID != id {
			newServerConns = append(newServerConns, conn)
		}
	}

	gw.srvConnections.Store(serviceName, newServerConns)

	muxConn, ok := gw.srvMuxConns.Load(serviceName)
	if !ok {
		return false
	}
	muxConn.Del(id)
	return true
}

func (gw *Gateway) RegisterService(name string, optFns ...RegisterOptFunc) error {
	var opts = &RegisterOption{}
	for _, optFn := range optFns {
		if err := optFn(opts); err != nil {
			return err
		}
	}

	switch opts.Method {
	case ServiceMethodHandler:
		if opts.Handler == nil {
			return errors.New("handler is nil")
		}

		gw.addService(name, &srvRegister{
			Name:            name,
			remote:          true,
			registerHandler: opts.Handler,
		})
		gw.addServiceConn(name, "", opts.Conn)
	case ServiceMethodClient:
		if opts.ClientCtor == nil || opts.ServiceHandleClient == nil {
			return errors.New("client is nil")
		}

		gw.addService(name, &srvRegister{
			Name:                name,
			remote:              true,
			declare:             true,
			clientCtor:          opts.ClientCtor,
			serviceHandleClient: opts.ServiceHandleClient,
		})
	case ServiceMethodImpl:
		if opts.Impl == nil {
			return errors.New("impl is nil")
		}

		gw.addService(name, &srvRegister{
			Name:   name,
			remote: false,
		})

		gw.addServiceImpl(name, opts.Impl)
	}

	return nil
}

func (gw *Gateway) Serve(addr string) error {
	gw.prevAddr = addr

	if err := gw.init(); err != nil {
		return err
	}

	return http.ListenAndServe(addr, gw)
}

func (gw *Gateway) ServeTLS(addr string, certFile, keyFile string) error {
	gw.prevAddr = addr
	if err := gw.init(); err != nil {
		return err
	}

	return http.ListenAndServeTLS(addr, certFile, keyFile, gw)
}

func (gw *Gateway) createServer() *http.Server {
	gw.initGWServer()

	// build router and initial middlewares
	r := gw.buildRouter()

	r.PathPrefix(gw.ApiPrefix).Handler(gw.gwmux)

	httpServer := &http.Server{
		Addr:    gw.prevAddr,
		Handler: r,
	}

	return httpServer
}

func (gw *Gateway) buildRouter() *mux.Router {
	r := mux.NewRouter()
	// use middlewares
	gw.addMetrics()
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
			r.Handle(path, handler)
		}
	}
}

func (gw *Gateway) addRouter(path string, handler http.Handler) {
	gw.routers = append(gw.routers, map[string]http.Handler{path: handler})
}

func (gw *Gateway) addMetrics() {
	gw.addRouter("/metrics", promhttp.Handler())
}

func (gw *Gateway) initRegistry() {
	if gw.registry == nil {
		gw.registry = registry.Default
	}
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

func (gw *Gateway) setupServers(ctx context.Context) error {
	gw.initGWServer()

	var errs error
	gw.srvRegisters.Range(func(serviceName string, srv *srvRegister) bool {
		if err := gw.setupRegister(ctx, srv); err != nil {
			errs = multierror.Append(errs, err)
		}

		return true
	})

	return errs
}

func (gw *Gateway) setupRegister(ctx context.Context, srv *srvRegister) error {
	if srv.isRegistred {
		return errors.New("already registered")
	}

	if srv.remote {
		switch {
		case srv.registerHandler != nil:
			conns, _ := gw.srvConnections.Load(srv.Name)
			if len(conns) == 0 {
				return errors.New("no connection")
			}

			if err := srv.registerHandler(ctx, gw.gwmux, conns[0].Conn); err != nil {
				return err
			}
			srv.isRegistred = true
		case srv.serviceHandleClient != nil && srv.clientCtor != nil:
			if muxconn, ok := gw.srvMuxConns.LoadOrStore(srv.Name, &Muxer{}); !ok {
				serviceHandler := delegate.ServiceHandleClientProxy{
					HandleClient: srv.serviceHandleClient,
					ClientCtor:   srv.clientCtor,
				}

				if err := serviceHandler.Call(ctx, gw.gwmux, muxconn); err != nil {
					return err
				}
			} else {
				return errors.New("service already registered")
			}
		}
	} else {
		if serviceImpl, ok := gw.srvImpls.Load(srv.Name); !ok {
			if err := srv.directRegister(ctx, gw.gwmux, serviceImpl); err != nil {
				return err
			}
			srv.isRegistred = true
		} else {
			return errors.New("service impl dont exist")
		}
	}

	return nil
}

func (gw *Gateway) init() error {
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

	gw.serve = gw.createServer()
	gw.initRegistry()

	gw.registry.Discovery(gw.handleDiscovery)
	go gw.registry.Start(ctx)

	if err := gw.setupServers(ctx); err != nil {
		return err
	}
	// register grpc server into gateway
	return nil
}

func (gw *Gateway) handleDiscovery(desc registry.RegistryMessage) {
	switch desc.Method {
	case registry.ServiceJoin:
		gw.Logger.Debug("service join", zap.String("service", desc.Desc.Service), zap.String("id", desc.Desc.ID), zap.String("target", desc.Desc.TargetURI))
		if srv, ok := gw.srvRegisters.Load(desc.Desc.Service); ok {
			if !srv.declare {
				return
			}

			if !gw.hasConnected(desc.Desc.Service, desc.Desc.ID) {
				newCtx, _ := context.WithCancel(gw.ctx)
				conn, err := gw.DialContext(newCtx, desc.Desc.TargetURI, grpc.WithInsecure())
				if err != nil {
					return
				}

				gw.addServiceConn(desc.Desc.Service, desc.Desc.ID, conn)
				gw.Logger.Debug("service connected", zap.String("service", desc.Desc.Service), zap.String("id", desc.Desc.ID), zap.String("target", desc.Desc.TargetURI))
			}
		}
		// g.addService(desc.Service, desc.Callback, desc.Conn)
	case registry.ServiceLeave:
		gw.Logger.Debug("service leave", zap.String("service", desc.Desc.Service), zap.String("id", desc.Desc.ID), zap.String("target", desc.Desc.TargetURI))
		if srv, ok := gw.srvRegisters.Load(desc.Desc.Service); ok {
			if !srv.declare {
				return
			}

			gw.removeServiceConn(desc.Desc.Service, desc.Desc.ID)
		}
	}
}
