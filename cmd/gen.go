package main

import (
	"github.com/hysios/mx/gen"
	icli "github.com/hysios/mx/internal/cli"
	"github.com/hysios/mx/internal/gen/template"
	"github.com/hysios/mx/utils"
	"github.com/urfave/cli/v2"
)

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

				LogError(gateway.Gen(&gen.Output{
					Directory: c.String("output"),
					Verbose:   c.Bool("verbose"),
					Overwrite: c.Bool("overwrite"),
				}))
				return nil
			},
		},
	}
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
