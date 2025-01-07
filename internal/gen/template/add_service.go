package template

import (
	"embed"

	"github.com/hysios/mx/gen"
	"github.com/hysios/mx/internal/cli"
	"github.com/hysios/mx/utils"
)

//go:embed addservice addservice/**/*
var addServiceFs embed.FS

var (
	AddService = &gen.FileSystem{
		Root:     "addservice",
		Contents: addServiceFs,
	}
)

func init() {
	AddService.AddFile("services/{{.Name}}/service.go", AddService.MustParse("addservice/service.go.tmpl"))
	AddService.AddFile("proto/{{.Name}}.proto", AddService.MustParse("addservice/service.proto.tmpl"))

	AddService.AddFileTypeContext(".proto", func(baseCtx gen.Context) (gen.Context, error) {
		var (
			messages = make(map[string]*gen.ProtoMessage)
			msgIdx   = make([]string, 0)
		)
		return &gen.ProtofileContext{
			Context:     baseCtx,
			Package:     baseCtx.Value("Name").(string),
			FullPackage: baseCtx.Value("FullPackage").(string) + "/gen/proto",
			Options:     Service.GetProtoOptions(baseCtx),
			Services:    Service.GetProtoServices(baseCtx, messages, msgIdx),
			Messages: func() []*gen.ProtoMessage {
				var msgs []*gen.ProtoMessage
				for _, name := range msgIdx {
					msgs = append(msgs, messages[name])
				}
				return msgs
			}(),
		}, nil
	})

	AddService.AddFileTypeContext(".go", func(baseCtx gen.Context) (gen.Context, error) {
		return &gen.GofileContext{
			Context: baseCtx,
			PkgName: baseCtx.Value("Name").(string),
		}, nil
	})

	AddService.AddFileContext("services/{{.Name}}/service.go", func(baseCtx gen.Context) (gen.Context, error) {
		var (
			pkgpath = baseCtx.Value("FullPackage").(string)
			methods = baseCtx.Value("Methods").([]*cli.Method)
		)
		return &gen.ServiceContext{
			GofileContext: gen.GofileContext{
				Context: baseCtx,
				PkgName: baseCtx.Value("Name").(string),
				GoImports: [][2]string{
					{pkgpath + "/gen/proto", "pb"},
				},
			},
			ServiceName: utils.CamelCase(baseCtx.Value("ServiceName").(string)),
			Methods:     Service.GetServiceMethods(methods),
		}, nil
	})

	AddService.After(runProtogen)
	AddService.After(runImports)
}
