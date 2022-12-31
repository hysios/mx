package mx

import (
	"context"
	"errors"

	"google.golang.org/grpc"
)

// SignalConn is a wrapper of grpc.ClientConnInterface, used to capture any type of error when calling Invoke or NewStream method.
// The error will be sent to the errCh channel, which can be obtained by calling the Err method.
type SignalConn struct {
	conn  grpc.ClientConnInterface
	errCh chan error
}

func NewSignalConn(conn grpc.ClientConnInterface) *SignalConn {
	return &SignalConn{
		conn:  conn,
		errCh: make(chan error, 1),
	}
}

// SignalConn implement grpc.ClientConnInterface
func (s *SignalConn) Invoke(ctx context.Context, method string, args interface{}, reply interface{}, opts ...grpc.CallOption) (err error) {
	defer func() {
		// recover any type error
		if r := recover(); r != nil {
			switch _err := r.(type) {
			case error:
				s.errCh <- _err
				err = _err
			case string:
				s.errCh <- errors.New(_err)
				err = errors.New(_err)
			default:
				s.errCh <- errors.New("unknown error")
				err = errors.New("unknown error")

			}
		}
	}()

	err = s.conn.Invoke(ctx, method, args, reply, opts...)
	if err != nil {
		s.errCh <- err
	}

	return
}

func (s *SignalConn) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (stream grpc.ClientStream, err error) {
	defer func() {
		// recover any type error
		if r := recover(); r != nil {
			switch _err := r.(type) {
			case error:
				s.errCh <- _err
				err = _err
			case string:
				s.errCh <- errors.New(_err)
				err = errors.New(_err)
			default:
				s.errCh <- errors.New("unknown error")
				err = errors.New("unknown error")

			}
		}
	}()

	stream, err = s.conn.NewStream(ctx, desc, method, opts...)
	if err != nil {
		s.errCh <- err
	}

	return
}

func (s *SignalConn) Err() <-chan error {
	return s.errCh
}

func (gw *Gateway) DialContext(ctx context.Context, target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	opts = append(opts, grpc.WithChainUnaryInterceptor(gw.clientUnaryInterceptors...))
	opts = append(opts, grpc.WithChainStreamInterceptor(gw.clientStreamInterceptors...))

	conn, err := grpc.DialContext(ctx, target, opts...)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
