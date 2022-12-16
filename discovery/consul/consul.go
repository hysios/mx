package consul

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/hysios/mx/discovery"
)

func NewConsulDiscovery() discovery.ServiceDiscover {
	c := &consulDiscovery{}
	go c.Run()
	return c
}

type consulDiscovery struct {
	Namespace string

	interval time.Duration
	config   *api.Config
	cli      *api.Client
	closefn  context.CancelFunc
	ctx      context.Context
	joinCh   chan *discovery.ServiceDesc
	leaveCh  chan *discovery.ServiceDesc
	shadow   map[string]*discovery.ServiceDesc
}

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

	if c.shadow == nil {
		c.shadow = make(map[string]*discovery.ServiceDesc)
	}

	if c.interval == 0 {
		c.interval = time.Second * 5
	}

	// create joinch and leavech
	if c.joinCh == nil {
		c.joinCh = make(chan *discovery.ServiceDesc, 10)
	}

	if c.leaveCh == nil {
		c.leaveCh = make(chan *discovery.ServiceDesc, 10)
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
				newServices   = c.buildSerives(services)
				adds, _, dels = c.diffServices(newServices)
			)

			log.Printf("adds %v, dels %v", adds, dels)
			for _, srv := range adds {
				desc := &discovery.ServiceDesc{
					Service:    srv,
					Protocol:   "",
					ResolveURI: c.resolveSerivceUri(srv),
				}
				if len(c.joinCh) < cap(c.leaveCh) {
					c.joinCh <- desc
					c.shadow[srv] = desc
				}
			}

			for _, srv := range dels {
				if len(c.leaveCh) < cap(c.leaveCh) {
					c.leaveCh <- &discovery.ServiceDesc{
						Service:    srv,
						Protocol:   "",
						ResolveURI: c.resolveSerivceUri(srv),
					}
					delete(c.shadow, srv)
				}
			}

		case <-c.ctx.Done():
			return c.ctx.Err()
		}
	}
}

func (c *consulDiscovery) resolveSerivceUri(service string) string {
	return fmt.Sprintf("consul://%s/%s", c.config.Address, service)
}

func (c *consulDiscovery) buildSerives(srvs map[string]*api.AgentService) map[string]bool {
	services := make(map[string]bool)
	for _, srv := range srvs {
		services[srv.Service] = true
	}

	return services
}

func (c *consulDiscovery) diffServices(services map[string]bool) (adds []string, updates []string, dels []string) {
	for srv := range services {
		if _, ok := c.shadow[srv]; !ok {
			adds = append(adds, srv)
		}
	}

	for srv := range c.shadow {
		if _, ok := services[srv]; !ok {
			dels = append(dels, srv)
		}
	}

	return
}

func (c *consulDiscovery) Close() error {
	c.closefn()

	return nil
}

func (c *consulDiscovery) DiscoveryJoin() chan *discovery.ServiceDesc {
	c.init()

	return c.joinCh
}

func (c *consulDiscovery) DiscoveryLeave() chan *discovery.ServiceDesc {
	c.init()

	return c.leaveCh
}

func init() {
	// register consul discovery
	discovery.Registry("consul", NewConsulDiscovery)
}
