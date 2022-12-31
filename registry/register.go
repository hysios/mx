package registry

type ServiceDiscover interface {
	Notify() chan RegistryMessage
}

type Agent interface {
	Register(desc ServiceDesc) error
	Deregister(serviceID string) error
	Lookup(serviceName string, optfns ...LookupOptionFunc) ([]ServiceDesc, bool)
}

type ServiceKind int

type ServiceDesc struct {
	ID        string
	Kind      ServiceKind
	Service   string
	TargetURI string
	Protocol  string
	Address   string
	Namespace string
}

var (
	DefaultAgent Agent
)

func RegistryProvider(name string, ctor func() Provider) {
	Default.providerRegistry.Register(name, ctor)
}

func LookupProvider(name string) (ctor func() Provider, ok bool) {
	return Default.providerRegistry.Lookup(name)
}
