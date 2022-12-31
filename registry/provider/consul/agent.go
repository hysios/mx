package consul

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/hysios/mx/registry"
)

type AgentOption struct {
	Config *api.Config
}

type AgentOptionFunc func(*AgentOption)

func WithConfig(cfg *api.Config) AgentOptionFunc {
	return func(opt *AgentOption) {
		opt.Config = cfg
	}
}

func NewConsulAgent(optFns ...AgentOptionFunc) registry.Agent {
	var (
		opt = AgentOption{}
	)
	for _, fn := range optFns {
		fn(&opt)
	}

	if opt.Config == nil {
		opt.Config = api.DefaultConfig()
	}

	agent := &consulAgent{
		opts: opt,
	}

	cli, err := api.NewClient(opt.Config)
	if err != nil {
		panic(err)
	}

	if agent.resolverURI == nil {
		r := resolver{config: opt.Config}
		agent.resolverURI = r.normalResolveURI
	}

	ctx, cancel := context.WithCancel(context.Background())
	agent.ctx = ctx
	agent.closefn = cancel
	agent.cli = cli
	return agent
}

type consulAgent struct {
	cli         *api.Client
	opts        AgentOption
	ctx         context.Context
	closefn     context.CancelFunc
	resolverURI resolverURI
}

func (c *consulAgent) Register(desc registry.ServiceDesc) error {
	var (
		agent            = c.cli.Agent()
		host, _port, err = net.SplitHostPort(desc.Address)
		port, _          = strconv.Atoi(_port)
	)
	if err != nil {
		return err
	}

	if host == "::" {
		host = "127.0.0.1"
	}

	if err = agent.ServiceRegister(&api.AgentServiceRegistration{
		ID:        desc.ID,
		Name:      desc.Service,
		Port:      port,
		Address:   host,
		Namespace: desc.Namespace,
		Check: &api.AgentServiceCheck{
			CheckID:                        "service:" + desc.ID,
			TTL:                            "30s",
			Timeout:                        "45s",
			DeregisterCriticalServiceAfter: "60s",
		},
	}); err != nil {
		return err
	}

	_ = c.Update(desc.ID, "service registered", api.HealthPassing)

	go c.teardown()
	go c.updateSchedule(desc)

	return nil
}

func (c *consulAgent) updateSchedule(desc registry.ServiceDesc) error {
	var (
		agent = c.cli.Agent()
		tick  = time.NewTicker(15 * time.Second)
	)

	for {
		select {
		case t := <-tick.C:

			if err := agent.UpdateTTL("service:"+desc.ID, t.Format("2006-01-02 15:04:05"), api.HealthPassing); err != nil {
				log.Printf("update ttl failed: %v", err)
				continue
			}
		case <-c.ctx.Done():
			tick.Stop()
			return c.Deregister(desc.ID)
			// return c.ctx.Err()
		}
	}

	return nil
}

func (c consulAgent) Update(serviceID, output, status string) error {
	var (
		agent = c.cli.Agent()
	)
	if err := agent.UpdateTTL("service:"+serviceID, output, status); err != nil {
		return err
	}
	return nil
}

func (c *consulAgent) Deregister(serviceID string) error {
	var (
		agent = c.cli.Agent()
	)
	if err := agent.ServiceDeregister(serviceID); err != nil {
		return err
	}
	return nil
}

func (c *consulAgent) Lookup(serviceName string, optfns ...registry.LookupOptionFunc) ([]registry.ServiceDesc, bool) {
	var (
		agent = c.cli.Agent()
		opt   = registry.LookupOption{}
	)

	for _, fn := range optfns {
		fn(&opt)
	}

	services, err := agent.ServicesWithFilterOpts(fmt.Sprintf("Service==%s", serviceName), &api.QueryOptions{
		Namespace: opt.Namespace,
	})
	if err != nil {
		return nil, false
	}

	var descs []registry.ServiceDesc
	for _, service := range services {
		descs = append(descs, registry.ServiceDesc{
			ID:        service.ID,
			Service:   service.Service,
			Namespace: service.Namespace,
			TargetURI: c.resolverURI(service),
			Address:   fmt.Sprintf("%s:%d", service.Address, service.Port),
		})
	}

	return descs, true
}

// setupTeardown
func (c *consulAgent) teardown() {
	// on os signal ctrl+c or kill
	//
	// 1. deregister service
	// 2. close consul client
	// 3. close context
	// 4. close cancel function

	var ch = make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-ch

		if c.closefn != nil {
			c.closefn()
		}
	}()

}
