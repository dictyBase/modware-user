package testutils

import (
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	gnats "github.com/nats-io/go-nats"
	git "gopkg.in/src-d/go-git.v4"
)

var SchemaRepo string = "https://github.com/dictybase-docker/dictyuser-schema"
var natsHost = os.Getenv("NATS_HOST")
var natsPort = os.Getenv("NATS_PORT")
var natsAddr = fmt.Sprintf("nats://%s:%s", natsHost, natsPort)

func CheckPostgresEnv() error {
	envs := []string{
		"POSTGRES_USER",
		"POSTGRES_PASSWORD",
		"POSTGRES_DB",
		"POSTGRES_HOST",
	}
	for _, e := range envs {
		if len(os.Getenv(e)) == 0 {
			return fmt.Errorf("env %s is not set", e)
		}
	}
	return nil
}

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

func TearDownTest(db *sql.DB, t *testing.T) {
	userTbls := []string{"auth_user", "auth_user_info", "auth_user_role"}
	roleTbls := []string{"auth_permission", "auth_role", "auth_role_permission"}
	tbls := append(userTbls, roleTbls...)
	for _, tbl := range tbls {
		_, err := db.Exec(fmt.Sprintf("TRUNCATE %s CASCADE", tbl))
		if err != nil {
			t.Fatalf("unable to truncate table %s %s\n", tbl, err)
		}
	}
}

type TestPostgres struct {
	DB            *sql.DB
	ConnectParams *ConnectParams
}

// ConnectParams are the parameters required for connecting to arangodb
type ConnectParams struct {
	User     string `validate:"required"`
	Password string `validate:"required"`
	Database string `validate:"required"`
	Host     string `validate:"required"`
	Port     string `validate:"required"`
}

func NewTestPostgresFromEnv(isCreate bool) (*TestPostgres, error) {
	pg := new(TestPostgres)
	if err := CheckPostgresEnv(); err != nil {
		return pg, err
	}
	pg.ConnectParams = &ConnectParams{
		User:     os.Getenv("POSTGRES_USER"),
		Password: os.Getenv("POSTGRES_PASSWORD"),
		Host:     os.Getenv("POSTGRES_HOST"),
		Port:     os.Getenv("POSTGRES_PORT"),
		Database: os.Getenv("POSTGRES_DB"),
	}
	dbh, err := getPgxDbHandler(pg.ConnectParams)
	if err != nil {
		return pg, err
	}
	pg.DB = dbh
	if isCreate {
		newDB := RandomString(6, 8)
		_, err = pg.DB.Exec(fmt.Sprintf("CREATE DATABASE %s WITH TEMPLATE %s OWNER %s", newDB, pg.ConnectParams.Database, pg.ConnectParams.User))
		if err != nil {
			return pg, fmt.Errorf("issue creating new db %s", err)
		}
		pg.ConnectParams.Database = newDB
		newDBH, err := getPgxDbHandler(pg.ConnectParams)
		if err != nil {
			return pg, err
		}
		pg.DB = newDBH
	}
	return pg, nil
}

func getPgxDbHandler(cp *ConnectParams) (*sql.DB, error) {
	db := &sql.DB{}
	pgConn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cp.User, cp.Password, cp.Host, cp.Port, cp.Database)
	dbh, err := sql.Open("pgx", pgConn)
	if err != nil {
		return db, err
	}
	return dbh, nil
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
	timeout, err := time.ParseDuration("28s")
	if err != nil {
		return n, err
	}
	t1 := time.Now()
	for {
		if !nc.IsConnected() {
			if time.Since(t1).Seconds() > timeout.Seconds() {
				return n, errors.New("timed out trying to connect to nats server")
			}
			continue
		}
		break
	}
	n.Conn = nc
	return n, nil
}

func CloneDbSchemaRepo(repo string) (string, error) {
	path, err := ioutil.TempDir("", "content")
	if err != nil {
		return path, err
	}
	_, err = git.PlainClone(path, false, &git.CloneOptions{URL: repo})
	return path, err
}

// Generates a random string between a range(min and max) of length
func RandomString(min, max int) string {
	alphanum := []byte("abcdefghijklmnopqrstuvwxyz")
	rand.Seed(time.Now().UTC().UnixNano())
	size := min + rand.Intn(max-min)
	b := make([]byte, size)
	alen := len(alphanum)
	for i := 0; i < size; i++ {
		pos := rand.Intn(alen)
		b[i] = alphanum[pos]
	}
	return string(b)
}
