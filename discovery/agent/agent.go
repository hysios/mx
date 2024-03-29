package agent

import (
	"errors"
	"net/url"
	"strconv"

	"github.com/hysios/mx/config"
	"github.com/hysios/mx/config/provider/redis"
	"github.com/hysios/mx/discovery"
	"github.com/hysios/mx/logger"
	"github.com/hysios/mx/server"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

var Default discovery.Agent = MemoryAgent()

func Register(desc discovery.ServiceDesc) error {
	if Default == nil {
		return errors.New("discovery agent is not set")
	}

	return Default.Register(desc)
}

func Deregister(serviceID string) error {
	if Default == nil {
		return errors.New("discovery agent is not set")
	}

	return Default.Deregister(serviceID)
}

var (
	_config     *config.Config
	defaultDesc = []discovery.ServiceDesc{
		{
			ID:        "",
			TargetURI: "redis://127.0.0.1:6379/mx.config",
			Type:      "mx.config",
		},
	}
)

func defaultServiceDesc() discovery.ServiceDesc {
	return discovery.ServiceDesc{
		ID:        "",
		TargetURI: "redis://127.0.0.1:6379/" + discovery.Namespace + ".config",
		Type:      "mx.config",
	}
}

func configName() string {
	return discovery.Namespace + ".config"
}

func Config(defaults map[string]interface{}) (*config.Config, error) {
	if _config != nil {
		return _config, nil
	}

	if Default == nil {
		return nil, errors.New("discovery agent is not set")
	}

	descs, ok := Default.Lookup(configName(), discovery.WithServiceType("config_provider"))
	if !ok {
		// return nil, errors.New("mx.Config service not found")
		descs = []discovery.ServiceDesc{
			defaultServiceDesc(),
		}
	}

	if len(descs) == 0 {
		// return nil, errors.New("mx.Config service not found")
		descs = []discovery.ServiceDesc{
			defaultServiceDesc(),
		}
	}

	getpass := func(u *url.URL) string {
		pass, _ := u.User.Password()
		return pass
	}

	getrdb := func(u *url.URL) int {
		db := u.Query().Get("db")
		if len(db) == 0 {
			return 0
		}

		i, err := strconv.Atoi(db)
		if err != nil {
			return 0
		}
		return i
	}

	buildProviders := func() ([]config.ConfigProvider, error) {
		var (
			providers []config.ConfigProvider
			errs      error
		)
		for _, desc := range descs {
			desc.Type = "mx.config"
			u, err := url.Parse(desc.TargetURI)
			if err != nil {
				continue
			}

			switch u.Scheme {
			case "etcd":
				// providers = append(providers, config.NewEtcdProvider(desc))
			case "redis":
				provider, err := redis.NewRedisProvider(&redis.RedisOption{
					Addr:     u.Host,
					Password: getpass(u),
					DB:       getrdb(u),
					Key:      u.Path,
				})
				if err != nil {
					errs = multierr.Append(errs, err)
					continue
				}
				providers = append(providers, provider)
			}
		}

		return providers, errs
	}

	providers, err := buildProviders()
	if err != nil {
		return nil, err
	}

	_config = config.NewConfig(defaults, providers...)
	return _config, nil
}

func SetDefaultAgent(agent discovery.Agent) {
	Default = agent
}

func RegisterServer(server *server.Server) error {
	go func() {
		<-server.AddrCh()
		for _, desc := range server.ServiceDescs() {
			if err := Register(desc); err != nil {
				logger.Logger.Warn("register service failed", zap.Any("service", desc), zap.Error(err))
			}
		}
	}()

	return nil
}

type lazyProvider struct {
	provider  config.ConfigProvider
	providers []config.ConfigProvider
}
