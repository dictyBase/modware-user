package nats

import (
	"context"
	"fmt"
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
	"github.com/dictyBase/modware-user/testutils"
	_ "github.com/jackc/pgx/stdlib"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	runner "gopkg.in/mgutz/dat.v2/sqlx-runner"
)

var natsHost = os.Getenv("NATS_HOST")
var natsPort = os.Getenv("NATS_PORT")
var db *sql.DB

const (
	grpcPort = ":9595"
)

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

func TestMain(m *testing.M) {
	pg, err := testutils.NewTestPostgresFromEnv(true)
	if err != nil {
		log.Fatalf("unable to construct new NewTestPostgresFromEnv instance %s", err)
	}
	db = pg.DB
	_, err = testutils.NewTestNatsFromEnv()
	if err != nil {
		log.Fatalf("unable to construct new NewTestNatsFromEnv instance %s", err)
	}
	if err := testutils.SetupTestDB(db); err != nil {
		log.Fatalf("error setting up test db %s", err)
	}
	go runGRPCServer(db)
	defer db.Close()
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
	defer testutils.TearDownTest(db, t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, "localhost"+grpcPort, grpc.WithInsecure(), grpc.WithBlock())
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
	defer testutils.TearDownTest(db, t)
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
	defer testutils.TearDownTest(db, t)
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
