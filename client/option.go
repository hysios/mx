package client

import "google.golang.org/grpc"

type MakeOption struct {
	mockClient         grpc.ClientConnInterface
	unaryInterceptors  []grpc.UnaryClientInterceptor
	streamInterceptors []grpc.StreamClientInterceptor
}

type MakeOptionFunc func(*MakeOption)

func WithMockClient(mockClient grpc.ClientConnInterface) MakeOptionFunc {
	return func(o *MakeOption) {
		o.mockClient = mockClient
	}
}

func WithUnaryClientInterceptor(interceptors ...grpc.UnaryClientInterceptor) MakeOptionFunc {
	return func(o *MakeOption) {
		o.unaryInterceptors = interceptors
	}
}

func WithStreamClientInterceptor(interceptors ...grpc.StreamClientInterceptor) MakeOptionFunc {
	return func(o *MakeOption) {
		o.streamInterceptors = interceptors
	}
}
