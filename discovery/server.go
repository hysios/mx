package discovery

import (
	"context"

	"github.com/hysios/mx/utils"
)

var Default = &ServiceDiscovery{}

type ServiceDiscovery struct {
	discoveryFns     []func(desc RegistryMessage)
	closefn          context.CancelFunc
	providerRegistry utils.Registry[Provider]
}

func (discovery *ServiceDiscovery) Discovery(discovry func(desc RegistryMessage)) {
	discovery.init()

	discovery.discoveryFns = append(discovery.discoveryFns, discovry)
}

func (discovery *ServiceDiscovery) Close() error {
	if discovery.closefn != nil {
		discovery.closefn()
	}

	return nil
}

func (discovery *ServiceDiscovery) Start(ctx context.Context) error {
	discovery.init()
	ctx, cancel := context.WithCancel(ctx)
	discovery.closefn = cancel

	return discovery.run(ctx)
}

func (discovery *ServiceDiscovery) init() {
}

func (discovery *ServiceDiscovery) run(ctx context.Context) error {
	var ch = make(chan RegistryMessage, 10)

	discovery.providerRegistry.Range(func(name string, ctor utils.Ctor[Provider]) {
		srvDiscover := ctor().Discover()
		go func() {
			for {
				select {
				case desc := <-srvDiscover.Notify():
					ch <- desc
				case <-ctx.Done():
					return
				}
			}
		}()
	})

	for {
		select {
		case desc := <-ch:
			discovery.dispatch(desc)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (discovery *ServiceDiscovery) dispatch(desc RegistryMessage) {
	for _, fn := range discovery.discoveryFns {
		fn(desc)
	}
}
