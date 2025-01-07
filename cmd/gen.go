package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/hysios/mx/gen"
	icli "github.com/hysios/mx/internal/cli"
	"github.com/hysios/mx/internal/gen/template"
	"github.com/hysios/mx/utils"
	"github.com/urfave/cli/v2"
	"golang.org/x/mod/modfile"
)

func getModuleName(dir string) (string, error) {
	gomodPath := filepath.Join(dir, "go.mod")
	content, err := os.ReadFile(gomodPath)
	if err != nil {
		return "", err
	}

	f, err := modfile.Parse("go.mod", content, nil)
	if err != nil {
		return "", err
	}

	return f.Module.Mod.Path, nil
}

func genSubCmds() []*cli.Command {
	return []*cli.Command{
		{
			Name:  "service",
			Usage: "generate a new service",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "name",
					Usage:    "service package name",
					Required: true,
				},
				&cli.StringFlag{
					Name:     "pkg-name",
					Usage:    "service package name",
					Required: true,
				},
				// service-name
				&cli.StringFlag{
					Name:       "service-name",
					Usage:      "service name",
					HasBeenSet: true,
					Action: func(c *cli.Context, v string) error {
						if v == "" {
							v = utils.CamelCase(c.String("name")) + "Service"
							_ = c.Set("service-name", v)
						}
						return nil
					},
				},
				// with protobuf file
				&cli.BoolFlag{
					Name:  "gen-proto",
					Usage: "generate a new service with protobuf file",
				},
				// overwrite
				&cli.BoolFlag{
					Name:  "overwrite",
					Usage: "overwrite existing files",
				},
				// output directory
				&cli.StringFlag{
					Name:  "output",
					Usage: "output directory",
					Aliases: []string{
						"o",
					},
				},
				// services directory
				&cli.StringFlag{
					Name:  "services-dir",
					Usage: "services directory",
					Value: "services",
				},
				// verbose
				&cli.BoolFlag{
					Name:  "verbose",
					Usage: "verbose output",
					Aliases: []string{
						"v",
					},
				},
				methodFlag(),
				crudFlag(),
			},
			Action: func(c *cli.Context) error {
				service := template.Service
				service.AddVariable("Name", c.String("name"))
				service.AddVariable("FullPackage", c.String("pkg-name"))
				service.AddVariable("ServiceName", c.String("service-name"))
				service.AddVariable("ServiceDesc", c.String("service-name")+"_ServiceDesc")
				service.AddVariable("ProtoPkgName", "pb")
				service.AddVariable("FileProto", "File_proto_"+c.String("name")+"_proto")
				service.AddVariable("Methods", parseMethods(c.StringSlice("method")))

				LogError(service.Gen(&gen.Output{
					Directory: c.String("output"),
					Verbose:   c.Bool("verbose"),
					Overwrite: c.Bool("overwrite"),
				}))
				return nil
			},
		},
		{
			Name:  "gateway",
			Usage: "generate a new gateway",
			Flags: []cli.Flag{
				// output directory
				&cli.StringFlag{
					Name:  "output",
					Usage: "output directory",
					Aliases: []string{
						"o",
					},
				},
				&cli.StringFlag{
					Name:     "pkg-name",
					Usage:    "gateway package name",
					Required: true,
				},
				&cli.StringFlag{
					Name:  "namespace",
					Usage: "services namespace",
				},
				&cli.BoolFlag{
					Name:  "verbose",
					Usage: "verbose output",
					Aliases: []string{
						"v",
					},
				},
				&cli.BoolFlag{
					Name:  "overwrite",
					Usage: "overwrite existing files",
				},
			},
			Action: func(c *cli.Context) error {
				gateway := template.Gateway
				gateway.AddVariable("FullPackage", c.String("pkg-name"))
				ns := c.String("namespace")
				if ns == "" {
					p, _ := filepath.Abs(c.String("output"))
					ns = getGatewayParent(p)
				}
				gateway.AddVariable("Namespace", ns)

				LogError(gateway.Gen(&gen.Output{
					Directory: c.String("output"),
					Verbose:   c.Bool("verbose"),
					Overwrite: c.Bool("overwrite"),
				}))
				return nil
			},
		},
		{
			Name:  "add",
			Usage: "add a new service to existing project",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "name",
					Usage:    "service package name",
					Required: true,
				},
				&cli.StringFlag{
					Name:  "pkg-name",
					Usage: "service package name",
				},
				&cli.StringFlag{
					Name:       "service-name",
					Usage:      "service name",
					HasBeenSet: true,
					Action: func(c *cli.Context, v string) error {
						if v == "" {
							v = utils.CamelCase(c.String("name")) + "Service"
							_ = c.Set("service-name", v)
						}
						return nil
					},
				},
				&cli.BoolFlag{
					Name:  "gen-proto",
					Usage: "generate a new service with protobuf file",
				},
				&cli.BoolFlag{
					Name:    "overwrite",
					Usage:   "overwrite existing files",
					Aliases: []string{"y"},
				},
				&cli.StringFlag{
					Name:    "output",
					Usage:   "output directory",
					Aliases: []string{"o"},
				},
				methodFlag(),
				crudFlag(),
				&cli.StringFlag{
					Name:  "namespace",
					Usage: "service namespace",
				},
			},
			Action: func(c *cli.Context) error {
				// Get gateway namespace from gateway dir if not specified
				ns := c.String("namespace")
				if ns == "" {
					p, _ := os.Getwd()
					ns = getGatewayParent(p)
				}

				if c.String("pkg-name") == "" {
					getPkgName(c)
				}

				// Setup service template
				service := template.AddService
				service.AddVariable("Name", c.String("name"))
				service.AddVariable("FullPackage", c.String("pkg-name"))
				service.AddVariable("ServiceName", c.String("service-name"))
				service.AddVariable("ServiceDesc", c.String("service-name")+"_ServiceDesc")
				service.AddVariable("ProtoPkgName", "pb")
				service.AddVariable("FileProto", "File_proto_"+c.String("name")+"_proto")
				service.AddVariable("Methods", parseMethods(c.StringSlice("method")))
				service.AddVariable("Namespace", ns)

				// Generate service files
				output := c.String("output")
				if output == "" {
					curDir, _ := os.Getwd()
					output = curDir
				}

				LogError(service.Gen(&gen.Output{
					Directory: output,
					Verbose:   c.Bool("verbose"),
					Overwrite: c.Bool("overwrite"),
				}))

				return nil
			},
		},
	}
}

// getPkgName
func getPkgName(c *cli.Context) (string, error) {
	// Get current directory
	curDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Get module name from go.mod
	modName, err := getModuleName(curDir)
	if err != nil {
		return "", fmt.Errorf("failed to get module name: %v", err)
	}
	c.Set("pkg-name", modName)
	return modName, nil
}

func methodFlag() cli.Flag {
	return &cli.StringSliceFlag{
		Name:  "method",
		Usage: "add a method to the service",
		Aliases: []string{
			"m",
		},
		Action: func(c *cli.Context, v []string) error {
			methods := c.StringSlice("method")
			for _, m := range methods {
				err := icli.CheckMethod(m)
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func crudFlag() cli.Flag {
	return &cli.StringFlag{
		Name:  "crud",
		Usage: "add a crud method to the service",
		Aliases: []string{
			"c",
		},
		Value: "lcgur",
		Action: func(c *cli.Context, v string) error {
			// list
			// create
			// get
			// update
			// remove
			return nil
		},
	}
}

func parseMethods(methods []string) []*icli.Method {
	var meths = make([]*icli.Method, 0)
	for _, meth := range methods {
		method, err := icli.ParseMethod(meth)
		if err != nil {
			LogError(err)
		}
		meths = append(meths, method)
	}

	return meths
}

func getGatewayParent(output string) string {
	if output == "" {
		output, _ = os.Getwd()
	}

	return strings.TrimPrefix(path.Base(strings.TrimSuffix(output, "/gateway")), "/")
}

// Helper function to update gateway imports
func updateGatewayImports(gatewayDir string, pkgName string) error {
	// This function would update the gateway's imports to include the new service
	// Implementation depends on your gateway structure
	return nil
}
