package discovery

import (
	"errors"

	"github.com/hysios/mx/registry"
)

type ServiceDiscover interface {
	DiscoveryJoin() chan *ServiceDesc
	DiscoveryLeave() chan *ServiceDesc
}

type ServiceDesc struct {
	Service    string
	Protocol   string
	ResolveURI string
}

var (
	discoverRegistry = registry.Registry[ServiceDiscover]{}
)

func Registry(name string, ctor func() ServiceDiscover) {
	discoverRegistry.Register(name, ctor)
}

func Lookup(name string) (ctor func() ServiceDiscover, ok bool) {
	return discoverRegistry.Lookup(name)
}

func Open(name string) (ServiceDiscover, error) {
	ctor, ok := Lookup(name)
	if !ok {
		return nil, errors.New("not found service discover")
	}

	return ctor(), nil
}

func Range(fn func(name string, ctor func() ServiceDiscover)) {
	discoverRegistry.Range(func(_name string, ctor registry.Ctor[ServiceDiscover]) {
		fn(_name, ctor)
	})
}
