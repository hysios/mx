package mx

import (
	"context"

	"github.com/hysios/mx/discovery"
)

type ServiceDiscovery struct {
	ch           chan *discovery.ServiceDesc
	discoveryFns []func(desc *discovery.ServiceDesc)
	closefn      context.CancelFunc
}

func (discover *ServiceDiscovery) Discovery(discovry func(desc *discovery.ServiceDesc)) {
	discover.init()

	discover.discoveryFns = append(discover.discoveryFns, discovry)
}

func (discover *ServiceDiscovery) Close() error {
	close(discover.ch)

	return nil
}

func (discover *ServiceDiscovery) init() {
	if discover.ch == nil {
		discover.ch = make(chan *discovery.ServiceDesc)
	}

	if discover.closefn == nil {
		ctx, cancel := context.WithCancel(context.Background())
		discover.closefn = cancel

		go discover.run(ctx)
	}
}

func (discover *ServiceDiscovery) run(ctx context.Context) error {
	discovery.Range(func(name string, ctor func() discovery.ServiceDiscover) {
		srvDiscover := ctor()
		go func() {
			joinch := srvDiscover.DiscoveryJoin()
			for v := range joinch {
				discover.ch <- v
			}
		}()
	})

	for {
		select {
		case desc := <-discover.ch:
			for _, fn := range discover.discoveryFns {
				fn(desc)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
