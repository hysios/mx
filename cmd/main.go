package main

import (
	"os"

	"github.com/hysios/mx/logger"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func main() {
	(&cli.App{
		Name:  "mx",
		Usage: "mx is a bootstrap tool for microservices gateway",
		Commands: []*cli.Command{
			&cli.Command{
				Name:        "gen",
				Usage:       "generate a microservices stubs",
				Subcommands: genSubCmds(),
			},
			provisionCmd(),
		},
	}).Run(os.Args)
}

func LogError(err error) {
	if err != nil {
		logger.Cli.Error("run command failed", zap.Error(err))
	}
}
