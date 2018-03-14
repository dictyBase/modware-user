package server

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/dictyBase/go-genproto/dictybaseapis/api/jsonapi"
	"github.com/golang/protobuf/ptypes"

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

func TestGetUser(t *testing.T) {
	defer tearDownTest(t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()

	client := pb.NewUserServiceClient(conn)
	nuser, err := client.CreateUser(context.Background(), NewUser("bobsacamano@seinfeld.org"))
	if err != nil {
		t.Fatalf("could not store the user %s\n", err)
	}
	guser, err := client.GetUser(context.Background(), &jsonapi.GetRequest{Id: nuser.Data.Id})
	if err != nil {
		t.Fatalf("could not fetch the user %s\n", err)
	}
	if guser.Data.Id != nuser.Data.Id {
		t.Fatalf("expected id %d does not match %d\n", nuser.Data.Id, guser.Data.Id)
	}
	if guser.Data.Attributes.Email != "bobsacamano@seinfeld.org" {
		t.Fatalf("expected email %s does not match %s\n", guser.Data.Attributes.Email, "bobsacamano@seinfeld.org")
	}
	if guser.Data.Attributes.Country != "US" {
		t.Fatalf("expected country name does not match %s\n", guser.Data.Attributes.Country)
	}
}

func TestGetUserWithRole(t *testing.T) {
	defer tearDownTest(t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()

	roleClient := pb.NewRoleServiceClient(conn)
	nrole, err := roleClient.CreateRole(context.Background(), NewRole("fetcher"))
	if err != nil {
		t.Fatalf("could not store the role %s\n", err)
	}
	client := pb.NewUserServiceClient(conn)
	nuser, err := client.CreateUser(context.Background(), NewUserWithRole("bobsacamano@seinfeld.org", nrole))
	if err != nil {
		t.Fatalf("could not store the user %s\n", err)
	}

	guser, err := client.GetUser(
		context.Background(),
		&jsonapi.GetRequest{Id: nuser.Data.Id, Include: "roles"},
	)
	if err != nil {
		t.Fatalf("could not fetch the user %s\n", err)
	}
	if guser.Data.Id != nuser.Data.Id {
		t.Fatalf("expected id %d does not match %d\n", nuser.Data.Id, guser.Data.Id)
	}
	if guser.Data.Attributes.Email != "bobsacamano@seinfeld.org" {
		t.Fatalf("expected email %s does not match %s\n", guser.Data.Attributes.Email, "bobsacamano@seinfeld.org")
	}
	if !guser.Data.Attributes.IsActive {
		t.Fatal("expected user is expected to be active")
	}
	if m, _ := regexp.MatchString("include=roles", guser.Links.Self); !m {
		t.Fatalf("expected link %s does not contain include query parameter", guser.Links.Self)
	}
	if len(guser.Included) != 1 {
		t.Fatalf("expected no of included roled does match with %d\n", len(guser.Included))
	}
	for _, a := range guser.Included {
		roleData := &pb.RoleData{}
		if err := ptypes.UnmarshalAny(a, roleData); err != nil {
			t.Fatalf("error in unmarshaling any types %s\n", err)
		} else {
			if roleData.Id != nrole.Data.Id {
				t.Fatalf("expected id does not match with %s\n", roleData.Id)
			}
			if roleData.Links.Self != nrole.Links.Self {
				t.Fatalf("expected link does not match with %s\n", roleData.Links.Self)
			}
			if roleData.Attributes.Role != nrole.Data.Attributes.Role {
				t.Fatalf("expected permission does not match with %s\n", roleData.Attributes.Role)
			}
		}
	}
}
