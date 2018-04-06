package nats

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"testing"

	"database/sql"

	"github.com/dictyBase/modware-user/server"
	gnats "github.com/nats-io/go-nats"

	"github.com/nats-io/go-nats/encoders/protobuf"

	"github.com/dictyBase/apihelpers/aphdocker"
	"github.com/dictyBase/go-genproto/dictybaseapis/pubsub"
	pb "github.com/dictyBase/go-genproto/dictybaseapis/user"
	"github.com/dictyBase/modware-user/message"
	gclient "github.com/dictyBase/modware-user/message/grpc-client"
	"github.com/pressly/goose"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	runner "gopkg.in/mgutz/dat.v2/sqlx-runner"
)

var (
	db       *sql.DB
	natsHost string
	natsPort string
)
var schemaRepo string = "https://github.com/dictybase-docker/dictyuser-schema"

const (
	grpcPort = ":9595"
)

func tearDownTest(t *testing.T) {
	for _, tbl := range []string{"auth_permission", "auth_role", "auth_user", "auth_user_info", "auth_user_role", "auth_role_permission"} {
		_, err := db.Exec(fmt.Sprintf("TRUNCATE %s CASCADE", tbl))
		if err != nil {
			t.Fatalf("unable to truncate table %s %s\n", t, err)
		}
	}
}

func runGRPCServer(db *sql.DB) {
	dbh := runner.NewDB(db, "postgres")
	grpcS := grpc.NewServer()
	pb.RegisterUserServiceServer(grpcS, server.NewUserService(dbh))
	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		panic(err)
	}
	log.Printf("starting grpc server at port %s", grpcPort)
	if err := grpcS.Serve(lis); err != nil {
		panic(err)
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

func TestMain(m *testing.M) {
	// storage(postgresql connection)
	pg, err := aphdocker.NewPgDocker()
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}
	pgresource, err := pg.Run()
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}
	db, err = pg.RetryDbConnection()
	if err != nil {
		log.Fatal(err)
	}
	// add the citext extension
	_, err = db.Exec("CREATE EXTENSION citext")
	if err != nil {
		log.Fatal(err)
	}
	dir, err := aphdocker.CloneDbSchemaRepo(schemaRepo)
	defer os.RemoveAll(dir)
	if err != nil {
		log.Fatalf("issue with cloning %s repo %s\n", schemaRepo, err)
	}
	if err := goose.Up(db, dir); err != nil {
		log.Fatalf("issue with running database migration %s\n", err)
	}
	nats, err := aphdocker.NewNatsDocker()
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	// nats messaging server startup
	nresource, err := nats.Run()
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}
	_, err = nats.RetryNatsConnection()
	if err != nil {
		log.Fatal(err)
	}
	natsHost = nats.GetIP()
	natsPort = nats.GetPort()
	go runGRPCServer(db)
	code := m.Run()
	if err = pg.Purge(pgresource); err != nil {
		log.Fatalf("unable to remove postgresql container %s\n", err)
	}
	if err = nats.Purge(nresource); err != nil {
		log.Fatalf("unable to remove nats container %s\n", err)
	}
	os.Exit(code)
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
		t.Fatalf("expected user id %s does not match %s", nuser.Data.Id, ruser.User.Data.Id)
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
