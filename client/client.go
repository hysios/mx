package client

import (
	"errors"
	"reflect"
	"sync"

	"github.com/hysios/mx"
	"github.com/hysios/mx/discovery"
	"github.com/hysios/mx/discovery/agent"
	"github.com/hysios/mx/internal/delegate"
	"github.com/hysios/mx/logger"
	"github.com/hysios/mx/utils"
	"go.uber.org/zap"

	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	"google.golang.org/grpc"
)

type Client any

var (
	clientRegistry = utils.Registry[mx.ClientCtor]{}
	commonOption   MakeOption
)

func getDiscoveryConn(serviceName string, opts *MakeOption) (grpc.ClientConnInterface, error) {
	services, ok := agent.Default.Lookup(serviceName,
		discovery.WithServiceType(mx.ServerType),
		discovery.WithNamespace(discovery.Namespace),
	)
	if !ok {
		return nil, mx.ErrServiceNotFound
	}

	if len(services) == 0 {
		return nil, mx.ErrServiceNotFound
	}

	logger.Logger.Info("dial", zap.String("target", services[0].TargetURI))
	rawconn, err := dial(services[0].TargetURI, opts)
	if err != nil {
		return nil, err
	}

	return rawconn, nil
}

func getDialConn(opts *MakeOption) (grpc.ClientConnInterface, error) {
	rawconn, err := dial(opts.ConnectURI, opts)
	if err != nil {
		return nil, err
	}

	return rawconn, nil
}

func Make(serviceName string, impl interface{}, optfns ...MakeOptionFunc) error {
	opts := evaluteOptions(optfns...)

	ctor, ok := clientRegistry.Lookup(serviceName)
	if !ok {
		return mx.ErrServiceNotFound
	}

	var client interface{}

	if opts.ConnectURI != "" {
		rawconn, err := getDialConn(opts)
		if err != nil {
			return err
		}

		proxy := delegate.ClientCtor{ClientCtor: ctor()}
		client, err = proxy.Call(rawconn)
		if err != nil {
			return err
		}
	} else {
		rawconn, err := getDiscoveryConn(serviceName, opts)
		if err != nil {
			return err
		}

		proxy := delegate.ClientCtor{ClientCtor: ctor()}
		client, err = proxy.Call(rawconn)
		if err != nil {
			return err
		}
	}

	reflect.ValueOf(impl).Elem().Set(reflect.ValueOf(client))
	return nil
}

var (
	clientCache sync.Map
)

func LMake(serviceName string, recvfn interface{}, optfns ...MakeOptionFunc) error {
	opts := evaluteOptions(optfns...)

	ctor, ok := clientRegistry.Lookup(serviceName)
	if !ok {
		return mx.ErrServiceNotFound
	}

	receType := reflect.TypeOf(recvfn).Elem()

	if receType.Kind() != reflect.Func {
		return errors.New("recvfn must be a function")
	}

	if receType.NumIn() != 0 {
		return errors.New("recvfn must be a function without any input parameters")
	}

	// receType out must be one, and be kind of interface
	if receType.NumOut() != 1 && receType.Out(0).Kind() != reflect.Interface {
		return errors.New("recvfn must be a function with one output parameter, and the parameter must be kind of interface")
	}

	ctorfn := reflect.MakeFunc(receType, func(_ []reflect.Value) []reflect.Value {
		val := cacheWith(serviceName, func() any {
			var (
				conn   grpc.ClientConnInterface
				client interface{}
				err    error
			)

			if opts.mockClient != nil {
				conn = opts.mockClient
			} else if opts.ConnectURI != "" {
				logger.Logger.Info("dial", zap.String("target", opts.ConnectURI))
				conn, err = getDialConn(opts)
				if err != nil {
					panic(err)
				}
			} else {
				conn, err = getDiscoveryConn(serviceName, opts)
				if err != nil {
					panic(err)
				}
				sgconn := mx.NewSignalConn(conn)
				go func() {
					err := <-sgconn.Err()
					logger.Logger.Warn("grpc connection error", zap.Error(err))
					cleanCache(serviceName)
				}()

				conn = sgconn
			}

			proxy := delegate.ClientCtor{ClientCtor: ctor()}
			client, err = proxy.Call(conn)
			if err != nil {
				panic(err)
			}
			return client
		})

		return []reflect.Value{reflect.ValueOf(val)}
	})

	reflect.ValueOf(recvfn).Elem().Set(ctorfn)
	return nil
}

func Registry(serviceName string, clientCtor mx.ClientCtor) {
	clientProxy := delegate.ClientCtor{
		ClientCtor: clientCtor,
	}
	if err := clientProxy.Valid(); err != nil {
		panic(err)
	}

	clientRegistry.Register(serviceName, func() mx.ClientCtor {
		return clientCtor
	})
}

func cacheWith(serviceName string, fn func() any) any {
	client, ok := clientCache.Load(serviceName)
	if ok {
		return client
	}

	client = fn()
	clientCache.Store(serviceName, client)
	return client
}

func cleanCache(servcieName string) {
	clientCache.Delete(servcieName)
}

func SetUnaryClientInterceptor(interceptors ...grpc.UnaryClientInterceptor) {
	commonOption.unaryInterceptors = interceptors
}

func SetStreamClientInterceptor(interceptors ...grpc.StreamClientInterceptor) {
	commonOption.streamInterceptors = interceptors
}

func dial(target string, opts *MakeOption) (*grpc.ClientConn, error) {
	var (
		dialOpts []grpc.DialOption
	)

	if opts.Insecure {
		dialOpts = append(dialOpts, grpc.WithInsecure())
	}

	dialOpts = append(dialOpts, grpc.WithChainUnaryInterceptor(opts.unaryInterceptors...))
	dialOpts = append(dialOpts, grpc.WithChainStreamInterceptor(opts.streamInterceptors...))
	dialOpts = append(dialOpts, opts.dialOptions...)

	return grpc.Dial(target, dialOpts...)
}

func evaluteOptions(opts ...MakeOptionFunc) *MakeOption {
	opt := &MakeOption{
		unaryInterceptors:  append(commonOption.unaryInterceptors, defaultUnaryInterceptors()...),
		streamInterceptors: append(commonOption.streamInterceptors, defaultStreamInterceptors()...),
	}

	for _, optfn := range opts {
		optfn(opt)
	}

	return opt
}

func defaultUnaryInterceptors() []grpc.UnaryClientInterceptor {
	return []grpc.UnaryClientInterceptor{
		grpc_opentracing.UnaryClientInterceptor(),
	}
}

func defaultStreamInterceptors() []grpc.StreamClientInterceptor {
	return []grpc.StreamClientInterceptor{
		grpc_opentracing.StreamClientInterceptor(),
	}
}
