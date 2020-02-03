package nats

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"testing"
	"time"

	"database/sql"

	"github.com/dictyBase/modware-user/server"
	gnats "github.com/nats-io/go-nats"

	"github.com/nats-io/go-nats/encoders/protobuf"

	"github.com/dictyBase/go-genproto/dictybaseapis/pubsub"
	pb "github.com/dictyBase/go-genproto/dictybaseapis/user"
	"github.com/dictyBase/modware-user/message"
	gclient "github.com/dictyBase/modware-user/message/grpc-client"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/pressly/goose"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	runner "gopkg.in/mgutz/dat.v2/sqlx-runner"
	git "gopkg.in/src-d/go-git.v4"
)

var pgAddr = fmt.Sprintf("%s:%s", os.Getenv("POSTGRES_HOST"), os.Getenv("POSTGRES_PORT"))
var pgConn = fmt.Sprintf(
	"postgres://%s:%s@%s/%s?sslmode=disable",
	os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD"), pgAddr, os.Getenv("POSTGRES_DB"))
var natsHost = os.Getenv("NATS_HOST")
var natsPort = os.Getenv("NATS_PORT")
var natsAddr = fmt.Sprintf("nats://%s:%s", natsHost, natsPort)
var schemaRepo string = "https://github.com/dictybase-docker/dictyuser-schema"
var db *sql.DB

const (
	grpcPort = ":9595"
)

func tearDownTest(t *testing.T) {
	for _, tbl := range []string{"auth_permission", "auth_role", "auth_user", "auth_user_info", "auth_user_role", "auth_role_permission"} {
		_, err := db.Exec(fmt.Sprintf("TRUNCATE %s CASCADE", tbl))
		if err != nil {
			t.Fatalf("unable to truncate table %s %s\n", tbl, err)
		}
	}
}

func runGRPCServer(db *sql.DB) {
	dbh := runner.NewDB(db, "postgres")
	grpcS := grpc.NewServer()
	pb.RegisterUserServiceServer(grpcS, server.NewUserService(dbh))
	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("error listening to grpc port %s", err)
	}
	log.Printf("starting grpc server at port %s", grpcPort)
	if err := grpcS.Serve(lis); err != nil {
		log.Fatalf("error serving user server %s", err)
	}
}

func NewUser(email string) *pb.CreateUserRequest {
	return &pb.CreateUserRequest{
		Data: &pb.CreateUserRequest_Data{
			Type: "users",
			Attributes: &pb.UserAttributes{
				FirstName:    "Todd",
				LastName:     "Gad",
				Email:        email,
				Organization: "Gadd organization",
				GroupName:    "Gadd group",
				FirstAddress: "34, ronan place",
				City:         "Tokurihm",
				State:        "TL",
				Zipcode:      "54321",
				Country:      "US",
				Phone:        "435-234-8791",
				IsActive:     true,
			},
		},
	}
}

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

type TestPostgres struct {
	DB *sql.DB
}

func NewTestPostgresFromEnv() (*TestPostgres, error) {
	pg := new(TestPostgres)
	if err := CheckPostgresEnv(); err != nil {
		return pg, err
	}
	dbh, err := sql.Open("pgx", pgConn)
	if err != nil {
		return pg, err
	}
	timeout, err := time.ParseDuration("28s")
	if err != nil {
		return pg, err
	}
	t1 := time.Now()
	for {
		if err := dbh.Ping(); err != nil {
			if time.Since(t1).Seconds() > timeout.Seconds() {
				return pg, errors.New("timed out, no connection retrieved")
			}
			continue
		}
		break
	}
	pg.DB = dbh
	return pg, nil
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

func cloneDbSchemaRepo(repo string) (string, error) {
	path, err := ioutil.TempDir("", "content")
	if err != nil {
		return path, err
	}
	_, err = git.PlainClone(path, false, &git.CloneOptions{URL: repo})
	return path, err
}

func TestMain(m *testing.M) {
	pg, err := NewTestPostgresFromEnv()
	if err != nil {
		log.Fatalf("unable to construct new NewTestPostgresFromEnv instance %s", err)
	}
	db = pg.DB
	// add the citext extension
	_, err = db.Exec("CREATE EXTENSION citext")
	if err != nil {
		log.Fatal(err)
	}
	dir, err := cloneDbSchemaRepo(schemaRepo)
	defer os.RemoveAll(dir)
	if err != nil {
		log.Fatalf("issue with cloning %s repo %s\n", schemaRepo, err)
	}
	if err := goose.Up(db, dir); err != nil {
		log.Fatalf("issue with running database migration %s\n", err)
	}
	_, err = NewTestNatsFromEnv()
	if err != nil {
		log.Fatalf("unable to construct new NewTestNatsFromEnv instance %s", err)
	}
	go runGRPCServer(db)
	os.Exit(m.Run())
}

func newNatsRequest(host, port string) (*gnats.EncodedConn, error) {
	nc, err := gnats.Connect(fmt.Sprintf("nats://%s:%s", host, port))
	if err != nil {
		return &gnats.EncodedConn{}, err
	}
	enc, err := gnats.NewEncodedConn(nc, protobuf.PROTOBUF_ENCODER)
	if err != nil {
		return &gnats.EncodedConn{}, err
	}
	return enc, nil
}

func replyUser(subj string, c message.UserClient, req *pubsub.IdRequest) *pubsub.UserReply {
	switch subj {
	case "UserService.Get":
		u, err := c.Get(req.Id)
		if err != nil {
			st, _ := status.FromError(err)
			return &pubsub.UserReply{
				Status: st.Proto(),
				Exist:  false,
			}
		}
		return &pubsub.UserReply{
			Exist: true,
			User:  u,
		}
	case "UserService.Exist":
		exist, err := c.Exist(req.Id)
		if err != nil {
			st, _ := status.FromError(err)
			return &pubsub.UserReply{
				Status: st.Proto(),
				Exist:  exist,
			}
		}
		return &pubsub.UserReply{
			Exist: exist,
		}
	case "UserService.Delete":
		deleted, err := c.Delete(req.Id)
		if err != nil {
			st, _ := status.FromError(err)
			return &pubsub.UserReply{
				Status: st.Proto(),
				Exist:  deleted,
			}
		}
		return &pubsub.UserReply{
			Exist: deleted,
		}
	default:
		return &pubsub.UserReply{
			Status: status.Newf(codes.Internal, "subject %s is not supported", subj).Proto(),
		}
	}
}

func TestUserGetReply(t *testing.T) {
	defer tearDownTest(t)
	conn, err := grpc.Dial("localhost"+grpcPort, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()
	req, err := newNatsRequest(natsHost, natsPort)
	if err != nil {
		t.Fatalf("cannot connect to nats %s\n", err)
	}
	defer req.Close()
	client := pb.NewUserServiceClient(conn)
	nuser, err := client.CreateUser(context.Background(), NewUser("bobsacamano@seinfeld.org"))
	if err != nil {
		t.Fatalf("could not store the user %s\n", err)
	}
	reply, err := NewReply(natsHost, natsPort)
	if err != nil {
		t.Fatalf("could not connect to nats server %s\n", err)
	}
	defer reply.Stop()
	nclient := gclient.NewUserClient(conn)
	err = reply.Start("UserService.*", nclient, replyUser)
	if err != nil {
		t.Fatalf("could not start nats reply subscription %s", err)
	}
	ruser := &pubsub.UserReply{}
	err = req.RequestWithContext(
		context.Background(),
		"UserService.Get",
		&pubsub.IdRequest{Id: nuser.Data.Id},
		ruser,
	)
	if err != nil {
		t.Fatalf("error with sending nats request %s", err)
	}
	if !ruser.Exist {
		t.Fatalf("error in fetching user %s", status.ErrorProto(ruser.Status))
	}
	if ruser.User.Data.Id != nuser.Data.Id {
		t.Fatalf("expected user id %d does not match %d", nuser.Data.Id, ruser.User.Data.Id)
	}
	if nuser.Data.Attributes.Email != ruser.User.Data.Attributes.Email {
		t.Fatalf("expected user email %s does not match %s", nuser.Data.Attributes.Email, ruser.User.Data.Attributes.Email)
	}
}

func TestUserExistReply(t *testing.T) {
	defer tearDownTest(t)
	conn, err := grpc.Dial("localhost"+grpcPort, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()
	req, err := newNatsRequest(natsHost, natsPort)
	if err != nil {
		t.Fatalf("cannot connect to nats %s\n", err)
	}
	defer req.Close()
	client := pb.NewUserServiceClient(conn)
	nuser, err := client.CreateUser(context.Background(), NewUser("art@vandelay.org"))
	if err != nil {
		t.Fatalf("could not store the user %s\n", err)
	}
	reply, err := NewReply(natsHost, natsPort)
	if err != nil {
		t.Fatalf("could not connect to nats server %s\n", err)
	}
	defer reply.Stop()
	nclient := gclient.NewUserClient(conn)
	err = reply.Start("UserService.*", nclient, replyUser)
	if err != nil {
		t.Fatalf("could not start nats reply subscription %s", err)
	}
	ruser := &pubsub.UserReply{}
	err = req.RequestWithContext(
		context.Background(),
		"UserService.Exist",
		&pubsub.IdRequest{Id: nuser.Data.Id},
		ruser,
	)
	if !ruser.Exist {
		t.Fatalf("error in checking existence of user %s", status.ErrorProto(ruser.Status))
	}
}

func TestUserDeleteReply(t *testing.T) {
	defer tearDownTest(t)
	conn, err := grpc.Dial("localhost"+grpcPort, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()
	req, err := newNatsRequest(natsHost, natsPort)
	if err != nil {
		t.Fatalf("cannot connect to nats %s\n", err)
	}
	defer req.Close()
	client := pb.NewUserServiceClient(conn)
	nuser, err := client.CreateUser(context.Background(), NewUser("bobsacamano@seinfeld.org"))
	if err != nil {
		t.Fatalf("could not store the user %s\n", err)
	}
	reply, err := NewReply(natsHost, natsPort)
	if err != nil {
		t.Fatalf("could not connect to nats server %s\n", err)
	}
	defer reply.Stop()
	nclient := gclient.NewUserClient(conn)
	err = reply.Start("UserService.*", nclient, replyUser)
	if err != nil {
		t.Fatalf("could not start nats reply subscription %s", err)
	}
	ruser := &pubsub.UserReply{}
	err = req.RequestWithContext(
		context.Background(),
		"UserService.Delete",
		&pubsub.IdRequest{Id: nuser.Data.Id},
		ruser,
	)
	if !ruser.Exist {
		t.Fatalf("error in delete user %s", status.ErrorProto(ruser.Status))
	}
}
