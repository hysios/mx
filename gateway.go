package mx

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/hysios/mx/discovery"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/multierr"
	"google.golang.org/grpc"
)

// Gateway grpc gateway
type Gateway struct {
	ApiPrefix string

	middlewares []Middleware // middleware chain
	// streamInterceptors []grpc.StreamServerInterceptor
	// unaryInterceptors  []grpc.UnaryServerInterceptor
	// grpcserver         *grpc.Server
	muxOptions   []runtime.ServeMuxOption
	gwmux        *runtime.ServeMux
	serve        *http.Server
	prevAddr     string
	discovery    *ServiceDiscovery // service discovery
	routers      []map[string]http.Handler
	notFounder   http.Handler
	srvRegisters map[string]ServerRegister
}

type ServerRegister struct {
	Name           string
	Remote         bool
	Register       ServiceClientRegister
	DirectRegister ServiceDirectRegister
	Conn           *grpc.ClientConn
	Impl           any
}

type (
	ServiceClientRegister func(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error
	ServiceDirectRegister func(ctx context.Context, mux *runtime.ServeMux, impl any) error
)

// Middleware grpc gateway middleware
type Middleware func(http.Handler) http.Handler

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if g.serve == nil {
		g.serve = g.createServer()
	}

	g.serve.Handler.ServeHTTP(w, r)
}

// Use add middleware
func (g *Gateway) Use(m Middleware) {
	g.middlewares = append(g.middlewares, m)
}

func (g *Gateway) HandleFunc(path string, handler func(http.ResponseWriter, *http.Request)) {
	g.routers = append(g.routers, map[string]http.Handler{path: http.HandlerFunc(handler)})
}

func (g *Gateway) NotFoundFunc(handler http.Handler) {
	g.notFounder = handler
}

func (g *Gateway) AddServerClient(name string, callback ServiceClientRegister, conn *grpc.ClientConn) {
	if g.srvRegisters == nil {
		g.srvRegisters = make(map[string]ServerRegister)
	}

	g.srvRegisters[name] = ServerRegister{
		Name:     name,
		Remote:   true,
		Register: callback,
		Conn:     conn,
	}
}

func (g *Gateway) AddServer(name string, callback ServiceDirectRegister, impl any) {
	if g.srvRegisters == nil {
		g.srvRegisters = make(map[string]ServerRegister)
	}

	g.srvRegisters[name] = ServerRegister{
		Name:           name,
		DirectRegister: callback,
		Impl:           impl,
	}
}

func (g *Gateway) AddServerS(name string, callback ServiceClientRegister, connStr ConnString) error {
	if g.srvRegisters == nil {
		g.srvRegisters = make(map[string]ServerRegister)
	}

	conn, err := connStr.Open()
	if err != nil {
		return err
	}

	g.AddServerClient(name, callback, conn)
	return nil
}

func (g *Gateway) Serve(addr string) error {
	g.prevAddr = addr

	if err := g.init(); err != nil {
		return err
	}

	return http.ListenAndServe(addr, g)
}

func (g *Gateway) ServeTLS(addr string, certFile, keyFile string) error {
	g.prevAddr = addr
	if err := g.init(); err != nil {
		return err
	}

	return http.ListenAndServeTLS(addr, certFile, keyFile, g)
}

func (g *Gateway) createServer() *http.Server {
	g.initGWServer()

	// build router and initial middlewares
	r := g.buildRouter()

	r.PathPrefix(g.ApiPrefix).Handler(g.gwmux)

	httpServer := &http.Server{
		Addr:    g.prevAddr,
		Handler: r,
	}

	return httpServer
}

func (g *Gateway) buildRouter() *mux.Router {
	r := mux.NewRouter()
	// use middlewares
	g.addMetrics()
	g.buildMiddlewares(r)
	g.setupRouters(r)

	r.NotFoundHandler = g.notFounder
	return r
}

func (g *Gateway) buildMiddlewares(r *mux.Router) {
	for _, m := range g.middlewares {
		r.Use(mux.MiddlewareFunc(m))
	}
}

func (g *Gateway) setupRouters(r *mux.Router) {
	for _, router := range g.routers {
		for path, handler := range router {
			r.Handle(path, handler)
		}
	}
}

func (g *Gateway) addRouter(path string, handler http.Handler) {
	g.routers = append(g.routers, map[string]http.Handler{path: handler})
}

func (g *Gateway) addMetrics() {
	g.addRouter("/metrics", promhttp.Handler())
}

func (g *Gateway) initDiscovery() {
	if g.discovery == nil {
		g.discovery = &ServiceDiscovery{}
	}
}

func (g *Gateway) initGWServer() {
	if g.gwmux == nil {
		g.gwmux = runtime.NewServeMux(
			g.buildMuxOptions()...,
		)
	}
}

func (g *Gateway) buildMuxOptions() []runtime.ServeMuxOption {
	return g.muxOptions
}

func (g *Gateway) setupServers(ctx context.Context) error {
	g.initGWServer()

	var errs error
	for _, srv := range g.srvRegisters {
		if srv.Remote {
			if err := srv.Register(ctx, g.gwmux, srv.Conn); err != nil {
				errs = multierr.Append(errs, err)
			}
		} else {
			if err := srv.DirectRegister(ctx, g.gwmux, srv.Impl); err != nil {
				errs = multierr.Append(errs, err)
			}
		}
	}

	return errs
}

func (g *Gateway) init() error {
	var (
		ctx = context.Background()
	)

	if g.ApiPrefix == "" {
		g.ApiPrefix = "/api"
	}

	g.serve = g.createServer()
	g.initDiscovery()

	g.discovery.Discovery(func(desc *discovery.ServiceDesc) {
	})

	if err := g.setupServers(ctx); err != nil {
		return err
	}
	// register grpc server into gateway
	return nil
}
