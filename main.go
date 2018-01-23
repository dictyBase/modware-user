package main

import (
	"fmt"
	"os"

	"github.com/dictyBase/modware-user/commands"

	"gopkg.in/urfave/cli.v1"
)

func main() {
	app := cli.NewApp()
	app.Name = "run"
	app.Usage = "starts the modware-user microservice with HTTP and grpc backends"
	app.Version = "1.0.0"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "dictyuser-pass",
			EnvVar: "DICTYUSER_PASS",
			Usage:  "dictyuser database password",
		},
		cli.StringFlag{
			Name:   "dictyuser-db",
			EnvVar: "DICTYUSER_DB",
			Usage:  "dictyuser database name",
		},
		cli.StringFlag{
			Name:   "dictyuser-user",
			EnvVar: "DICTYUSER_USER",
			Usage:  "dictyuser database user",
		},
		cli.StringFlag{
			Name:   "dictyuser-host",
			Value:  "dictyuser-backend",
			EnvVar: "DICTYUSER_BACKEND_SERVICE_HOST",
			Usage:  "dictyuser database host",
		},
		cli.StringFlag{
			Name:   "dictyuser-port",
			EnvVar: "DICTYUSER_BACKEND_SERVICE_PORT",
			Usage:  "dictyuser database port",
		},
		cli.StringFlag{
			Name:  "port",
			Usage: "tcp port at which the servers will be available",
			Value: "9596",
		},
	}
	app.Before = validateArgs
	app.Action = commands.RunServer
	app.Run(os.Args)
}

func validateArgs(c *cli.Context) error {
	for _, p := range []string{"chado-pass", "chado-db", "chado-user"} {
		if !c.IsSet(p) {
			return cli.NewExitError(
				fmt.Sprintf("argument %s is missing", p),
				2,
			)
		}
	}
	return nil
}
