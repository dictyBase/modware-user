package server

import (
	"context"
	"fmt"
	"testing"

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

func TestRoleCreate(t *testing.T) {
	defer tearDownTest(t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()
	client := pb.NewRoleServiceClient(conn)
	nrole, err := client.CreateRole(context.Background(), NewRole("creator"))
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
}
