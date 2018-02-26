package server

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"testing"

	"github.com/dictyBase/apihelpers/aphdocker"
	"github.com/dictyBase/go-genproto/dictybaseapis/api/jsonapi"
	pb "github.com/dictyBase/go-genproto/dictybaseapis/user"
	"github.com/pressly/goose"
	"google.golang.org/grpc"
	runner "gopkg.in/mgutz/dat.v2/sqlx-runner"
)

var db *sql.DB
var schemaRepo string = "https://github.com/dictybase-docker/dictyuser-schema"

const (
	port = ":9596"
)

func runGRPCServer(db *sql.DB) {
	dbh := runner.NewDB(db, "postgres")
	grpcS := grpc.NewServer()
	pb.RegisterPermissionServiceServer(grpcS, NewPermissionService(dbh, "permissions"))
	lis, err := net.Listen("tcp", port)
	if err != nil {
		panic(err)
	}
	log.Printf("starting grpc server at port %s", port)
	if err := grpcS.Serve(lis); err != nil {
		panic(err)
	}
}

func TestMain(m *testing.M) {
	pg, err := aphdocker.NewPgDocker()
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}
	resource, err := pg.Run()
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
	go runGRPCServer(db)
	code := m.Run()
	if err = pg.Purge(resource); err != nil {
		log.Fatalf("unable to remove container %s\n", err)
	}
	os.Exit(code)
}

func NewPermission(perm string) *pb.CreatePermissionRequest {
	return &pb.CreatePermissionRequest{
		Data: &pb.CreatePermissionRequest_Data{
			Type: "permissions",
			Attributes: &pb.PermissionAttributes{
				Permission:  perm,
				Description: fmt.Sprintf("Ability to do %s", perm),
			},
		},
	}
}

func TestPermissionCreate(t *testing.T) {
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()
	client := pb.NewPermissionServiceClient(conn)
	nperm, err := client.CreatePermission(context.Background(), NewPermission("edit"))
	if err != nil {
		t.Fatalf("could not store the content %s\n", err)
	}
	if nperm.Data.Id < 1 {
		t.Fatalf("No id attribute value %d", nperm.Data.Id)
	}
	if nperm.Links.Self != nperm.Data.Links.Self {
		t.Fatalf("top link %s does not match resource link %s", nperm.Links.Self, nperm.Data.Links.Self)
	}
	if nperm.Data.Attributes.Permission != "edit" {
		t.Fatalf("Expected value of attribute permission did not match %s", nperm.Data.Attributes.Permission)
	}
}

func TestPermissionDelete(t *testing.T) {
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()
	client := pb.NewPermissionServiceClient(conn)
	nperm, err := client.CreatePermission(context.Background(), NewPermission("delete"))
	if err != nil {
		t.Fatalf("could not store the content %s\n", err)
	}
	_, err = client.DeletePermission(context.Background(), &jsonapi.DeleteRequest{Id: nperm.Data.Id})
	if err != nil {
		t.Fatalf("could not delete resource with id %s", nperm.Data.Id)
	}
}
