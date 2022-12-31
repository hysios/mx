package consul

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/hysios/mx/logger"
	"github.com/hysios/mx/registry"
	"github.com/hysios/mx/registry/agent"
	"go.uber.org/zap"
)

func NewConsulProvider() registry.ServiceDiscover {
	c := &consulDiscovery{}
	go c.Run()
	return c
}

type consulDiscovery struct {
	Namespace string

	interval    time.Duration
	config      *api.Config
	cli         *api.Client
	closefn     context.CancelFunc
	ctx         context.Context
	msgch       chan registry.RegistryMessage
	shadow      map[string]registry.ServiceDesc
	resolverURI resolverURI
}

type resolver struct {
	config *api.Config
}

func (r *resolver) consulResolverURI(srv *api.AgentService) string {
	return fmt.Sprintf("consul://%s/%s", r.config.Address, srv.Service)
}

func (r *resolver) normalResolveURI(srv *api.AgentService) string {
	return fmt.Sprintf("%s:%d", srv.Address, srv.Port)
}

type resolverURI func(*api.AgentService) string

// init
func (c *consulDiscovery) init() error {
	if c.config == nil {
		c.config = api.DefaultConfig()
	}

	if c.cli == nil {
		cli, err := api.NewClient(c.config)
		if err != nil {
			return nil
		}
		c.cli = cli
	}

	if c.resolverURI == nil {
		r := resolver{config: c.config}
		c.resolverURI = r.normalResolveURI
	}

	if c.shadow == nil {
		c.shadow = make(map[string]registry.ServiceDesc)
	}

	if c.interval == 0 {
		c.interval = time.Second * 5
	}

	// create msg channel
	if c.msgch == nil {
		c.msgch = make(chan registry.RegistryMessage, 10)
	}

	return nil
}

func (c *consulDiscovery) Run() error {
	c.init()

	var (
		agent       = c.cli.Agent()
		ctx, cancel = context.WithCancel(context.Background())
		tick        = time.NewTicker(c.interval)
	)

	c.closefn = cancel
	c.ctx = ctx
	// loop interval 5s to filter services
	for {
		select {
		case <-tick.C:
			services, err := agent.ServicesWithFilterOpts("", &api.QueryOptions{
				Namespace: c.Namespace,
			})

			if err != nil {
				continue
			}

			var (
				adds, _, dels = c.diffServices(services)
			)

			if len(adds) != 0 || len(dels) != 0 {
				logger.Logger.Debug("change services", zap.Strings("adds", adds), zap.Strings("dels", dels))
			}

			for _, id := range adds {
				desc := registry.ServiceDesc{
					ID:        id,
					Service:   services[id].Service,
					Address:   services[id].Address,
					Namespace: services[id].Namespace,
					TargetURI: c.resolverURI(services[id]),
				}

				if len(c.msgch) < cap(c.msgch) {
					c.msgch <- registry.RegistryMessage{
						Method: registry.ServiceJoin,
						Desc:   desc,
					}
					c.shadow[id] = desc
				}
			}

			for _, id := range dels {
				if len(c.msgch) < cap(c.msgch) {
					c.msgch <- registry.RegistryMessage{
						Method: registry.ServiceLeave,
						Desc: registry.ServiceDesc{
							ID:        id,
							Service:   c.shadow[id].Service,
							Protocol:  "",
							Address:   c.shadow[id].Address,
							Namespace: c.shadow[id].Namespace,
						},
					}

					delete(c.shadow, id)
				}
			}

		case <-c.ctx.Done():
			return c.ctx.Err()
		}
	}
}

func (c *consulDiscovery) consulResolverURI(srv *api.AgentService) string {
	return fmt.Sprintf("consul://%s/%s", c.config.Address, srv.Service)
}

func (c *consulDiscovery) normalResolveURI(srv *api.AgentService) string {
	return fmt.Sprintf("%s:%d", srv.Address, srv.Port)
}

func (c *consulDiscovery) diffServices(services map[string]*api.AgentService) (adds []string, updates []string, dels []string) {
	for srvId := range services {
		if _, ok := c.shadow[srvId]; !ok {
			adds = append(adds, srvId)
		}
	}

	for srvId := range c.shadow {
		if _, ok := services[srvId]; !ok {
			dels = append(dels, srvId)
		}
	}

	return
}

func (c *consulDiscovery) Close() error {
	c.closefn()

	return nil
}

func (c *consulDiscovery) Notify() chan registry.RegistryMessage {
	c.init()

	return c.msgch
}

func init() {
	// register consul discovery
	registry.RegistryProvider("consul", func() registry.Provider {
		return &provider{}
	})

	agent.SetDefaultAgent(NewConsulAgent())
}

type provider struct {
}

func (p *provider) Discover() registry.ServiceDiscover {
	return NewConsulProvider()
}

func (p *provider) Agent() registry.Agent {
	return NewConsulAgent()
}
