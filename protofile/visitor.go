package protofile

import (
	"github.com/yoheimuta/go-protoparser/v4/parser"
)

type Visitor struct {
	target *Protofile
	syntax Syntax
	ctxs   map[ContextType]any
	lastCt ContextType
	types  map[string]any
	lazy   []func()
}

type Syntax string

const (
	ProtoV2 = "proto2"
	ProtoV3 = "proto3"
)

type ContextType int

const (
	CTPackage ContextType = iota
	CTSerivce
	CTMessage
	CTField
	CTOptions
	CTRPC
	CTFile
)

func NewVisitor(protofile *Protofile) *Visitor {
	return &Visitor{

		target: protofile,
		ctxs:   make(map[ContextType]any),
		types:  make(map[string]any),
	}
}

func (vis *Visitor) VisitComment(_ *parser.Comment) {
	// panic("not implemented") // TODO: Implement
}

func (vis *Visitor) VisitEmptyStatement(_ *parser.EmptyStatement) (next bool) {
	// panic("not implemented") // TODO: Implement
	return true
}

func (vis *Visitor) VisitEnum(_ *parser.Enum) (next bool) {
	return true
}

func (vis *Visitor) VisitEnumField(_ *parser.EnumField) (next bool) {
	return true
}

func (vis *Visitor) VisitExtend(_ *parser.Extend) (next bool) {
	return true
}

func (vis *Visitor) VisitExtensions(_ *parser.Extensions) (next bool) {
	return true
}

func (vis *Visitor) VisitField(field *parser.Field) (next bool) {
	msgdesc := vis.ctxs[CTMessage].(*MessageDesc)

	fielddesc := &FieldDesc{
		Name:     field.FieldName,
		Type:     field.Type,
		Required: field.IsRequired,
		Optional: field.IsOptional,
		Repeated: field.IsRepeated,
		Number:   field.FieldNumber,
		Options:  make([]OptionDesc, 0),
		// Option: OptionDesc{
		// 	Tag: field.FieldTag,
		// },
	}
	msgdesc.Fields = append(msgdesc.Fields, fielddesc)
	vis.setCtx(CTOptions, &fielddesc.Options)
	return true
}

func (vis *Visitor) VisitGroupField(_ *parser.GroupField) (next bool) {
	return true
}

func (vis *Visitor) VisitImport(_ *parser.Import) (next bool) {
	return true
}

func (vis *Visitor) VisitMapField(_ *parser.MapField) (next bool) {
	return true
}

func (vis *Visitor) VisitMessage(message *parser.Message) (next bool) {
	msgdesc := &MessageDesc{
		Name: message.MessageName,
	}

	vis.target.Messages = append(vis.target.Messages, msgdesc)
	vis.types[message.MessageName] = msgdesc

	vis.setCtx(CTMessage, msgdesc)
	return true
}

func (vis *Visitor) VisitOneof(_ *parser.Oneof) (next bool) {
	return true
}

func (vis *Visitor) VisitOneofField(_ *parser.OneofField) (next bool) {
	return true
}

func (vis *Visitor) VisitOption(option *parser.Option) (next bool) {
	switch vis.lastCt {
	case CTOptions:
		options := vis.ctxs[CTOptions].(*[]OptionDesc)

		*options = append(*options, OptionDesc{
			Name:     option.OptionName,
			Constant: option.Constant,
		})
	case CTRPC:

	case CTField:
	case CTSerivce:
		srvdesc := vis.ctxs[CTSerivce].(*ServiceDesc)
		srvdesc.Options = append(srvdesc.Options, OptionDesc{
			Name:     option.OptionName,
			Constant: option.Constant,
		})
	case CTPackage:
		pkgdesc := vis.ctxs[CTPackage].(*PkgDesc)
		pkgdesc.Options = append(pkgdesc.Options, OptionDesc{
			Name:     option.OptionName,
			Constant: option.Constant,
		})
	}
	return true
}

func (vis *Visitor) VisitPackage(pkg *parser.Package) (next bool) {
	pkgdesc := &PkgDesc{
		Name: pkg.Name,
	}

	vis.target.Pkg = pkgdesc
	vis.setCtx(CTPackage, pkgdesc)
	return true
}

func (vis *Visitor) VisitReserved(_ *parser.Reserved) (next bool) {
	return true
}

func (vis *Visitor) VisitRPC(rpc *parser.RPC) (next bool) {
	srvdesc := vis.ctxs[CTSerivce].(*ServiceDesc)

	req := &RequestDesc{
		Type:   rpc.RPCRequest.MessageType,
		Stream: rpc.RPCRequest.IsStream,
	}

	req.lazyMessage(vis)
	resp := &ResponseDesc{
		Type:   rpc.RPCResponse.MessageType,
		Stream: rpc.RPCResponse.IsStream,
	}
	resp.lazyMessage(vis)

	rpcdesc := &RPCDesc{
		Name:   rpc.RPCName,
		Input:  req,
		OUtput: resp,
	}

	rpcdesc.Options = make([]OptionDesc, 0)
	for _, opt := range rpc.Options {
		rpcdesc.Options = append(rpcdesc.Options, OptionDesc{
			Name:     opt.OptionName,
			Constant: opt.Constant,
		})
	}

	srvdesc.RPCs = append(srvdesc.RPCs, rpcdesc)
	vis.setCtx(CTRPC, rpcdesc)

	return true
}

func (vis *Visitor) VisitService(srv *parser.Service) (next bool) {
	srvdesc := &ServiceDesc{
		Name: srv.ServiceName,
	}
	vis.target.Services = append(vis.target.Services, srvdesc)
	vis.setCtx(CTSerivce, srvdesc)
	return true
}

func (vis *Visitor) VisitSyntax(syntax *parser.Syntax) (next bool) {
	switch syntax.Version() {
	case 3:
		vis.syntax = ProtoV3
	case 2:
		vis.syntax = ProtoV2
	default:
		vis.syntax = ProtoV3
	}
	return true
}

// func (vis *Visitor) push(obj any) {
// 	vis.ctxs = append(vis.ctxs, obj)
// }

// func (vis *Visitor) pop() any {
// 	obj := vis.ctxs[len(vis.ctxs)-1]
// 	vis.ctxs = vis.ctxs[:len(vis.ctxs)-1]
// 	return obj
// }

// func (vis *Visitor) wrap(obj any, fn func(obj any)) {
// 	vis.push(obj)
// 	fn(obj)
// 	vis.pop()
// }

func (vis *Visitor) setCtx(ct ContextType, obj any) {
	vis.ctxs[ct] = obj
	vis.lastCt = ct
}

func (vis *Visitor) addLazy(fn func()) {
	vis.lazy = append(vis.lazy, fn)
}

func (vis *Visitor) doLazies() error {
	for _, fn := range vis.lazy {
		fn()
	}
	return nil
}
