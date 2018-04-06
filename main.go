package main

import (
	"os"

	"github.com/dictyBase/modware-user/commands"
	"github.com/dictyBase/modware-user/validate"

	"gopkg.in/urfave/cli.v1"
)

func main() {
	app := cli.NewApp()
	app.Name = "modware-user"
	app.Usage = "starts the modware-user microservice with HTTP and grpc backends"
	app.Version = "1.0.0"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "log-format",
			Usage: "format of the logging out, either of json or text.",
			Value: "json",
		},
		cli.StringFlag{
			Name:  "log-level",
			Usage: "log level for the application",
			Value: "error",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:   "start-user-reply",
			Usage:  "start the reply messaging(nats) backend for user microservice",
			Action: commands.RunUserReply,
			Before: validate.ValidateReplyArgs,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "user-grpc-host",
					EnvVar: "USER_API_SERVICE_HOST",
					Usage:  "grpc host address for user service",
				},
				cli.StringFlag{
					Name:   "user-grpc-port",
					EnvVar: "USER_API_SERVICE_PORT",
					Usage:  "grpc port for user service",
				},
				cli.StringFlag{
					Name:   "messaging-host",
					EnvVar: "NATS_SERVICE_HOST",
					Usage:  "host address for messaging server",
				},
				cli.StringFlag{
					Name:   "messaging-port",
					EnvVar: "NATS_SERVICE_PORT",
					Usage:  "port for messaging server",
				},
			},
		},
		{
			Name:   "start-role-server",
			Usage:  "starts the modware-role microservice with HTTP and grpc backends",
			Action: commands.RunRoleServer,
			Before: validate.ValidateArgs,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "user-api-http-host",
					EnvVar: "USER_API_HTTP_HOST",
					Usage:  "public hostname serving the http api, by default the default port will be appended to http://localhost",
				},
				cli.StringFlag{
					Name:   "dictycontent-pass",
					EnvVar: "DICTYCONTENT_PASS",
					Usage:  "dictycontent database password",
				},
				cli.StringFlag{
					Name:   "dictycontent-db",
					EnvVar: "DICTYCONTENT_DB",
					Usage:  "dictycontent database name",
				},
				cli.StringFlag{
					Name:   "dictycontent-user",
					EnvVar: "DICTYCONTENT_USER",
					Usage:  "dictycontent database user",
				},
				cli.StringFlag{
					Name:   "dictycontent-host",
					Value:  "dictycontent-backend",
					EnvVar: "DICTYCONTENT_BACKEND_SERVICE_HOST",
					Usage:  "dictycontent database host",
				},
				cli.StringFlag{
					Name:   "dictycontent-port",
					EnvVar: "DICTYCONTENT_BACKEND_SERVICE_PORT",
					Usage:  "dictycontent database port",
				},
				cli.StringFlag{
					Name:  "port",
					Usage: "tcp port at which the role server will be available",
					Value: "9597",
				},
			},
		},
		{
			Name:   "start-permission-server",
			Usage:  "starts the modware-permission microservice with HTTP and grpc backends",
			Action: commands.RunPermissionServer,
			Before: validate.ValidateArgs,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "user-api-http-host",
					EnvVar: "USER_API_HTTP_HOST",
					Usage:  "public hostname serving the http api, by default the default port will be appended to http://localhost",
				},
				cli.StringFlag{
					Name:   "dictycontent-pass",
					EnvVar: "DICTYCONTENT_PASS",
					Usage:  "dictycontent database password",
				},
				cli.StringFlag{
					Name:   "dictycontent-db",
					EnvVar: "DICTYCONTENT_DB",
					Usage:  "dictycontent database name",
				},
				cli.StringFlag{
					Name:   "dictycontent-user",
					EnvVar: "DICTYCONTENT_USER",
					Usage:  "dictycontent database user",
				},
				cli.StringFlag{
					Name:   "dictycontent-host",
					Value:  "dictycontent-backend",
					EnvVar: "DICTYCONTENT_BACKEND_SERVICE_HOST",
					Usage:  "dictycontent database host",
				},
				cli.StringFlag{
					Name:   "dictycontent-port",
					EnvVar: "DICTYCONTENT_BACKEND_SERVICE_PORT",
					Usage:  "dictycontent database port",
				},
				cli.StringFlag{
					Name:  "port",
					Usage: "tcp port at which the user server will be available",
					Value: "9596",
				},
			},
		},
		{
			Name:   "start-user-server",
			Usage:  "starts the modware-user microservice with HTTP and grpc backends",
			Action: commands.RunUserServer,
			Before: validate.ValidateArgs,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "user-api-http-host",
					EnvVar: "USER_API_HTTP_HOST",
					Usage:  "public hostname serving the http api, by default the default port will be appended to http://localhost",
				},
				cli.StringFlag{
					Name:   "dictycontent-pass",
					EnvVar: "DICTYCONTENT_PASS",
					Usage:  "dictycontent database password",
				},
				cli.StringFlag{
					Name:   "dictycontent-db",
					EnvVar: "DICTYCONTENT_DB",
					Usage:  "dictycontent database name",
				},
				cli.StringFlag{
					Name:   "dictycontent-user",
					EnvVar: "DICTYCONTENT_USER",
					Usage:  "dictycontent database user",
				},
				cli.StringFlag{
					Name:   "dictycontent-host",
					Value:  "dictycontent-backend",
					EnvVar: "DICTYCONTENT_BACKEND_SERVICE_HOST",
					Usage:  "dictycontent database host",
				},
				cli.StringFlag{
					Name:   "dictycontent-port",
					EnvVar: "DICTYCONTENT_BACKEND_SERVICE_PORT",
					Usage:  "dictycontent database port",
				},
				cli.StringFlag{
					Name:  "port",
					Usage: "tcp port at which the user server will be available",
					Value: "9596",
				},
			},
		},
	}
	app.Run(os.Args)
}
