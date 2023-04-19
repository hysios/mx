package consul

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/consul/api"
	"github.com/hysios/mx"
	"github.com/hysios/mx/discovery"
	"github.com/hysios/mx/discovery/agent"
	"github.com/hysios/mx/logger"
	"go.uber.org/zap"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

func NewConsulProvider() discovery.ServiceDiscover {
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
	msgch       chan discovery.RegistryMessage
	shadow      map[string]discovery.ServiceDesc
	resolverURI resolverURI
}

type resolver struct {
	config *api.Config
}

func (r *resolver) consulResolverURI(srv *api.AgentService) string {
	return fmt.Sprintf("consul://%s/%s", r.config.Address, srv.Service)
}

func (r *resolver) normalResolveURI(srv *api.AgentService) string {
	target, ok := srv.Meta["targetURI"]
	if !ok {
		return fmt.Sprintf("%s:%d", srv.Address, srv.Port)
	}

	return target
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
		c.shadow = make(map[string]discovery.ServiceDesc)
	}

	if c.interval == 0 {
		c.interval = time.Second * 5
	}

	// create msg channel
	if c.msgch == nil {
		c.msgch = make(chan discovery.RegistryMessage, 10)
	}

	return nil
}

// namespace
func (c *consulDiscovery) namespace() string {
	if c.Namespace == "" {
		return discovery.Namespace
	}

	return c.Namespace
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
			services, err := c.filterServices(agent, discovery.WithServiceType(mx.ServerType))
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
				desc := discovery.ServiceDesc{
					ID:        id,
					Service:   services[id].Service,
					Address:   services[id].Address,
					Namespace: services[id].Meta["namespace"],
					TargetURI: c.resolverURI(services[id]),
					Group:     services[id].Meta["group"],
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
					c.msgch <- discovery.RegistryMessage{
						Method: discovery.ServiceJoin,
						Desc:   desc,
					}
					c.shadow[id] = desc
				}
			}

			for _, id := range dels {
				if len(c.msgch) < cap(c.msgch) {
					c.msgch <- discovery.RegistryMessage{
						Method: discovery.ServiceLeave,
						Desc: discovery.ServiceDesc{
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

func (c *consulDiscovery) filterServices(agent *api.Agent, optfn ...discovery.LookupOptionFunc) (map[string]*api.AgentService, error) {
	var (
		opts = discovery.LookupOption{
			Namespace: c.namespace(),
		}
	)
	for _, fn := range optfn {
		fn(&opts)
	}

	services, err := agent.ServicesWithFilterOpts("", nil)

	filterd := make(map[string]*api.AgentService)
	for _, srv := range services {
		if !opts.MatchNamespace(srv.Meta["namespace"]) {
			continue
		}

		if opts.MatchServiceType(srv.Meta["service_type"]) {
			filterd[srv.ID] = srv
		}
	}

	return filterd, err
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

func (c *consulDiscovery) Notify() chan discovery.RegistryMessage {
	c.init()

	return c.msgch
}

func init() {
	// register consul discovery
	discovery.RegistryProvider("consul", func() discovery.Provider {
		return &provider{}
	})

	agent.SetDefaultAgent(NewConsulAgent())
}

type provider struct {
}

func (p *provider) Discover() discovery.ServiceDiscover {
	return NewConsulProvider()
}

func (p *provider) Agent() discovery.Agent {
	return NewConsulAgent()
}
