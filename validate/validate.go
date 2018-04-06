package validate

import (
	"fmt"

	cli "gopkg.in/urfave/cli.v1"
)

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
