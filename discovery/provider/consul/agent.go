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
	"github.com/hysios/mx/discovery"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type AgentOption struct {
	Config    *api.Config
	Namespace string
}

type AgentOptionFunc func(*AgentOption)

func WithConfig(cfg *api.Config) AgentOptionFunc {
	return func(opt *AgentOption) {
		opt.Config = cfg
	}
}

func WithNamespace(ns string) AgentOptionFunc {
	return func(opt *AgentOption) {
		opt.Namespace = ns
	}
}

func NewConsulAgent(optFns ...AgentOptionFunc) discovery.Agent {
	var (
		opt = AgentOption{}
	)
	for _, fn := range optFns {
		fn(&opt)
	}

	if opt.Config == nil {
		opt.Config = api.DefaultConfig()
		// opt.Config.Namespace = discovery.Namespace
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
	Namespace   string
	cli         *api.Client
	opts        AgentOption
	ctx         context.Context
	closefn     context.CancelFunc
	resolverURI resolverURI
	pack        discovery.FileDescriptorPacker
}

func (c *consulAgent) namespace() string {
	if c.Namespace == "" {
		c.Namespace = discovery.Namespace
	}

	return c.Namespace
}

func (c *consulAgent) Register(desc discovery.ServiceDesc) error {
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

	var meta = map[string]string{
		"service_type": desc.Type,
	}

	if c.namespace() != "" {
		meta["namespace"] = c.namespace()
	}

	if desc.FileDescriptor != nil {
		b, err := c.pack.Pack(desc.FileDescriptor)
		if err != nil {
			return err
		}

		if desc.FileDescriptorKey == "" {
			desc.FileDescriptorKey = desc.FileDescriptor.Path()
		}

		meta["file_descriptor_key"] = desc.FileDescriptorKey
		if _, err := c.cli.KV().Put(&api.KVPair{
			Key:   fmt.Sprintf("mx/registry/protofile/%s/%s", c.Namespace, desc.FileDescriptorKey),
			Value: b,
		}, nil); err != nil {
			return err
		}
	}

	// add service group to meta
	if desc.Group != "" {
		meta["group"] = desc.Group
	}

	if err = agent.ServiceRegister(&api.AgentServiceRegistration{
		ID:        desc.ID,
		Name:      desc.Service,
		Port:      port,
		Address:   host,
		Meta:      meta,
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

func (c *consulAgent) updateSchedule(desc discovery.ServiceDesc) error {
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

func (c *consulAgent) getFileDescriptor(key string) (desc protoreflect.FileDescriptor, err error) {
	var (
		pair *api.KVPair
	)

	pair, _, err = c.cli.KV().Get(fmt.Sprintf("mx/registry/protofile/%s/%s", c.Namespace, key), nil)
	if err != nil {
		return
	}

	if pair == nil {
		err = fmt.Errorf("file descriptor not found: %s", key)
		return
	}

	return c.pack.Unpack(pair.Value)
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

func (c *consulAgent) Lookup(serviceName string, optfns ...discovery.LookupOptionFunc) ([]discovery.ServiceDesc, bool) {
	var (
		agent = c.cli.Agent()
		opt   = discovery.LookupOption{}
	)

	for _, fn := range optfns {
		fn(&opt)
	}

	services, err := agent.ServicesWithFilterOpts(fmt.Sprintf("Service==%s", serviceName), nil)
	if err != nil {
		return nil, false
	}

	var descs []discovery.ServiceDesc
	for _, service := range services {
		if !opt.MatchServiceType(service.Meta["service_type"]) {
			continue
		}

		if !opt.MatchNamespace(service.Meta["namespace"]) {
			continue
		}

		var filedescriptor protoreflect.FileDescriptor
		if service.Meta["file_descriptor_key"] != "" {
			if desc, err := c.getFileDescriptor(service.Meta["file_descriptor_key"]); err == nil {
				filedescriptor = desc
			}
		}

		descs = append(descs, discovery.ServiceDesc{
			ID:                service.ID,
			Service:           service.Service,
			Namespace:         service.Namespace,
			TargetURI:         c.resolverURI(service),
			Address:           fmt.Sprintf("%s:%d", service.Address, service.Port),
			FileDescriptorKey: service.Meta["file_descriptor_key"],
			FileDescriptor:    filedescriptor,
			Group:             service.Meta["group"],
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
