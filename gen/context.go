package gen

import (
	"fmt"
	"io"
	"strings"
)

type ctxCtor func(baseCtx Context) (Context, error)

type Context interface {
	Scan()
	Vars() map[string]interface{}
	With(ctx Context) Context
	Value(key string) interface{}
}

type BaseContext struct {
	vars map[string]interface{}
}

type GofileContext struct {
	Context
	PkgName   string
	GoImports [][2]string
}

type GomodContext struct {
	Context
	ModulePackage string
	GoVersion     string
	GoRequires    [][2]string
}

type ProtofileContext struct {
	Context
	Package     string
	FullPackage string
	Options     []*ProtoOption
	Services    []*ProtoService
	Messages    []*ProtoMessage
	Enums       []*ProtoEnum
}

type ServiceContext struct {
	GofileContext
	ServiceName string
	Methods     []Method
}

type Method struct {
	Name       string
	HttpMethod string
	InputArgs  []Type
	OutputArgs []Type
}

type Type struct {
	Module string
	Define string
	IsPtr  bool
	Name   string
}

func (typ *Type) Type() string {
	if typ.Module == "" {
		return typ.ptr(typ.Define)
	}

	return typ.ptr(fmt.Sprintf("%s.%s", typ.Module, typ.Define))
}

func (typ *Type) ptr(t string) string {

	if typ.IsPtr {
		return fmt.Sprintf("*%s", t)
	}

	return t
}

// implements Context
func (*BaseContext) Scan() {}

func (*BaseContext) With(ctx Context) Context {
	return ctx
}

func (bctx *BaseContext) Vars() map[string]interface{} {
	return bctx.vars
}

func (ctx *BaseContext) Value(key string) interface{} {
	return ctx.vars[key]
}

// implements Context
func (gctx *GofileContext) With(ctx Context) Context {
	gctx.Context = ctx
	return gctx
}

// implements Context
func (pctx *ProtofileContext) With(ctx Context) Context {
	pctx.Context = ctx
	return pctx
}

// implements Context
func (sctx *ServiceContext) With(ctx Context) Context {
	sctx.Context = ctx
	return sctx
}

// ServiceStruct is a struct that contains a list of files and a list of
// file type contexts. It is used to generate a service.
func (ctx *ServiceContext) ServiceStruct() string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "type %s struct {\n", ctx.ServiceName)
	fmt.Fprintf(&sb, "\tpb.Unimplemented%sServer\n", ctx.ServiceName)
	sb.WriteString("}\n")
	return sb.String()
}

func (ctx *ServiceContext) ServiceImplements() string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "func (s *%s) ", ctx.ServiceName)
	for _, m := range ctx.Methods {
		sb.WriteString(m.Name)
		sb.WriteString("(")
		for i, arg := range m.InputArgs {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(arg.Name)
			sb.WriteString(" ")
			sb.WriteString(arg.Type())
		}
		sb.WriteString(") (")
		for i, arg := range m.OutputArgs {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(arg.Name)
			sb.WriteString(" ")
			sb.WriteString(arg.Type())
		}
		sb.WriteString(") {\n")
		sb.WriteString("\t// TODO: implement\n")
		sb.WriteString("\treturn\n")
		sb.WriteString("}\n")
	}
	return sb.String()
}

func (pctx *ProtofileContext) ProtoImports() string {
	return "// imports "
}

func (pctx *ProtofileContext) GatewayOptions() string {
	var sb strings.Builder

	for _, opt := range pctx.Options {
		fmt.Fprintf(&sb, "%s\n", opt.String())
	}

	return sb.String()
}

func (pctx *ProtofileContext) ProtoServices() []*ProtoService {
	return pctx.Services
}

func (pctx *ProtofileContext) ProtoMessages() []*ProtoMessage {
	return pctx.Messages
}

func (pctx *ProtofileContext) ProtoEnums() []*ProtoEnum {
	return pctx.Enums
}

type ProtoService struct {
	ServiceName string
	Methods     []*ProtoMethod
}

func (psctx *ProtoService) String() string {
	var sb strings.Builder

	Fprintf(&sb, "service %s {\n", psctx.ServiceName)
	Indent(func() {
		for _, m := range psctx.Methods {
			Fprintf(&sb, "rpc %s (%s) returns (%s) {\n", m.Method, m.Input.RefType(), m.Output.RefType())
			for _, opt := range m.Options {
				Fprintf(&sb, "%s\n", opt.String())
			}

			WriteString(&sb, "}\n")
		}
	})
	WriteString(&sb, "}\n")

	return sb.String()
}

type ProtoMethod struct {
	Method  string
	Input   *ProtoMessage
	Output  *ProtoMessage
	Options []*ProtoOption
}

type ProtoMessage struct {
	Stream      bool
	MessageName string
	Fields      []*ProtoField
}

func (pm *ProtoMessage) String() string {
	var sb strings.Builder

	if pm.Stream {
		sb.WriteString("stream ")
	}

	fmt.Fprintf(&sb, "message %s {\n", pm.MessageName)
	for idx, field := range pm.Fields {
		fmt.Fprintf(&sb, "\t%s %s = %d;\n", field.FieldType, field.FieldName, idx+1)
	}
	sb.WriteString("}\n")

	return sb.String()
}

func (pm *ProtoMessage) RefType() string {
	var sb strings.Builder

	if pm.Stream {
		sb.WriteString("stream ")
	}

	fmt.Fprintf(&sb, "%s", pm.MessageName)

	return sb.String()
}

type ProtoField struct {
	FieldName string
	FieldType string
}

type ProtoOption struct {
	OptionName  string
	OptionValue ProtoOptionValue
}

func (po *ProtoOption) String() string {
	var sb strings.Builder
	Fprintf(&sb, "option (%s) = {\n", po.OptionName)
	Indent(func() {
		for _, opt := range po.OptionValue.Values {
			Fprintf(&sb, "%s: %s\n", opt.Key, opt.Val)
		}
	})
	WriteString(&sb, "};")
	return sb.String()
}

type ProtoOptionValue struct {
	Values []OptionValue
}

type OptionValue struct {
	Key     string
	Val     string
	Comment string
}

type ProtoEnum struct {
}

func (modctx *GomodContext) Requires() []*GoRequire {
	var requires = make([]*GoRequire, 0)
	for _, require := range modctx.GoRequires {
		requires = append(requires, &GoRequire{
			PkgName: require[0],
			Version: require[1],
		})
	}

	return requires
}

type GoRequire struct {
	PkgName string
	Version string
	// Comment string
}

type GoImport struct {
	PkgName string
	Alias   string
}

func (req *GoRequire) String() string {
	return fmt.Sprintf("\t%s %s", req.PkgName, req.Version) //, "//"+req.Comment)
}

// Imports GofileContext Imports method
func (gctx *GofileContext) Imports() []*GoImport {
	var imports = make([]*GoImport, 0)
	for _, impl := range gctx.GoImports {
		imports = append(imports, &GoImport{
			PkgName: impl[0],
			Alias:   impl[1],
		})
	}

	return imports
}

func (imp *GoImport) String() string {
	if imp.Alias == "" {
		return fmt.Sprintf("\t\"%s\"", imp.PkgName)
	} else {
		return fmt.Sprintf("\t%s \"%s\"", imp.Alias, imp.PkgName)
	}
}

var indent int

func Fprintf(w io.Writer, format string, a ...interface{}) {
	_, _ = io.WriteString(w, strings.Repeat("\t", indent))
	fmt.Fprintf(w, format, a...)
}

func WriteString(w io.Writer, s string) {
	_, _ = io.WriteString(w, strings.Repeat("\t", indent))
	io.WriteString(w, s)
}

func Indent(fn ...func()) {
	indent++
	if len(fn) > 0 {
		fn[0]()
		Unindent()
	}
}

func Unindent() {
	indent--
}
