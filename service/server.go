package service

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"

	"github.com/hysios/mx/discovery"
	"github.com/hysios/mx/errors"
	"github.com/hysios/mx/logger"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Server is a service
type Server struct {
	ID   string
	Name string

	opts        ServerOption
	desc        *grpc.ServiceDesc
	impl        any
	grpcserver  *grpc.Server
	grpcOptions []grpc.ServerOption
	ln          net.Listener
	logger      *zap.Logger

	unaryInterceptors  []grpc.UnaryServerInterceptor
	streamInterceptors []grpc.StreamServerInterceptor

	listenAddrs []chan net.Addr
}

func New(name string, impl any, optfns ...ServerOptionFunc) *Server {
	var opts ServerOption
	for _, fn := range optfns {
		if err := fn(&opts); err != nil {
			panic(err)
		}
	}

	if opts.Logger == nil {
		opts.Logger = logger.Cli
	}
	if opts.ServiceDesc != nil {
		validService(name, impl, opts.ServiceDesc)
	}

	return &Server{
		Name:               name,
		opts:               opts,
		impl:               impl,
		unaryInterceptors:  make([]grpc.UnaryServerInterceptor, 0),
		streamInterceptors: make([]grpc.StreamServerInterceptor, 0),
		logger:             opts.Logger,
	}
}

func NewServiceDesc(desc *grpc.ServiceDesc, impl any) *Server {
	return New(desc.ServiceName, impl, WithServiceDesc(desc))
}

func NewServiceFileDescriptor(desc *grpc.ServiceDesc, impl any, filedesc protoreflect.FileDescriptor) *Server {
	return New(desc.ServiceName, impl, WithServiceDesc(desc), WithFileDescriptor(filedesc))
}

func validService(name string, impl any, serviceDesc *grpc.ServiceDesc) {
	if !strings.HasSuffix(serviceDesc.ServiceName, name) {
		logger.Logger.Warn("service name is not match with service desc", zap.String("service_name", name), zap.String("service_desc", serviceDesc.ServiceName))
	}
}

func (s *Server) init() {
	s.initServer()

	if s.opts.ServiceDesc != nil {
		s.registerDesc(s.opts.ServiceDesc, s.impl)
	} else if s.opts.ServiceRegistrar != nil {
		s.opts.ServiceRegistrar(s.grpcserver, s.impl)
	}
}

func (s *Server) initServer() {
	if s.opts.Logger == nil {
		s.opts.Logger = zap.L()
	}

	if s.grpcserver == nil {
		s.grpcserver = grpc.NewServer(s.buildGrpcOptions()...)
	}
}

func (s *Server) recoverFunc(p interface{}) (err error) {
	return status.Errorf(codes.Unknown, "panic triggered: %s", errors.Wrap(p))
}

func (s *Server) authFunc(ctx context.Context) (context.Context, error) {
	return ctx, nil
}

func (s *Server) buildGrpcOptions() []grpc.ServerOption {
	var (
		options []grpc.ServerOption
	)

	options = append(options, grpc.UnaryInterceptor(s.buildUnaryServerInterceptor()))
	options = append(options, grpc.StreamInterceptor(s.buildStreamServerInterceptor()))
	options = append(options, s.grpcOptions...)
	return options
}

func (s *Server) buildUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	var interceptors = []grpc.UnaryServerInterceptor{
		grpc_ctxtags.UnaryServerInterceptor(),
		grpc_opentracing.UnaryServerInterceptor(),
		grpc_prometheus.UnaryServerInterceptor,
		grpc_zap.UnaryServerInterceptor(s.opts.Logger),
		grpc_auth.UnaryServerInterceptor(s.authFunc),
		grpc_recovery.UnaryServerInterceptor(grpc_recovery.WithRecoveryHandler(s.recoverFunc)),
	}

	interceptors = append(interceptors, s.unaryInterceptors...)
	return grpc_middleware.ChainUnaryServer(
		interceptors...,
	)
}

func (s *Server) buildStreamServerInterceptor() grpc.StreamServerInterceptor {
	var interceptors = []grpc.StreamServerInterceptor{
		grpc_ctxtags.StreamServerInterceptor(),
		grpc_opentracing.StreamServerInterceptor(),
		grpc_prometheus.StreamServerInterceptor,
		grpc_zap.StreamServerInterceptor(s.opts.Logger),
		grpc_auth.StreamServerInterceptor(s.authFunc),
		grpc_recovery.StreamServerInterceptor(grpc_recovery.WithRecoveryHandler(s.recoverFunc)),
	}

	interceptors = append(interceptors, s.streamInterceptors...)

	return grpc_middleware.ChainStreamServer(
		interceptors...,
	)
}

func (s *Server) AddServiceDesc(desc *grpc.ServiceDesc, impl any) {
	s.initServer()

	s.registerDesc(desc, impl)
}

func (s *Server) AddService(another *Server) {
	s.initServer()

	s.registerDesc(another.desc, another.impl)
}

func (s *Server) registerDesc(desc *grpc.ServiceDesc, impl any) {
	if init, ok := impl.(interface {
		Init() error
	}); ok {
		if err := init.Init(); err != nil {
			panic(err)
		}
	}

	s.grpcserver.RegisterService(desc, impl)
}

func (s *Server) Serve(lns net.Listener) error {
	s.init()
	s.ln = lns

	servech := make(chan net.Addr, 1)
	go s.waitForStart(servech)

	grpc_prometheus.Register(s.grpcserver)
	s.teardown()

	go func() {
		time.Sleep(time.Millisecond * 500)
		servech <- lns.Addr()
		s.logger.Info("server start", zap.String("name", s.Name), zap.String("address", lns.Addr().String()))
	}()
	return s.grpcserver.Serve(lns)
}

func (s *Server) ServeOn(addr string) error {
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

func (s *Server) Start() error {
	return s.ServeOn(":0")
}

func (s *Server) Addr() net.Addr {
	return <-s.AddrCh()
}

func (s *Server) AddrCh() chan net.Addr {
	ch := make(chan net.Addr, 1)
	if s.ln == nil {
		s.addListen(ch)
	} else {
		ch <- s.ln.Addr()
	}

	return ch
}

func (s *Server) waitForStart(serveCh chan net.Addr) {
	for {
		select {
		case addr := <-serveCh:
			for _, ch := range s.listenAddrs {
				select {
				case ch <- addr:
				default:
				}
			}
		}
	}
}

func (s *Server) addListen(ch chan net.Addr) {
	s.listenAddrs = append(s.listenAddrs, ch)
}

func (s *Server) GetID() string {
	if s.ID != "" {
		return s.ID
	}

	return fmt.Sprintf("%s_%d", s.Name, os.Getpid())
}

func (s *Server) teardown() {
	// on os signal ctrl+c && kill  trigger graceful stop
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM|syscall.SIGINT|syscall.SIGKILL)
	go func() {
		<-c
		s.grpcserver.GracefulStop()
	}()
}

func (s *Server) ServiceDesc() discovery.ServiceDesc {
	if s.opts.FileDescriptor == nil {
		return discovery.ServiceDesc{
			ID:        s.GetID(),
			Namespace: s.opts.Namespace,
			Service:   s.Name,
			Type:      "grpc_server",
			Address:   s.Addr().String(),
		}
	} else {
		return discovery.ServiceDesc{
			ID:                s.GetID(),
			Namespace:         s.opts.Namespace,
			Service:           s.Name,
			Type:              "grpc_server",
			Address:           s.Addr().String(),
			FileDescriptor:    s.opts.FileDescriptor,
			FileDescriptorKey: s.opts.FileDescriptor.Path(),
		}
	}
}
