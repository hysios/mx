package agent

import (
	"errors"

	"github.com/hysios/mx/registry"
)

var Default registry.Agent = MemoryAgent()

func Register(desc registry.ServiceDesc) error {
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

func SetDefaultAgent(agent registry.Agent) {
	Default = agent
}
