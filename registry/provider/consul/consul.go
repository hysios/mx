package consul

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/consul/api"
	"github.com/hysios/mx/logger"
	"github.com/hysios/mx/registry"
	"github.com/hysios/mx/registry/agent"
	"go.uber.org/zap"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
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

func (c *consulDiscovery) getFileDescriptor(key string) (desc protoreflect.FileDescriptor, err error) {
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

	out := &descriptorpb.FileDescriptorProto{}
	err = proto.Unmarshal(pair.Value, out)
	if err != nil {
		return
	}

	desc, err = protodesc.NewFile(out, protoregistry.GlobalFiles)
	return
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

				if serviceType, ok := services[id].Meta["service_type"]; ok {
					desc.Type = serviceType
					if serviceType != "grpc_server" {
						continue
					}
				}

				if services[id].Meta["file_descriptor_key"] != "" {
					desc.FileDescriptorKey = services[id].Meta["file_descriptor_key"]
					filedescriptor, err := c.getFileDescriptor(desc.FileDescriptorKey)
					if err != nil {
						logger.Logger.Error("getFileDescriptor", zap.Error(err))
						continue
					}
					desc.FileDescriptor = filedescriptor
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
							Type:      "",
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
