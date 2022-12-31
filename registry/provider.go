package registry

const (
	ServiceJoin  = "join"
	ServiceLeave = "leave"
)

type Provider interface {
	Discover() ServiceDiscover
}

type AgentProvider interface {
	Agent() Agent
}

type RegistryMessage struct {
	Method string
	Desc   ServiceDesc
}
