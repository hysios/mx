package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/hysios/mx"
	"golang.org/x/sync/errgroup"

	pb "github.com/hysios/mx/example/gen/proto"
)

const (
	Service int = 0x01
	Gateway int = 0x02
	Both    int = Service | Gateway
)

var (
	serve  int
	direct bool
)

func main() {
	flags()
	var g errgroup.Group

	if serve&Service != 0 {
		g.Go(func() error {
			var srv = mx.NewService("HelloService", &pb.HelloService_ServiceDesc, &HelloService{})
			log.Printf("listen on %s", ":9080")
			log.Fatalf("grpc server error %s", srv.ServeOn(":9080"))
			return nil
		})
	}

	if serve&Gateway != 0 {
		g.Go(func() error {
			var gw = newGateway()
			log.Printf("listen on %s", ":8080")
			log.Fatalf("gateway shutdown error %s", gw.Serve(":8080"))
			return nil
		})
	}

	g.Wait()
}

func newGateway() *mx.Gateway {
	var gw = &mx.Gateway{}

	gw.Use(func(h http.Handler) http.Handler {
		return handlers.CombinedLoggingHandler(os.Stdout, h)
	})
	gw.Use(handlers.RecoveryHandler())

	gw.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	})

	if direct {
		gw.AddServer("HelloService", func(ctx context.Context, mux *runtime.ServeMux, impl any) error {
			return pb.RegisterHelloServiceHandlerServer(ctx, mux, impl.(*HelloService))
		}, &HelloService{})
	} else {
		log.Print(gw.AddServerS("HelloService", pb.RegisterHelloServiceHandler, mx.ConnString("localhost:9080")))
	}

	return gw
}

func flags() {
	flag.IntVar(&serve, "serve", 0, "grpc serve mode")
	flag.BoolVar(&direct, "direct", false, "direct register")
	flag.Parse()
}

// HelloService is a service
type HelloService struct {
	pb.UnimplementedHelloServiceServer
}

func (s *HelloService) Hello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloResponse, error) {
	panic("nonimplement")
}
