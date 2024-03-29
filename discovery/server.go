package discovery

import (
	"context"
	"time"

	"github.com/hysios/mx/utils"
)

var Default = &ServiceDiscovery{}

type ServiceDiscovery struct {
	Namespace        string
	discoveryFns     []func(desc RegistryMessage)
	closefn          context.CancelFunc
	providerRegistry utils.Registry[Provider]
	queue            []RegistryMessage
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
	if discovery.Namespace == "" {
		discovery.Namespace = Namespace
	}

	if discovery.queue == nil {
		discovery.queue = make([]RegistryMessage, 0)
	}
}

func (discovery *ServiceDiscovery) run(ctx context.Context) error {
	var ch = make(chan RegistryMessage, 100)

	discovery.providerRegistry.Range(func(name string, ctor utils.Ctor[Provider]) {
		srvDiscover := ctor().Discover()
		go func() {
			for {
				select {
				case desc := <-srvDiscover.Notify():
					if len(ch) < cap(ch) {
						ch <- desc
					} else {
						discovery.queue = append(discovery.queue, desc)
					}
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
		case <-time.After(10 * time.Second):
			for _, desc := range discovery.queue {
				discovery.dispatch(desc)
			}
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
