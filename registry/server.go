package registry

import (
	"context"

	"github.com/hysios/mx/utils"
)

var Default = &ServiceRegistry{}

type ServiceRegistry struct {
	discoveryFns     []func(desc RegistryMessage)
	closefn          context.CancelFunc
	providerRegistry utils.Registry[Provider]
}

func (registry *ServiceRegistry) Discovery(discovry func(desc RegistryMessage)) {
	registry.init()

	registry.discoveryFns = append(registry.discoveryFns, discovry)
}

func (registry *ServiceRegistry) Close() error {
	if registry.closefn != nil {
		registry.closefn()
	}

	return nil
}

func (registry *ServiceRegistry) Start(ctx context.Context) error {
	registry.init()
	ctx, cancel := context.WithCancel(ctx)
	registry.closefn = cancel

	return registry.run(ctx)
}

func (registry *ServiceRegistry) init() {
}

func (registry *ServiceRegistry) run(ctx context.Context) error {
	var ch = make(chan RegistryMessage, 10)

	registry.providerRegistry.Range(func(name string, ctor utils.Ctor[Provider]) {
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
			registry.dispatch(desc)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (registry *ServiceRegistry) dispatch(desc RegistryMessage) {
	for _, fn := range registry.discoveryFns {
		fn(desc)
	}
}
