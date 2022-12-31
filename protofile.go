package mx

type FileDesc struct {
	Name     string
	PkgPath  string
	RawDesc  []byte
	Services []*ServiceDesc
	Messages []*MessageDesc
}

type ServiceDesc struct {
	ServiceName string
}

type MessageDesc struct {
	Name string

	Fields []struct {
		Name string
		Type string
	}
}
