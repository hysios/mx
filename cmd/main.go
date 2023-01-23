package main

import (
	"os"

	"github.com/hysios/mx/gateway"
	"github.com/hysios/mx/logger"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func main() {
	(&cli.App{
		Name:  "mx",
		Usage: "mx is a bootstrap tool for microservices gateway",
		Commands: []*cli.Command{
			{
				Name:        "gen",
				Usage:       "generate a microservices stubs",
				Subcommands: genSubCmds(),
			},
			provisionCmd(),
			{
				Name:  "gateway",
				Usage: "run a microservices gateway",
				Aliases: []string{
					"gw",
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "addr",
						Usage: "gateway listen address",
						Value: ":8080",
					},
				},
				Action: func(ctx *cli.Context) error {
					return gateway.New().Serve(ctx.String("addr"))
				},
			},
		},
	}).Run(os.Args)
}

func LogError(err error) {
	if err != nil {
		logger.Cli.Error("run command failed", zap.Error(err))
	}
}

func init() {
	logger.SetLogger(zap.NewExample())
}
