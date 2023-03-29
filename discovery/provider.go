package discovery

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

var Namespace = "mx"

func SetNamespace(ns string) {
	Namespace = ns
}
