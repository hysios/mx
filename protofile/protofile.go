package protofile

import (
	"github.com/hysios/mx/internal/proto"
)

// Protofile build proto file to mux Handler to call grpc service.
type Protofile struct {
	PkgPath  string
	proto    *proto.Proto
	Pkg      *PkgDesc
	Options  []OptionDesc
	Services []*ServiceDesc
	Messages []*MessageDesc
}

func Parse(b []byte) (*Protofile, error) {
	proto, err := proto.Parse(b)
	if err != nil {
		return nil, err
	}

	var (
		protofile = &Protofile{
			proto: proto,
		}
		vistor = NewVisitor(protofile)
	)

	proto.Accept(vistor)
	vistor.doLazies()

	return protofile, nil
}
