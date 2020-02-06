package testutils

import (
	"fmt"
	"os"

	gnats "github.com/nats-io/go-nats"
)

var natsHost = os.Getenv("NATS_HOST")
var natsPort = os.Getenv("NATS_PORT")
var natsAddr = fmt.Sprintf("nats://%s:%s", natsHost, natsPort)

func CheckNatsEnv() error {
	envs := []string{
		"NATS_HOST",
		"NATS_PORT",
	}
	for _, e := range envs {
		if len(os.Getenv(e)) == 0 {
			return fmt.Errorf("env %s is not set", e)
		}
	}
	return nil
}

type TestNats struct {
	Conn *gnats.Conn
}

func NewTestNatsFromEnv() (*TestNats, error) {
	n := new(TestNats)
	if err := CheckNatsEnv(); err != nil {
		return n, err
	}
	nc, err := gnats.Connect(natsAddr)
	if err != nil {
		return n, err
	}
	n.Conn = nc
	return n, nil
}
