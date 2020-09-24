package main

import (
	"os"

	"github.com/dictyBase/modware-user/commands"
	"github.com/dictyBase/modware-user/validate"

	"github.com/urfave/cli"
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
			Name:   "load-users",
			Usage:  "load dictybase users(colleagues) into the backend",
			Action: commands.LoadUser,
			Before: validate.ValidateLoad,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "remote-path, rp",
					Usage: "full path(relative to the bucket) of s3 object which will be download",
					Value: "import/users.tar.gz",
				},
				cli.StringFlag{
					Name:   "s3-server",
					Usage:  "S3 server endpoint",
					Value:  "minio",
					EnvVar: "MINIO_SERVICE_HOST",
				},
				cli.StringFlag{
					Name:   "s3-server-port",
					Usage:  "S3 server port",
					EnvVar: "MINIO_SERVICE_PORT",
				},
				cli.StringFlag{
					Name:  "s3-bucket",
					Usage: "S3 bucket where the import data is kept",
					Value: "dictybase",
				},
				cli.StringFlag{
					Name:   "access-key, akey",
					EnvVar: "S3_ACCESS_KEY",
					Usage:  "access key for S3 server, required based on command run",
				},
				cli.StringFlag{
					Name:   "secret-key, skey",
					EnvVar: "S3_SECRET_KEY",
					Usage:  "secret key for S3 server, required based on command run",
				},
				cli.StringFlag{
					Name:   "user-grpc-host",
					EnvVar: "USER_API_SERVICE_HOST",
					Usage:  "grpc host address for user service",
					Value:  "user-api",
				},
				cli.StringFlag{
					Name:   "user-grpc-port",
					EnvVar: "USER_API_SERVICE_PORT",
					Usage:  "grpc port for user service",
				},
				cli.StringFlag{
					Name:  "data-file",
					Value: "users.csv",
					Usage: "file containing user data that is present in the bucket",
				},
			},
		},
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
					Value:  "nats",
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
					Name:   "dictyuser-pass",
					EnvVar: "DICTYUSER_PASSWORD",
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
					Value:  "dictycontent-backend",
					EnvVar: "DICTYCONTENT_BACKEND_SERVICE_HOST",
					Usage:  "dictyuser database host",
				},
				cli.StringFlag{
					Name:   "dictyuser-port",
					EnvVar: "DICTYCONTENT_BACKEND_SERVICE_PORT",
					Usage:  "dictyuser database port",
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
					Name:   "dictyuser-pass",
					EnvVar: "DICTYUSER_PASSWORD",
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
					Value:  "dictycontent-backend",
					EnvVar: "DICTYCONTENT_BACKEND_SERVICE_HOST",
					Usage:  "dictyuser database host",
				},
				cli.StringFlag{
					Name:   "dictyuser-port",
					EnvVar: "DICTYCONTENT_BACKEND_SERVICE_PORT",
					Usage:  "dictyuser database port",
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
					Name:   "dictyuser-pass",
					EnvVar: "DICTYUSER_PASSWORD",
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
					Value:  "dictycontent-backend",
					EnvVar: "DICTYCONTENT_BACKEND_SERVICE_HOST",
					Usage:  "dictyuser database host",
				},
				cli.StringFlag{
					Name:   "dictyuser-port",
					EnvVar: "DICTYCONTENT_BACKEND_SERVICE_PORT",
					Usage:  "dictyuser database port",
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
