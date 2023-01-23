package mx

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"

	"google.golang.org/grpc"
)

type LoopStrategy int

const (
	RoundRobin LoopStrategy = iota
	Random     LoopStrategy = iota
)

// Muxer is a load balancer connector implementation that can distribute
// multiple connections to multiple services.
type Muxer struct {
	Streagy  LoopStrategy
	conns    []abstractConn
	connLock sync.RWMutex
	lastIdx  atomic.Int32
}

type abstractConn struct {
	ServiceID string
	Conn      grpc.ClientConnInterface
}

// Add is used to add a new connection to the muxer.
func (m *Muxer) Add(id string, conn grpc.ClientConnInterface) bool {
	m.connLock.Lock()
	defer m.connLock.Unlock()

	for _, c := range m.conns {
		if c.ServiceID == id {
			return false
		}
	}

	m.conns = append(m.conns, abstractConn{
		ServiceID: id,
		Conn:      conn,
	})

	return true
}

// Remove is used to remove a connection by service id from the muxer.
func (m *Muxer) Remove(id string) grpc.ClientConnInterface {
	m.connLock.Lock()
	defer m.connLock.Unlock()

	for i, c := range m.conns {
		if c.ServiceID == id {
			m.conns = append(m.conns[:i], m.conns[i+1:]...)
			return c.Conn
		}
	}

	return nil
}

// Invoke performs a unary RPC and returns after the response is received
// into reply.
func (m *Muxer) Invoke(ctx context.Context, method string, args interface{}, reply interface{}, opts ...grpc.CallOption) error {
	return m.do(func(c abstractConn) error {
		return c.Conn.Invoke(ctx, method, args, reply, opts...)
	})
}

// NewStream begins a streaming RPC.
func (m *Muxer) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (stream grpc.ClientStream, err error) {
	err = m.do(func(c abstractConn) error {
		stream, err = c.Conn.NewStream(ctx, desc, method, opts...)
		return err
	})

	return
}

func (m *Muxer) do(fn func(abstractConn) error) error {
	m.connLock.RLock()
	defer m.connLock.RUnlock()

	if len(m.conns) == 0 {
		return errors.New("no grpc client connection")
	}

	switch m.Streagy {
	case RoundRobin:
		idx := int(m.lastIdx.Add(1)) % len(m.conns)
		return fn(m.conns[idx])
	case Random:
		idx := rand.Intn(len(m.conns))
		return fn(m.conns[idx])
	}
	return nil
}
