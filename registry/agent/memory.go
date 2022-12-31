package agent

import (
	"github.com/hysios/mx/registry"
)

// memory is a simple in-memory implementation of the registry.Agent interface.
type memory struct {
	services map[string]registry.ServiceDesc
}

// MemoryAgent returns a new Memory instance.
func MemoryAgent() *memory {
	return &memory{
		services: make(map[string]registry.ServiceDesc),
	}
}

// Register registers a new service.
func (m *memory) Register(desc registry.ServiceDesc) error {
	m.services[desc.ID] = desc
	return nil
}

// Deregister deregisters a service.
func (m *memory) Deregister(serviceID string) error {
	delete(m.services, serviceID)
	return nil
}

// Lookup looks up a service.
func (m *memory) Lookup(serviceName string, optfns ...registry.LookupOptionFunc) ([]registry.ServiceDesc, bool) {
	var services []registry.ServiceDesc
	for _, desc := range m.services {
		if desc.Service == serviceName {
			services = append(services, desc)
		}
	}
	return services, len(services) > 0
}
