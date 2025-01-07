package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hysios/mx/discovery/agent"
	_ "github.com/hysios/mx/discovery/provider/consul"
	"github.com/hysios/mx/gateway"
	"github.com/hysios/mx/logger"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func main() {
	app := &cli.App{
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
			{
				Name:  "config",
				Usage: "config a microservices gateway",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "provider",
						Usage: "config provider, example: consul",
						Value: "consul",
					},
				},
				Subcommands: []*cli.Command{
					{
						Name:  "set",
						Usage: "set a config item",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "key",
								Usage: "config key, example: gateway.addr",
								Aliases: []string{
									"k",
								},
							},
							&cli.StringFlag{
								Name:  "type",
								Usage: "config value type",
								Value: "string",
							},
						},
						Action: func(ctx *cli.Context) error {
							cfg, err := agent.Config(nil)
							if err != nil {
								return err
							}

							ss := strings.SplitN(ctx.String("key"), "=", 2)
							if len(ss) != 2 {
								return cli.Exit("invalid key format, example: key=value", 1)
							}
							key, val := ss[0], ss[1]

							switch ctx.String("type") {
							case "string":
								if _, err := cfg.Set(key, val); err != nil {
									setError(key, err)
								}
							case "int":
								i, err := strconv.Atoi(val)
								if err != nil {
									return cli.Exit("invalid int value", 1)
								}
								if _, err := cfg.Set(key, i); err != nil {
									setError(key, err)
								}
							case "bool":
								b, err := strconv.ParseBool(val)
								if err != nil {
									return cli.Exit("invalid bool value", 1)
								}
								if _, err := cfg.Set(key, b); err != nil {
									setError(key, err)
								}
							case "float":
								f, err := strconv.ParseFloat(val, 64)
								if err != nil {
									return cli.Exit("invalid float value", 1)
								}
								if _, err := cfg.Set(key, f); err != nil {
									setError(key, err)
								}
							case "duration":
								d, err := time.ParseDuration(val)
								if err != nil {
									return cli.Exit("invalid duration value", 1)
								}
								if _, err := cfg.Set(key, d); err != nil {
									setError(key, err)
								}
							case "time":
								t, err := time.Parse("2006-01-02 15:04:05", val)
								if err != nil {
									return cli.Exit("invalid time value", 1)
								}
								if _, err := cfg.Set(key, t.UnixMilli()); err != nil {
									setError(key, err)
								}
							// case "stringSlice":
							// 	cfg.Set(ctx.String("key"), ctx.StringSlice("value"))
							// case "intSlice":
							// 	cfg.Set(ctx.String("key"), ctx.IntSlice("value"))
							// case "floatSlice":
							// 	cfg.Set(ctx.String("key"), ctx.Float64Slice("value"))
							default:
								return cli.Exit("invalid type", 1)
							}

							return nil
						},
					},
					{
						Name:  "get",
						Usage: "get a config item",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "key",
								Usage: "config key",
							},
							&cli.BoolFlag{
								Name:  "quite",
								Usage: "quite mode",
							},
						},
						Action: func(ctx *cli.Context) error {
							cfg, err := agent.Config(nil)
							if err != nil {
								return err
							}

							var val, ok = cfg.Get(ctx.String("key"))
							if !ok {
								return cli.Exit("key not found", 1)
							}

							if ctx.Bool("quite") {
								fmt.Println(val.Data())
								return nil
							}

							fmt.Printf("key: %s => %v\n", ctx.String("key"), val.Data())
							return nil
						},
					},
					{
						Name:  "cat",
						Usage: "cat a config file",
						Action: func(ctx *cli.Context) error {
							cfg, err := agent.Config(nil)
							if err != nil {
								return err
							}

							all := cfg.All()
							fmt.Println(all.MustJSON())
							return nil
						},
					},
					&cli.Command{
						Name: "put",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "data",
								Usage: "config data",
							},
						},
						Action: func(ctx *cli.Context) error {
							cfg, err := agent.Config(nil)
							if err != nil {
								return err
							}

							var dec *json.Decoder
							// is data has preifx @, read file
							if strings.HasPrefix(ctx.String("data"), "@") {
								file := ctx.String("data")[1:]
								f, err := os.OpenFile(file, os.O_RDONLY, 0644)
								if err != nil {
									return err
								}
								defer f.Close()
								dec = json.NewDecoder(f)

							} else {
								dec = json.NewDecoder(strings.NewReader(ctx.String("data")))
							}

							var data map[string]interface{}
							if err := dec.Decode(&data); err != nil {
								return err
							}

							cfg.Update(data)
							return nil
						},
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func setError(key string, err error) {
	if err != nil {
		cli.Exit(fmt.Sprintf("set key %s error %v", key, err), 1)
	}
}

func LogError(err error) {
	if err != nil {
		logger.Cli.Error("run command failed", zap.Error(err))
	}
}

func init() {
	logger.SetLogger(zap.NewExample())
}
