package server

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/dictyBase/go-genproto/dictybaseapis/api/jsonapi"
	pb "github.com/dictyBase/go-genproto/dictybaseapis/user"
	"github.com/dictyBase/modware-user/testutils"
	_ "github.com/jackc/pgx/stdlib"
	"google.golang.org/grpc"
	runner "gopkg.in/mgutz/dat.v2/sqlx-runner"
)

var db *sql.DB

const (
	port = ":9596"
)

func runGRPCServer(db *sql.DB) {
	dbh := runner.NewDB(db, "postgres")
	grpcS := grpc.NewServer()
	pb.RegisterPermissionServiceServer(grpcS, NewPermissionService(dbh))
	pb.RegisterRoleServiceServer(grpcS, NewRoleService(dbh))
	pb.RegisterUserServiceServer(grpcS, NewUserService(dbh))
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("error listening to grpc port %s", err)
	}
	log.Printf("starting grpc server at port %s", port)
	if err := grpcS.Serve(lis); err != nil {
		log.Fatalf("error serving %s", err)
	}
}

func TestMain(m *testing.M) {
	pg, err := testutils.NewTestPostgresFromEnv()
	if err != nil {
		log.Fatalf("unable to construct new NewTestPostgresFromEnv instance %s", err)
	}
	db = pg.DB
	if err := testutils.SetupTestDB(db); err != nil {
		log.Fatalf("error setting up test db %s", err)
	}
	go runGRPCServer(db)
	defer db.Close()
	os.Exit(m.Run())
}

func NewPermission(perm, resource string) *pb.CreatePermissionRequest {
	return &pb.CreatePermissionRequest{
		Data: &pb.CreatePermissionRequest_Data{
			Type: "permissions",
			Attributes: &pb.PermissionAttributes{
				Permission:  perm,
				Resource:    resource,
				Description: fmt.Sprintf("Ability to do %s in %s", perm, resource),
			},
		},
	}
}

func TestPermissionCreate(t *testing.T) {
	defer testutils.TearDownTest(db, t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, "localhost"+port, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()
	client := pb.NewPermissionServiceClient(conn)
	nperm, err := client.CreatePermission(context.Background(), NewPermission("create", "literature"))
	if err != nil {
		t.Fatalf("could not store the permission %s\n", err)
	}
	if nperm.Data.Id < 1 {
		t.Fatalf("No id attribute value %d", nperm.Data.Id)
	}
	if nperm.Links.Self != nperm.Data.Links.Self {
		t.Fatalf("top link %s does not match resource link %s", nperm.Links.Self, nperm.Data.Links.Self)
	}
	if nperm.Data.Attributes.Permission != "create" {
		t.Fatalf("Expected value of attribute permission did not match %s", nperm.Data.Attributes.Permission)
	}
	if nperm.Data.Attributes.Resource != "literature" {
		t.Fatalf("Expected value of resource did not match %s", nperm.Data.Attributes.Resource)
	}
}

func TestPermissionGet(t *testing.T) {
	defer testutils.TearDownTest(db, t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()
	client := pb.NewPermissionServiceClient(conn)
	nperm, err := client.CreatePermission(context.Background(), NewPermission("get", "genome"))
	if err != nil {
		t.Fatalf("could not store the permission %s\n", err)
	}
	eperm, err := client.GetPermission(context.Background(), &jsonapi.GetRequestWithFields{Id: nperm.Data.Id})
	if err != nil {
		t.Fatalf("could not retrieve permission with id %d", nperm.Data.Id)
	}
	if nperm.Data.Id != eperm.Data.Id {
		t.Fatalf("expected id %d does not match %d\n", nperm.Data.Id, eperm.Data.Id)
	}
	efperm, err := client.GetPermission(
		context.Background(),
		&jsonapi.GetRequestWithFields{Id: nperm.Data.Id, Fields: "permission"},
	)
	if err != nil {
		t.Fatalf("could not retrieve permission with id %d", nperm.Data.Id)
	}
	if len(efperm.Data.Attributes.Resource) != 0 {
		t.Fatalf("expecting nil but retrieved %s\n", efperm.Data.Attributes.Resource)
	}
	if m, _ := regexp.MatchString("fields=permission", efperm.Links.Self); !m {
		t.Fatalf("expected link %s does not contain fields query parameter", efperm.Links.Self)
	}
}

func TestPermissionGetAllWithFields(t *testing.T) {
	defer testutils.TearDownTest(db, t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()
	client := pb.NewPermissionServiceClient(conn)
	for _, pt := range []string{"get", "create", "edit", "delete", "admin"} {
		_, err := client.CreatePermission(
			context.Background(),
			NewPermission(pt, "strain"),
		)
		if err != nil {
			t.Fatalf("could not store the permission %s\n", err)
		}
	}
	fperms, err := client.ListPermissions(
		context.Background(),
		&jsonapi.SimpleListRequest{Fields: "permission"},
	)
	if err != nil {
		t.Fatalf("could not fetch all permissions with fields %s\n", err)
	}
	if len(fperms.Data) != 5 {
		t.Fatalf("expected entries does not match %d\n", len(fperms.Data))
	}
	if m, _ := regexp.MatchString("fields=permission", fperms.Links.Self); !m {
		t.Fatalf("expected link %s does not contain fields query parameter", fperms.Links.Self)
	}
	for _, perm := range fperms.Data {
		if len(perm.Attributes.Description) != 0 {
			t.Fatalf("expecting nil but retrieved %s\n", perm.Attributes.Description)
		}
		if perm.Links.Self != fmt.Sprintf("/permissions/%d", perm.Id) {
			t.Fatalf("expected link does not match %s\n", perm.Links.Self)
		}
	}
}

func TestPermissionGetAll(t *testing.T) {
	defer testutils.TearDownTest(db, t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()
	client := pb.NewPermissionServiceClient(conn)
	for _, pt := range []string{"get", "create", "edit", "delete", "admin"} {
		_, err := client.CreatePermission(
			context.Background(),
			NewPermission(pt, "genotype"),
		)
		if err != nil {
			t.Fatalf("could not store the permission %s\n", err)
		}
	}
	lperms, err := client.ListPermissions(context.Background(), &jsonapi.SimpleListRequest{})
	if err != nil {
		t.Fatalf("could not fetch all permissions %s\n", err)
	}
	if len(lperms.Data) != 5 {
		t.Fatalf("expected entries does not match %d\n", len(lperms.Data))
	}
	for _, perm := range lperms.Data {
		if perm.Id < 1 {
			t.Fatalf("expected id does not match %d\n", perm.Id)
		}
		if perm.Links.Self != fmt.Sprintf("/permissions/%d", perm.Id) {
			t.Fatalf("expected link does not match %s\n", perm.Links.Self)
		}
		if perm.Attributes.Resource != "genotype" {
			t.Fatalf("expected resource does not match %s\n", perm.Attributes.Resource)
		}
	}
}

func TestPermissionGetAllWithFieldsAndFilter(t *testing.T) {
	defer testutils.TearDownTest(db, t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()
	client := pb.NewPermissionServiceClient(conn)
	for _, pt := range []string{"get", "create", "edit", "delete", "admin"} {
		_, err := client.CreatePermission(
			context.Background(),
			NewPermission(pt, "goa"),
		)
		if err != nil {
			t.Fatalf("could not store the permission %s\n", err)
		}
	}
	fperms, err := client.ListPermissions(
		context.Background(),
		&jsonapi.SimpleListRequest{
			Fields: "permission",
			Filter: "permission==edit",
		},
	)
	if err != nil {
		t.Fatalf("could not fetch all permissions with fields %s\n", err)
	}
	if len(fperms.Data) < 1 {
		t.Fatalf("expected entries does not match %d\n", len(fperms.Data))
	}
	if m, _ := regexp.MatchString("fields=permission&filter=permission==edit", fperms.Links.Self); !m {
		t.Fatalf("expected link %s does not contain fields query parameter", fperms.Links.Self)
	}
	for _, perm := range fperms.Data {
		if len(perm.Attributes.Description) != 0 {
			t.Fatalf("expecting nil but retrieved %s\n", perm.Attributes.Description)
		}
		if perm.Attributes.Permission != "edit" {
			t.Fatalf("expected permission does not match with %s\n", perm.Attributes.Permission)
		}
		if perm.Links.Self != fmt.Sprintf("/permissions/%d", perm.Id) {
			t.Fatalf("expected link does not match %s\n", perm.Links.Self)
		}
	}
	mperms, err := client.ListPermissions(
		context.Background(),
		&jsonapi.SimpleListRequest{
			Fields: "permission",
			Filter: "permission=@dm",
		},
	)
	if err != nil {
		t.Fatalf("could not fetch all permissions with fields %s\n", err)
	}
	if len(mperms.Data) < 1 {
		t.Fatalf("expected entries does not match %d\n", len(fperms.Data))
	}
	if m, _ := regexp.MatchString("fields=permission&filter=permission=@dm", mperms.Links.Self); !m {
		t.Fatalf("expected link %s does not contain fields query parameter", mperms.Links.Self)
	}
	for _, perm := range mperms.Data {
		if perm.Attributes.Permission != "admin" {
			t.Fatalf("expected permission does not match with %s\n", perm.Attributes.Permission)
		}
	}
}

func TestPermissionUpdate(t *testing.T) {
	defer testutils.TearDownTest(db, t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()
	client := pb.NewPermissionServiceClient(conn)
	nperm, err := client.CreatePermission(context.Background(), NewPermission("edit", "frontpage"))
	if err != nil {
		t.Fatalf("could not store the permission %s\n", err)
	}
	fperm := &pb.UpdatePermissionRequest{
		Data: &pb.UpdatePermissionRequest_Data{
			Type: nperm.Data.Type,
			Id:   nperm.Data.Id,
			Attributes: &pb.PermissionAttributes{
				Permission:  "update",
				Description: fmt.Sprintf("Ability to do %s", "update"),
			},
		},
		Id: nperm.Data.Id,
	}
	uperm, err := client.UpdatePermission(context.Background(), fperm)
	if err != nil {
		t.Fatalf("cannot update permission %s\n", err)
	}
	if fperm.Data.Attributes.Permission != uperm.Data.Attributes.Permission {
		t.Fatalf(
			"expected permission %s does not match with %s",
			fperm.Data.Attributes.Permission,
			uperm.Data.Attributes.Permission,
		)
	}
}

func TestPermissionDelete(t *testing.T) {
	defer testutils.TearDownTest(db, t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()
	client := pb.NewPermissionServiceClient(conn)
	nperm, err := client.CreatePermission(context.Background(), NewPermission("delete", "genotype"))
	if err != nil {
		t.Fatalf("could not store the permission %s\n", err)
	}
	_, err = client.DeletePermission(context.Background(), &jsonapi.DeleteRequest{Id: nperm.Data.Id})
	if err != nil {
		t.Fatalf("could not delete resource with id %d", nperm.Data.Id)
	}
}
