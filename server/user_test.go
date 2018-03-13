package server

import (
	"context"
	"fmt"
	"testing"

	"github.com/dictyBase/go-genproto/dictybaseapis/api/jsonapi"

	pb "github.com/dictyBase/go-genproto/dictybaseapis/user"
	"google.golang.org/grpc"
)

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

func NewUpdateUserWithRole(email string, existingUser *pb.User, role *pb.Role) *pb.UpdateUserRequest {
	return &pb.UpdateUserRequest{
		Id: existingUser.Data.Id,
		Data: &pb.UpdateUserRequest_Data{
			Id:   existingUser.Data.Id,
			Type: existingUser.Data.Type,
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
			Relationships: &pb.ExistingUserRelationships{
				Roles: &pb.ExistingUserRelationships_Roles{
					Data: []*jsonapi.Data{
						&jsonapi.Data{Id: role.Data.Id, Type: role.Data.Type},
					},
				},
			},
		},
	}
}

func NewUserWithRole(email string, role *pb.Role) *pb.CreateUserRequest {
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
			Relationships: &pb.NewUserRelationships{
				Roles: &pb.NewUserRelationships_Roles{
					Data: []*jsonapi.Data{
						&jsonapi.Data{Id: role.Data.Id, Type: role.Data.Type},
					},
				},
			},
		},
	}
}

func TestCreateUserWithRole(t *testing.T) {
	defer tearDownTest(t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()

	rlient := pb.NewRoleServiceClient(conn)
	nrole, err := rlient.CreateRole(context.Background(), NewRole("creator"))
	if err != nil {
		t.Fatalf("could not store the role %s\n", err)
	}

	client := pb.NewUserServiceClient(conn)
	nuser, err := client.CreateUser(context.Background(), NewUserWithRole("todd@gad.org", nrole))
	if err != nil {
		t.Fatalf("could not store the user %s\n", err)
	}
	if nuser.Data.Id < 1 {
		t.Fatalf("No id attribute value %d", nuser.Data.Id)
	}
	if nuser.Links.Self != nuser.Data.Links.Self {
		t.Fatalf("top link %s does not match resource link %s", nuser.Links.Self, nuser.Data.Links.Self)
	}
	if nuser.Data.Attributes.Email != "todd@gad.org" {
		t.Fatalf("Expected value of attribute email did not match %s", nuser.Data.Attributes.Email)
	}
	if nuser.Data.Relationships.Roles.Links.Self != fmt.Sprintf("/users/%d/relationships/roles", nuser.Data.Id) {
		t.Fatalf("user's self relationship %s does not match", nuser.Data.Relationships.Roles.Links.Self)
	}
	if nuser.Data.Relationships.Roles.Links.Related != fmt.Sprintf("/users/%d/roles", nuser.Data.Id) {
		t.Fatalf("user's self relationship %s does not match", nuser.Data.Relationships.Roles.Links.Related)
	}
}

func TestUpdateUserWithRole(t *testing.T) {
	defer tearDownTest(t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()

	roleClient := pb.NewRoleServiceClient(conn)
	crole, err := roleClient.CreateRole(context.Background(), NewRole("creator"))
	if err != nil {
		t.Fatalf("could not store the role %s\n", err)
	}
	drole, err := roleClient.CreateRole(context.Background(), NewRole("updater"))
	if err != nil {
		t.Fatalf("could not store the role %s\n", err)
	}

	client := pb.NewUserServiceClient(conn)
	nuser, err := client.CreateUser(context.Background(), NewUserWithRole("kosmo@seinfeld.com", crole))
	if err != nil {
		t.Fatalf("could not store the user %s\n", err)
	}
	// Now update the role
	user, err := client.UpdateUser(
		context.Background(),
		NewUpdateUserWithRole("whatley@seinfeld.org", nuser, drole),
	)
	if err != nil {
		t.Fatalf("could not update the role %s\n", err)
	}
	if user.Data.Attributes.Email != "whatley@seinfeld.org" {
		t.Fatalf("expected email attribute value does not match %s\n", user.Data.Attributes.Email)
	}
}

func TestDeleteUser(t *testing.T) {
	defer tearDownTest(t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()

	client := pb.NewUserServiceClient(conn)
	nuser, err := client.CreateUser(context.Background(), NewUser("leo@seinfeld.org"))
	if err != nil {
		t.Fatalf("could not store the user %s\n", err)
	}
	_, err = client.DeleteUser(context.Background(), &jsonapi.DeleteRequest{Id: nuser.Data.Id})
	if err != nil {
		t.Fatalf("could not delete the user %s\n", err)
	}
}
