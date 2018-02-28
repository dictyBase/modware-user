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

func NewUpdateRoleWithPermission(newRole string, existingRole *pb.Role, perm *pb.Permission) *pb.UpdateRoleRequest {
	return &pb.UpdateRoleRequest{
		Id: existingRole.Data.Id,
		Data: &pb.UpdateRoleRequest_Data{
			Id:   existingRole.Data.Id,
			Type: existingRole.Data.Type,
			Attributes: &pb.RoleAttributes{
				Role:        newRole,
				Description: fmt.Sprintf("Ability to do %s", newRole),
			},
			Relationships: &pb.ExistingRoleRelationships{
				Permissions: &pb.ExistingRoleRelationships_Permissions{
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

func TestRoleUpdateWithPermission(t *testing.T) {
	defer tearDownTest(t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()

	permClient := pb.NewPermissionServiceClient(conn)
	cperm, err := permClient.CreatePermission(context.Background(), NewPermission("create"))
	if err != nil {
		t.Fatalf("could not store the permission %s\n", err)
	}
	dperm, err := permClient.CreatePermission(context.Background(), NewPermission("destroy"))
	if err != nil {
		t.Fatalf("could not store the permission %s\n", err)
	}

	client := pb.NewRoleServiceClient(conn)
	nrole, err := client.CreateRole(context.Background(), NewRoleWithPermission("creator", cperm))
	if err != nil {
		t.Fatalf("could not store the role %s\n", err)
	}
	// Now update the role
	urole, err := client.UpdateRole(
		context.Background(),
		NewUpdateRoleWithPermission("destroyer", nrole, dperm),
	)
	if err != nil {
		t.Fatalf("could not update the role %s\n", err)
	}
	if urole.Data.Attributes.Role != "destroyer" {
		t.Fatalf("expected role does not match %s\n", urole.Data.Attributes.Role)
	}
}

func TestRoleDelete(t *testing.T) {
	defer tearDownTest(t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()

	client := pb.NewRoleServiceClient(conn)
	nrole, err := client.CreateRole(context.Background(), NewRole("deleter"))
	if err != nil {
		t.Fatalf("could not store the role %s\n", err)
	}
	_, err = client.DeleteRole(context.Background(), &jsonapi.DeleteRequest{Id: nrole.Data.Id})
	if err != nil {
		t.Fatalf("could not delete the role %s\n", err)
	}
}
