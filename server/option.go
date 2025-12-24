package server

import (
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type RegisterOption struct {
}

type RegisterOptionFunc func(*RegisterOption) error

type Registrar func(s *grpc.Server, impl any)

type ServerOption struct {
	Namespace          string
	ServiceDesc        *grpc.ServiceDesc
	Logger             *zap.Logger
	FileDescriptor     protoreflect.FileDescriptor
	PersistentPort     bool
	PersistentPortFile string
}

type ServerOptionFunc func(*ServerOption) error

func WithServiceDesc(desc *grpc.ServiceDesc) ServerOptionFunc {
	return func(o *ServerOption) error {
		o.ServiceDesc = desc
		return nil
	}
}

func WithLogger(logger *zap.Logger) ServerOptionFunc {
	return func(o *ServerOption) error {
		o.Logger = logger
		return nil
	}
}

func WithNamespace(ns string) ServerOptionFunc {
	return func(o *ServerOption) error {
		o.Namespace = ns
		return nil
	}
}

func WithFileDescriptor(fd protoreflect.FileDescriptor) ServerOptionFunc {
	return func(o *ServerOption) error {
		o.FileDescriptor = fd
		return nil
	}
}

// 持久端口
func WithPersistentPort(filename ...string) ServerOptionFunc {
	return func(o *ServerOption) error {
		o.PersistentPort = true
		if len(filename) > 0 && filename[0] != "" {
			o.PersistentPortFile = filename[0]
		} else {
			o.PersistentPortFile = ".PORT"
		}
		return nil
	}
}
