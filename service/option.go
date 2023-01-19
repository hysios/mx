package service

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
	Namespace        string
	ServiceDesc      *grpc.ServiceDesc
	ServiceRegistrar Registrar
	Logger           *zap.Logger
	NoRegister       bool
	FileDescriptor   protoreflect.FileDescriptor
}

type ServerOptionFunc func(*ServerOption) error

func WithServiceDesc(desc *grpc.ServiceDesc) ServerOptionFunc {
	return func(o *ServerOption) error {
		o.ServiceDesc = desc
		return nil
	}
}

func WithServiceRegistrar(registrar Registrar) ServerOptionFunc {
	return func(o *ServerOption) error {
		o.ServiceRegistrar = registrar
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

func WithNoRegister() ServerOptionFunc {
	return func(o *ServerOption) error {
		o.NoRegister = true
		return nil
	}
}

func WithFileDescriptor(fd protoreflect.FileDescriptor) ServerOptionFunc {
	return func(o *ServerOption) error {
		o.FileDescriptor = fd
		return nil
	}
}
