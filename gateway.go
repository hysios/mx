package mx

import (
	"net/http"

	"github.com/gorilla/mux"
)

// Gateway grpc gateway
type Gateway struct {
	middlewares []Middleware // middleware chain
	// streamInterceptors []grpc.StreamServerInterceptor
	// unaryInterceptors  []grpc.UnaryServerInterceptor
	// grpcserver         *grpc.Server
	// gwmux      *runtime.ServeMux
	// muxOptions []runtime.ServeMuxOption

	serve     *http.Server
	prevAddr  string
	discovery *ServiceDiscovery // service discovery
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

	return r
}

func (g *Gateway) Init() error {
	g.serve = g.createServer()

	if g.discovery == nil {
		g.discovery = &ServiceDiscovery{}
	}

	g.discovery.Discovery(func(desc *ServiceDesc) {

	})
	// register grpc server into gateway
	return nil
}

func (g *Gateway) Serve(addr string) error {
	g.prevAddr = addr

	if err := g.Init(); err != nil {
		return err
	}

	return http.ListenAndServe(addr, g)
}

func (g *Gateway) ServeTLS(addr string, certFile, keyFile string) error {
	g.prevAddr = addr
	if err := g.Init(); err != nil {
		return err
	}

	return http.ListenAndServeTLS(addr, certFile, keyFile, g)
}
