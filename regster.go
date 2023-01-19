package mx

type ServiceRegister interface {
	Register(*srvRegister) error
}

type (
	ServiceMode string
)

const (
	SMClient  ServiceMode = "client"
	SMImpl    ServiceMode = "impl"
	SMDynamic ServiceMode = "dynamic"
)

func (gw *Gateway) registerService(srv *srvRegister) error {
	for _, reg := range gw.registers {
		if err := reg.Register(srv); err != nil {
			return err
		}
	}
	return nil
}

type descriptorRegister struct {
}

func (d *descriptorRegister) Register(srv *srvRegister) error {
	panic("nonimplement")
}
