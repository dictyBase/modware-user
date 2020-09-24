package validate

import (
	"fmt"

	"github.com/urfave/cli"
)

func ValidateReplyArgs(c *cli.Context) error {
	for _, p := range []string{
		"user-grpc-host",
		"user-grpc-port",
		"messaging-host",
		"messaging-port",
	} {
		if len(c.String(p)) == 0 {
			return cli.NewExitError(
				fmt.Sprintf("argument %s is missing", p),
				2,
			)
		}
	}
	return nil
}

func ValidateArgs(c *cli.Context) error {
	for _, p := range []string{
		"dictyuser-pass",
		"dictyuser-db",
		"dictyuser-user",
		"user-api-http-host",
	} {
		if len(c.String(p)) == 0 {
			return cli.NewExitError(
				fmt.Sprintf("argument %s is missing", p),
				2,
			)
		}
	}
	return nil
}

func validateS3Args(c *cli.Context) error {
	for _, p := range []string{"s3-server", "s3-bucket", "access-key", "secret-key"} {
		if len(c.String(p)) == 0 {
			return cli.NewExitError(fmt.Sprintf("argument %s is missing", p), 2)
		}
	}
	return nil
}

func ValidateLoad(c *cli.Context) error {
	if err := validateS3Args(c); err != nil {
		return err
	}
	for _, p := range []string{"user-grpc-host", "user-grpc-port"} {
		if len(c.String(p)) == 0 {
			return cli.NewExitError(fmt.Sprintf("argument %s is missing", p), 2)
		}
	}
	return nil
}
