package template

import (
	"embed"

	"github.com/hysios/mx/gen"
)

//go:embed gateway
var gatewayFs embed.FS

var (
	Gateway = &gen.FileSystem{
		Contents: gatewayFs,
		Root:     "gateway",
	}
)

func init() {
	Gateway.AddFileContext("go.mod", func(baseCtx gen.Context) (gen.Context, error) {
		return &gen.GomodContext{
			Context:       baseCtx,
			ModulePackage: baseCtx.Value("FullPackage").(string),
			GoVersion:     "1.19",
			GoRequires: [][2]string{
				{" github.com/hysios/mx", "v0.0.0-20221231104819-7f2485626a5f"},
			},
		}, nil
	})

	Gateway.AddFileContext("main.go", func(baseCtx gen.Context) (gen.Context, error) {
		return &gen.GofileContext{
			Context: baseCtx,
			PkgName: "main",
			GoImports: [][2]string{
				{"github.com/hysios/mx", ""},
			},
		}, nil
	})

	Gateway.After(runModtidy)
}
