package main

import "github.com/urfave/cli/v2"

func provisionCmd() *cli.Command {
	return &cli.Command{
		Name:  "provision",
		Usage: "provision a microservices service",
		Action: func(c *cli.Context) error {
			return nil
		},
	}
}
