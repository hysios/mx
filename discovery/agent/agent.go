package agent

import (
	"errors"

	"github.com/hysios/mx/discovery"
	"github.com/hysios/mx/service"
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

func SetDefaultAgent(agent discovery.Agent) {
	Default = agent
}

func RegisterServer(server *service.Server) error {
	go func() {
		<-server.AddrCh()
		Register(server.ServiceDesc())
	}()

	return nil
}
