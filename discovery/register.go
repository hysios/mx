package discovery

import "google.golang.org/protobuf/reflect/protoreflect"

type ServiceDiscover interface {
	Notify() chan RegistryMessage
}

type Agent interface {
	Register(desc ServiceDesc) error
	Deregister(serviceID string) error
	Lookup(serviceName string, optfns ...LookupOptionFunc) ([]ServiceDesc, bool)
}

type ServiceKind int

// ServiceDesc is the service description
type ServiceDesc struct {
	ID                string
	Kind              ServiceKind
	Service           string
	Version           string
	TargetURI         string
	Type              string
	Address           string
	Namespace         string
	Group             string
	FileDescriptorKey string
	FileDescriptor    protoreflect.FileDescriptor
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
