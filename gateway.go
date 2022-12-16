package mx

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/hysios/mx/discovery"
)

// Gateway grpc gateway
type Gateway struct {
	middlewares []Middleware // middleware chain
	// streamInterceptors []grpc.StreamServerInterceptor
	// unaryInterceptors  []grpc.UnaryServerInterceptor
	// grpcserver         *grpc.Server
	// gwmux      *runtime.ServeMux
	// muxOptions []runtime.ServeMuxOption

	serve      *http.Server
	prevAddr   string
	discovery  *ServiceDiscovery // service discovery
	routers    []map[string]http.Handler
	notFounder http.Handler
}

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

// func (g *Gateway) AddStreamInterceptor(interceptor grpc.StreamServerInterceptor) {
// 	g.streamInterceptors = append(g.streamInterceptors, interceptor)
// }

// func (g *Gateway) AddUnaryInterceptor(interceptor grpc.UnaryServerInterceptor) {
// 	g.unaryInterceptors = append(g.unaryInterceptors, interceptor)
// }

func (g *Gateway) createServer() *http.Server {
	// build router and initial middlewares
	r := g.buildRouter()

	httpServer := &http.Server{
		Addr:    g.prevAddr,
		Handler: r,
	}

	return httpServer
}

func (g *Gateway) buildRouter() *mux.Router {
	r := mux.NewRouter()
	// use middlewares
	for _, m := range g.middlewares {
		r.Use(mux.MiddlewareFunc(m))
	}

	for _, router := range g.routers {
		for path, handler := range router {
			r.Handle(path, handler)
		}
	}

	r.NotFoundHandler = g.notFounder

	return r
}

func (g *Gateway) initDiscovery() {
	if g.discovery == nil {
		g.discovery = &ServiceDiscovery{}
	}
}

func (g *Gateway) init() error {
	g.serve = g.createServer()
	g.initDiscovery()

	g.discovery.Discovery(func(desc *discovery.ServiceDesc) {

	})
	// register grpc server into gateway
	return nil
}

func (g *Gateway) HandleFunc(path string, handler func(http.ResponseWriter, *http.Request)) {
	g.routers = append(g.routers, map[string]http.Handler{path: http.HandlerFunc(handler)})
}

func (g *Gateway) NotFoundFunc(handler http.Handler) {
	g.notFounder = handler
}

func (g *Gateway) AddServer(s *http.Server) {
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
