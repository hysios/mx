package mx

import "context"

type ServiceDiscovery struct {
	ch           chan *ServiceDesc
	discoveryFns []func(desc *ServiceDesc)
	closefn      context.CancelFunc
}

type ServiceDesc struct {
	Name string
}

func (discover *ServiceDiscovery) Discovery(discovry func(desc *ServiceDesc)) {
	discover.init()

	discover.discoveryFns = append(discover.discoveryFns, discovry)
}

func (discover *ServiceDiscovery) Close() error {
	close(discover.ch)

	return nil
}

func (discover *ServiceDiscovery) init() {
	if discover.ch == nil {
		discover.ch = make(chan *ServiceDesc)
	}

	if discover.closefn == nil {
		ctx, cancel := context.WithCancel(context.Background())
		discover.closefn = cancel

		go discover.run(ctx)
	}
}

func (discover *ServiceDiscovery) run(ctx context.Context) error {
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
