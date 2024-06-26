package client

import "google.golang.org/grpc"

type MakeOption struct {
	ConnectURI         string
	Insecure           bool
	mockClient         grpc.ClientConnInterface
	unaryInterceptors  []grpc.UnaryClientInterceptor
	streamInterceptors []grpc.StreamClientInterceptor

	dialOptions []grpc.DialOption
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

func WithConnectURI(connectURI string) MakeOptionFunc {
	return func(o *MakeOption) {
		o.ConnectURI = connectURI
	}
}

func WithDialOptions(dialOptions ...grpc.DialOption) MakeOptionFunc {
	return func(o *MakeOption) {
		o.dialOptions = dialOptions
	}
}

func WithInsecure(on bool) MakeOptionFunc {
	return func(o *MakeOption) {
		o.Insecure = on
	}
}

func WithUserAgent(userAgent string) MakeOptionFunc {
	return func(o *MakeOption) {
		o.dialOptions = append(o.dialOptions, grpc.WithUserAgent(userAgent))
	}
}
