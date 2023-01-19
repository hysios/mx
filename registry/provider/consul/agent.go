package consul

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"syscall"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/hysios/mx/registry"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/runtime/protoimpl"
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

	var meta = map[string]string{
		"service_type": desc.Type,
	}

	if desc.FileDescriptor != nil {
		b, err := c.marshalFiledescriptor(desc.FileDescriptor)
		if err != nil {
			return err
		}

		if desc.FileDescriptorKey == "" {
			desc.FileDescriptorKey = desc.FileDescriptor.Path()
		}

		meta["file_descriptor_key"] = desc.FileDescriptorKey
		if _, err := c.cli.KV().Put(&api.KVPair{
			Key:   fmt.Sprintf("mx/registry/protofile/%s", desc.FileDescriptorKey),
			Value: b,
		}, nil); err != nil {
			return err
		}
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

func (c *consulAgent) marshalFiledescriptor(desc protoreflect.FileDescriptor) (b []byte, err error) {
	descProto := protodesc.ToFileDescriptorProto(desc)
	b, err = proto.MarshalOptions{AllowPartial: true, Deterministic: true}.Marshal(descProto)
	return
}

func (c *consulAgent) getFileDescriptor(key string) (desc protoreflect.FileDescriptor, err error) {
	var (
		pair *api.KVPair
	)

	pair, _, err = c.cli.KV().Get(fmt.Sprintf("mx/registry/protofile/%s", key), nil)
	if err != nil {
		return
	}

	if pair == nil {
		err = fmt.Errorf("file descriptor not found: %s", key)
		return
	}

	out := protoimpl.DescBuilder{
		GoPackagePath: reflect.TypeOf(struct{}{}).PkgPath(),
		RawDescriptor: pair.Value,
	}.Build()

	return out.File, nil
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
		if service.Meta["service_type"] == "grpc_server" {
			continue
		}
		var filedescriptor protoreflect.FileDescriptor
		if service.Meta["file_descriptor_key"] != "" {
			if desc, err := c.getFileDescriptor(service.Meta["file_descriptor_key"]); err == nil {
				filedescriptor = desc
			}
		}

		descs = append(descs, registry.ServiceDesc{
			ID:                service.ID,
			Service:           service.Service,
			Namespace:         service.Namespace,
			TargetURI:         c.resolverURI(service),
			Address:           fmt.Sprintf("%s:%d", service.Address, service.Port),
			FileDescriptorKey: service.Meta["file_descriptor_key"],
			FileDescriptor:    filedescriptor,
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
