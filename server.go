package mx

import (
	"net"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/hysios/mx/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Service is a service
type Service struct {
	Name string

	desc        *grpc.ServiceDesc
	impl        any
	grpcserver  *grpc.Server
	grpcOptions []grpc.ServerOption
}

func NewService(name string, desc *grpc.ServiceDesc, impl any) *Service {

	return &Service{
		Name: name,
		desc: desc,
		impl: impl,
	}
}

func (s *Service) init() {
	s.initServer()

	s.grpcserver.RegisterService(s.desc, s.impl)
}

func (s *Service) initServer() {
	if s.grpcserver == nil {
		s.grpcserver = grpc.NewServer(s.buildGrpcOptions()...)
	}
}

func (s *Service) customFunc(p interface{}) (err error) {
	return status.Errorf(codes.Unknown, "panic triggered: %s", errors.Wrap(p))
}

func (s *Service) buildGrpcOptions() []grpc.ServerOption {
	var (
		options   []grpc.ServerOption
		logger, _ = zap.NewDevelopment()
	)

	options = append(options, grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
		grpc_ctxtags.UnaryServerInterceptor(),
		grpc_opentracing.UnaryServerInterceptor(),
		grpc_prometheus.UnaryServerInterceptor,
		grpc_zap.UnaryServerInterceptor(logger),
		// grpc_auth.UnaryServerInterceptor(ServiceAuth(db, log)),
		grpc_recovery.UnaryServerInterceptor(grpc_recovery.WithRecoveryHandler(s.customFunc)),
	)))
	options = append(options, s.grpcOptions...)
	return options
}

func (s *Service) AddServiceDesc(desc *grpc.ServiceDesc, impl any) {
	s.initServer()

	s.grpcserver.RegisterService(desc, impl)
}

func (s *Service) AddService(another *Service) {
	s.initServer()

	s.grpcserver.RegisterService(another.desc, another.impl)
}

func (s *Service) Serve(lns net.Listener) error {
	s.init()

	grpc_prometheus.Register(s.grpcserver)

	return s.grpcserver.Serve(lns)
}

func (s *Service) ServeOn(addr string) error {
	h, port, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}

	ln, err := net.Listen("tcp", net.JoinHostPort(h, port))
	if err != nil {
		return err
	}

	return s.Serve(ln)
}
