package server

import (
	"database/sql"
	"log"
	"net"
	"os"
	"testing"

	"github.com/dictyBase/apihelpers/aphdocker"
	pb "github.com/dictyBase/go-genproto/dictybaseapis/content"
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
	pb.RegisterContentServiceServer(grpcS, NewPermissionService(dbh, "permissions"))
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
