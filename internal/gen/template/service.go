package template

import (
	"embed"
	"strconv"
	"strings"

	"github.com/hysios/mx/gen"
	"github.com/hysios/mx/internal/cli"
	"github.com/hysios/mx/utils"
)

//go:embed service service/**/*
var serviceFs embed.FS

var (
	Service = &gen.FileSystem{
		Root:     "service",
		Contents: serviceFs,
	}
)

func init() {
	Service.AddFile("services/{{.Name}}/service.go", Service.MustParse("service/service.go.tmpl"))
	Service.AddFile("proto/{{.Name}}.proto", Service.MustParse("service/service.proto.tmpl"))

	Service.AddFileTypeContext(".proto", func(baseCtx gen.Context) (gen.Context, error) {
		var (
			messages = make(map[string]*gen.ProtoMessage)
			msgIdx   = make([]string, 0)
		)
		return &gen.ProtofileContext{
			Context:     baseCtx,
			Package:     baseCtx.Value("Name").(string),
			FullPackage: baseCtx.Value("FullPackage").(string) + "/gen/proto",
			Options: func() []*gen.ProtoOption {
				var options []*gen.ProtoOption
				// option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_swagger) = {
				// 	info: {version: "1.0"}
				// 	external_docs: {
				// 	  url: "https://github.com/hysios/mx_example/proto"
				// 	  description: "mx framework api demo"
				// 	}
				// 	schemes: [
				// 	  HTTP,
				// 	  HTTPS
				// 	];
				//   };
				options = append(options, &gen.ProtoOption{
					OptionName: "grpc.gateway.protoc_gen_openapiv2.options.openapiv2_swagger",
					OptionValue: gen.ProtoOptionValue{
						Values: []gen.OptionValue{
							{
								Key: "info",
								Val: "{version: \"1.0\"}",
							},
							{
								Key: "external_docs",
								Val: "{url: \"" + baseCtx.Value("FullPackage").(string) + "/gen/proto" + "\", description: \"mx framework api demo\"}",
							},
							{
								Key: "schemes",
								Val: "[HTTP, HTTPS];",
							},
						},
					},
				})
				return options
			}(),
			Services: func() []*gen.ProtoService {
				var (
					services    []*gen.ProtoService
					serviceName = baseCtx.Value("ServiceName").(string)
					srv         = &gen.ProtoService{
						ServiceName: utils.CamelCase(serviceName),
					}
					methods []*gen.ProtoMethod
				)

				services = append(services, srv)
				for _, m := range baseCtx.Value("Methods").([]*cli.Method) {
					inputMessage := &gen.ProtoMessage{
						MessageName: utils.CamelCase(m.Input.Name),
						Fields: func() []*gen.ProtoField {
							var fields []*gen.ProtoField
							for _, f := range m.Input.Fields {
								fields = append(fields, &gen.ProtoField{
									FieldName: f.Name,
									FieldType: f.Type,
								})
							}
							return fields
						}(),
					}

					ouputMessage := &gen.ProtoMessage{
						MessageName: utils.CamelCase(m.Output.Name),
						Fields: func() []*gen.ProtoField {
							var fields []*gen.ProtoField
							for _, f := range m.Output.Fields {
								fields = append(fields, &gen.ProtoField{
									FieldName: f.Name,
									FieldType: f.Type,
								})
							}
							return fields
						}(),
					}

					methods = append(methods, &gen.ProtoMethod{
						Method: m.Name,
						Input:  inputMessage,
						Output: ouputMessage,
						Options: func() []*gen.ProtoOption {
							var (
								options    []*gen.ProtoOption
								httpMethod = m.HttpMethod
							)

							// option (google.api.http) = {
							// 	// Route to this method from GET requests to /api/v1/path
							// 	get: "/api/hello"
							//   };

							if httpMethod == "" {
								httpMethod = "get"
							}

							if m.Path != "" {

								options = append(options, &gen.ProtoOption{
									OptionName: "google.api.http",
									OptionValue: gen.ProtoOptionValue{
										Values: []gen.OptionValue{
											{
												Key: utils.Lower(m.HttpMethod),
												Val: quote(m.Path),
											},
										},
									},
								})
							}

							// option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_operation) = {
							//		summary: "Hello Method"
							//		tags: "HelloService"
							// };
							options = append(options, &gen.ProtoOption{
								OptionName: "grpc.gateway.protoc_gen_openapiv2.options.openapiv2_operation",
								OptionValue: gen.ProtoOptionValue{
									Values: []gen.OptionValue{
										{
											Key: "summary",
											Val: quote(m.Name),
										},
										{
											Key: "tags",
											Val: quote(serviceName),
										},
									},
								},
							})

							return options
						}(),
					})

					if _, ok := messages[inputMessage.MessageName]; !ok {
						messages[inputMessage.MessageName] = inputMessage
						msgIdx = append(msgIdx, inputMessage.MessageName)
					}

					if _, ok := messages[ouputMessage.MessageName]; !ok {
						messages[ouputMessage.MessageName] = ouputMessage
						msgIdx = append(msgIdx, ouputMessage.MessageName)
					}
				}
				srv.Methods = methods

				return services
			}(),
			Messages: func() []*gen.ProtoMessage {
				var msgs []*gen.ProtoMessage

				for _, name := range msgIdx {
					msgs = append(msgs, messages[name])
				}

				return msgs
			}(),
		}, nil
	})

	Service.AddFileTypeContext(".go", func(baseCtx gen.Context) (gen.Context, error) {
		return &gen.GofileContext{
			Context: baseCtx,
			PkgName: baseCtx.Value("Name").(string),
		}, nil
	})

	Service.AddFileContext("services/{{.Name}}/service.go", func(baseCtx gen.Context) (gen.Context, error) {
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
			Methods: func() []gen.Method {
				var ms []gen.Method
				for _, m := range methods {
					ms = append(ms, gen.Method{
						Name: utils.CamelCase(m.Name),
						InputArgs: func() []gen.Type {
							var ts []gen.Type

							// first is context.Context
							ts = append(ts, gen.Type{
								Module: "context",
								Define: "Context",
								Name:   "ctx",
							})

							ts = append(ts, gen.Type{
								Module: "pb",
								Define: m.Input.Name,
								IsPtr:  true,
								Name:   "req",
							})

							return ts
						}(),
						OutputArgs: func() []gen.Type {
							var ts []gen.Type
							ts = append(ts, gen.Type{
								Module: "pb",
								Define: m.Output.Name,
								IsPtr:  true,
								Name:   "resp",
							})

							// add error
							ts = append(ts, gen.Type{
								Define: "error",
								Name:   "err",
							})
							return ts
						}(),
						HttpMethod: m.HttpMethod,
					})
				}
				return ms
			}(),
		}, nil
	})

	Service.AddFileContext("go.mod", func(baseCtx gen.Context) (gen.Context, error) {
		return &gen.GomodContext{
			Context:       baseCtx,
			ModulePackage: baseCtx.Value("FullPackage").(string),
			GoVersion:     "1.19",
			GoRequires: [][2]string{
				{"google.golang.org/protobuf", "v1.27.1"},
				{"google.golang.org/grpc", "v1.40.0"},
			},
		}, nil
	})

	Service.AddFileContext("main.go", func(baseCtx gen.Context) (gen.Context, error) {
		var (
			pkgpath = baseCtx.Value("FullPackage").(string)
			name    = baseCtx.Value("Name").(string)
			goctx   = &gen.GofileContext{
				Context: baseCtx,
				PkgName: baseCtx.Value("Name").(string),
				GoImports: [][2]string{
					{pkgpath + "/gen/proto", "pb"},
					{pkgpath + "/services/" + name, ""},
					{"github.com/hysios/mx/server", ""},
				},
			}
		)

		return goctx, nil
	})

	Service.After(runProtogen)
	Service.After(runModtidy)
	Service.After(runImports)
}

func varname(name string) string {
	if strings.HasSuffix(name, "Request") {
		name = strings.TrimSuffix(name, "Request")
	}
	if strings.HasSuffix(name, "Response") {
		name = strings.TrimSuffix(name, "Response")
	}

	return utils.LowerCamel(name)
}

// ptr returns the pointer type of the given type name.
func ptr(name string) string {
	return "*" + name
}

func quote(s string) string {
	return strconv.Quote(s)
}
