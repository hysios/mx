package server

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"strconv"
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

	"github.com/hysios/mx"
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
	ID         string
	ServerName string

	opts         ServerOption
	serviceDescs []serviceDesc
	grpcserver   *grpc.Server
	grpcOptions  []grpc.ServerOption
	ln           net.Listener
	logger       *zap.Logger

	unaryInterceptors  []grpc.UnaryServerInterceptor
	streamInterceptors []grpc.StreamServerInterceptor

	listenAddrs []chan net.Addr
}

func New(name string, optfns ...ServerOptionFunc) *Server {
	var opts ServerOption
	for _, fn := range optfns {
		if err := fn(&opts); err != nil {
			panic(err)
		}
	}

	if opts.Logger == nil {
		opts.Logger = logger.Cli
	}

	return &Server{
		ServerName:         name,
		opts:               opts,
		unaryInterceptors:  make([]grpc.UnaryServerInterceptor, 0),
		streamInterceptors: make([]grpc.StreamServerInterceptor, 0),
		logger:             opts.Logger,
	}
}

func NewServiceDesc(desc grpc.ServiceDesc, impl any) *Server {
	s := New(desc.ServiceName)
	s.RegisterService(&desc, impl)
	return s
}

func NewServiceFileDescriptor(desc grpc.ServiceDesc, impl any, filedesc protoreflect.FileDescriptor) *Server {
	s := New(desc.ServiceName)
	s.RegisterService(&desc, impl, WithFileDescriptor(filedesc))
	return s
}

func validService(name string, serviceDesc *grpc.ServiceDesc) {
	if !strings.HasSuffix(serviceDesc.ServiceName, name) {
		logger.Logger.Warn("service name is not match with service desc", zap.String("service_name", name), zap.String("service_desc", serviceDesc.ServiceName))
	}
}

func (s *Server) init() {
	s.initServer()

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

func (s *Server) RegisterService(desc *grpc.ServiceDesc, impl any, optfns ...ServerOptionFunc) {
	s.initServer()

	s.registerDesc(desc, impl, optfns...)
}

func (s *Server) registerDesc(desc *grpc.ServiceDesc, impl any, optfns ...ServerOptionFunc) {
	var opts ServerOption
	for _, fn := range optfns {
		if err := fn(&opts); err != nil {
			panic(err)
		}
	}

	s.serviceDescs = append(s.serviceDescs, func() serviceDesc {
		return serviceDesc{
			desc:         desc,
			impl:         impl,
			namespace:    opts.Namespace,
			filedescript: opts.FileDescriptor,
		}
	}())

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
		s.logger.Info("server start", zap.String("name", s.ServerName), zap.String("address", lns.Addr().String()))
	}()
	return s.grpcserver.Serve(lns)
}

func (s *Server) ServeOn(addr string) error {
	// h, port, err := net.SplitHostPort(addr)
	// if err != nil {
	// 	return err
	// }
	// if port == "0" && s.opts.PersistentPort {
	// 	port = s.loadPersistentPort()
	// }

	ln, err := s.ListenWithPersistentPort(addr, s.opts.PersistentPort)
	// ln, err := net.Listen("tcp", net.JoinHostPort(h, port))
	if err != nil {
		return err
	}

	return s.Serve(ln)
}

// ListenWithPersistentPort listen on the given address, and save the port to file
func (s *Server) ListenWithPersistentPort(addr string, persistent bool) (ln net.Listener, err error) {
	h, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	if port == "0" && persistent {
		_port := s.loadPersistentPort()
		if _port != "" {
			port = _port
		} else {
			defer func() {
				if ln != nil {
					_, port, err := net.SplitHostPort(ln.Addr().String())
					if err != nil {
						return
					}
					_ = s.savePersistentPort(port)
				}
			}()
		}
	}

	return net.Listen("tcp", net.JoinHostPort(h, port))
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

	return fmt.Sprintf("%s_%d", s.ServerName, os.Getpid())
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

func (s *Server) ServiceDescs() []discovery.ServiceDesc {
	var descs []discovery.ServiceDesc
	for _, desc := range s.serviceDescs {
		var filedescriptkey string
		if desc.filedescript != nil {
			filedescriptkey = desc.filedescript.Path()
		}

		descs = append(descs, discovery.ServiceDesc{
			ID:                desc.GetID(),
			Namespace:         desc.namespace,
			Service:           desc.desc.ServiceName,
			Type:              mx.ServerType,
			Address:           s.Addr().String(),
			FileDescriptor:    desc.filedescript,
			FileDescriptorKey: filedescriptkey,
			Group:             s.GetID(),
		})
	}

	return descs
}

// loadPersistentPort
func (s *Server) loadPersistentPort() string {
	// open current dir .PORT file
	// if not exist create it
	// if exist read it
	// read text for number
	// if is new port and save it
	// at last return port

	f, err := os.OpenFile(".PORT", os.O_RDONLY, 0666)
	if err != nil {
		return ""
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return ""
	}

	port := string(b)
	_, err = strconv.Atoi(port)
	if err != nil {
		return ""
	}

	return port
}

// savePersistentPort
func (s *Server) savePersistentPort(port string) error {
	f, err := os.OpenFile(".PORT", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(port)
	if err != nil {
		return err
	}

	return nil
}

type serviceDesc struct {
	ID           string
	desc         *grpc.ServiceDesc
	impl         any
	namespace    string
	filedescript protoreflect.FileDescriptor
}

func (desc *serviceDesc) GetID() string {
	if desc.ID == "" {
		desc.ID = fmt.Sprintf("%s_%d", desc.desc.ServiceName, os.Getpid())
	}
	return desc.ID
}
