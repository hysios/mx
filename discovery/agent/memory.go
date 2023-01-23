package agent

import (
	"github.com/hysios/mx/discovery"
)

// memory is a simple in-memory implementation of the registry.Agent interface.
type memory struct {
	services map[string]discovery.ServiceDesc
}

// MemoryAgent returns a new Memory instance.
func MemoryAgent() *memory {
	return &memory{
		services: make(map[string]discovery.ServiceDesc),
	}
}

// Register registers a new service.
func (m *memory) Register(desc discovery.ServiceDesc) error {
	m.services[desc.ID] = desc
	return nil
}

// Deregister deregisters a service.
func (m *memory) Deregister(serviceID string) error {
	delete(m.services, serviceID)
	return nil
}

// Lookup looks up a service.
func (m *memory) Lookup(serviceName string, optfns ...discovery.LookupOptionFunc) ([]discovery.ServiceDesc, bool) {
	var services []discovery.ServiceDesc
	for _, desc := range m.services {
		if desc.Service == serviceName {
			services = append(services, desc)
		}
	}
	return services, len(services) > 0
}
