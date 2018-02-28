package server

import (
	"context"
	"fmt"
	"testing"

	"github.com/dictyBase/go-genproto/dictybaseapis/api/jsonapi"
	pb "github.com/dictyBase/go-genproto/dictybaseapis/user"
	"google.golang.org/grpc"
)

func NewRole(role string) *pb.CreateRoleRequest {
	return &pb.CreateRoleRequest{
		Data: &pb.CreateRoleRequest_Data{
			Type: "roles",
			Attributes: &pb.RoleAttributes{
				Role:        role,
				Description: fmt.Sprintf("Ability to do %s", role),
			},
		},
	}
}

func NewRoleWithPermission(role string, perm *pb.Permission) *pb.CreateRoleRequest {
	return &pb.CreateRoleRequest{
		Data: &pb.CreateRoleRequest_Data{
			Type: "roles",
			Attributes: &pb.RoleAttributes{
				Role:        role,
				Description: fmt.Sprintf("Ability to do %s", role),
			},
			Relationships: &pb.NewRoleRelationships{
				Permissions: &pb.NewRoleRelationships_Permissions{
					Data: []*jsonapi.Data{
						&jsonapi.Data{Id: perm.Data.Id, Type: perm.Data.Type},
					},
				},
			},
		},
	}
}

func TestRoleCreateWithPermission(t *testing.T) {
	defer tearDownTest(t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()

	permClient := pb.NewPermissionServiceClient(conn)
	perm, err := permClient.CreatePermission(context.Background(), NewPermission("create"))
	if err != nil {
		t.Fatalf("could not store the permission %s\n", err)
	}

	client := pb.NewRoleServiceClient(conn)
	nrole, err := client.CreateRole(context.Background(), NewRoleWithPermission("creator", perm))
	if err != nil {
		t.Fatalf("could not store the role %s\n", err)
	}
	if nrole.Data.Id < 1 {
		t.Fatalf("No id attribute value %d", nrole.Data.Id)
	}
	if nrole.Links.Self != nrole.Data.Links.Self {
		t.Fatalf("top link %s does not match resource link %s", nrole.Links.Self, nrole.Data.Links.Self)
	}
	if nrole.Data.Attributes.Role != "creator" {
		t.Fatalf("Expected value of attribute permission did not match %s", nrole.Data.Attributes.Role)
	}
	if nrole.Data.Relationships.Permissions.Links.Self != fmt.Sprintf("/roles/%d/relationships/permissions", perm.Data.Id) {
		t.Fatalf("permission's self relationship %s does not match", nrole.Data.Relationships.Permissions.Links.Self)
	}
	if nrole.Data.Relationships.Permissions.Links.Related != fmt.Sprintf("/roles/%d/permissions", perm.Data.Id) {
		t.Fatalf("permission's self relationship %s does not match", nrole.Data.Relationships.Permissions.Links.Related)
	}
}
