package protofile

type PkgDesc struct {
	Name    string
	Path    string
	Options []OptionDesc
}

type ServiceDesc struct {
	Name string

	RPCs    []*RPCDesc
	Options []OptionDesc
}

type MessageDesc struct {
	Name   string
	Fields []*FieldDesc
}

type FieldDesc struct {
	Name     string
	Type     string
	Number   string
	Required bool
	Optional bool
	Repeated bool
	Options  []OptionDesc
}

type RPCDesc struct {
	Name    string
	Input   *RequestDesc
	OUtput  *ResponseDesc
	Options []OptionDesc
}

type RequestDesc struct {
	Type    string
	Stream  bool
	Message *MessageDesc
}

func (req *RequestDesc) lazyMessage(vis *Visitor) *MessageDesc {
	if req.Message != nil {
		return req.Message
	}

	vis.addLazy(func() {
		msgdesc, ok := vis.types[req.Type].(*MessageDesc)
		if !ok {
			panic("not found message: " + req.Type)
		}
		req.Message = msgdesc
	})

	return req.Message
}

type ResponseDesc struct {
	Type    string
	Stream  bool
	Message *MessageDesc
}

func (resp *ResponseDesc) lazyMessage(vis *Visitor) *MessageDesc {
	if resp.Message != nil {
		return resp.Message
	}

	vis.addLazy(func() {
		msgdesc, ok := vis.types[resp.Type].(*MessageDesc)
		if !ok {
			panic("not found message: " + resp.Type)
		}
		resp.Message = msgdesc
	})

	return resp.Message
}

type OptionDesc struct {
	Name     string
	Constant string
}
