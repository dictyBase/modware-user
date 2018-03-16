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

func TestGetAllUsers(t *testing.T) {
	defer tearDownTest(t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()

	client := pb.NewUserServiceClient(conn)
	for i := 0; i < 28; i++ {
		_, err := client.CreateUser(
			context.Background(),
			NewUser(fmt.Sprintf("%s@seinfeld.com", RandString(10))),
		)
		if err != nil {
			t.Fatalf("could not store the user %s\n", err)
		}
	}
	lusers, err := client.ListUsers(context.Background(), &jsonapi.ListRequest{})
	if err != nil {
		t.Fatalf("could not fetch all users %s\n", err)
	}
	if len(lusers.Data) != 10 {
		t.Fatalf("expected entries does not match %d\n", len(lusers.Data))
	}
	if lusers.Links.Self != lusers.Links.First {
		t.Fatalf("self should match with first link %s", lusers.Links.First)
	}
	if m, _ := regexp.MatchString("pagenum=1&pagesize=10", lusers.Links.Self); !m {
		t.Fatalf("expected self link does not match %s", lusers.Links.Self)
	}
	if m, _ := regexp.MatchString("pagenum=2&pagesize=10", lusers.Links.Next); !m {
		t.Fatalf("expected next link does not match %s", lusers.Links.Next)
	}
	if m, _ := regexp.MatchString("pagenum=3&pagesize=10", lusers.Links.Last); !m {
		t.Fatalf("expected last link does not match %s", lusers.Links.Last)
	}
	for _, user := range lusers.Data {
		if user.Id < 1 {
			t.Fatalf("expected id does not match %d\n", user.Id)
		}
		if user.Links.Self != fmt.Sprintf("/users/%d", user.Id) {
			t.Fatalf("expected link does not match %s\n", user.Links.Self)
		}
	}
	page := lusers.Meta.Pagination
	if page.Records != 28 {
		t.Logf("expected total no of records does not match %d\n", page.Records)
	}
	if page.Size != 10 {
		t.Logf("expected page size does not match %d\n", page.Size)
	}
	if page.Number != 1 {
		t.Logf("expected page number does not match %d\n", page.Number)
	}
	if page.Total != 3 {
		t.Logf("expected no of pages does not match %d\n", page.Total)
	}

	tusers, err := client.ListUsers(context.Background(), &jsonapi.ListRequest{Pagenum: 3})
	if err != nil {
		t.Fatalf("could not fetch all users %s\n", err)
	}
	if len(tusers.Data) != 8 {
		t.Fatalf("expected entries does not match %d\n", len(tusers.Data))
	}
	if m, _ := regexp.MatchString("pagenum=3&pagesize=10", tusers.Links.Self); !m {
		t.Fatalf("expected link %s does not contain include query parameter", tusers.Links.Self)
	}
	if m, _ := regexp.MatchString("pagenum=1&pagesize=10", tusers.Links.First); !m {
		t.Fatalf("expected link %s does not contain include query parameter", tusers.Links.First)
	}
	if m, _ := regexp.MatchString("pagenum=2&pagesize=10", tusers.Links.Prev); !m {
		t.Fatalf("expected link %s does not contain include query parameter", tusers.Links.Prev)
	}
	tpage := tusers.Meta.Pagination
	if tpage.Number != 3 {
		t.Logf("expected page number does not match %d\n", tpage.Number)
	}

	susers, err := client.ListUsers(context.Background(), &jsonapi.ListRequest{Pagesize: 5})
	if err != nil {
		t.Fatalf("could not fetch all users %s\n", err)
	}
	if len(susers.Data) != 5 {
		t.Fatalf("expected entries does not match %d\n", len(susers.Data))
	}
	if m, _ := regexp.MatchString("pagenum=1&pagesize=5", susers.Links.Self); !m {
		t.Fatalf("expected link %s does not contain include query parameter", susers.Links.Self)
	}
	spage := susers.Meta.Pagination
	if spage.Number != 1 {
		t.Logf("expected page number does not match %d\n", spage.Number)
	}
	if spage.Size != 5 {
		t.Logf("expected page size does not match %d\n", spage.Size)
	}

	ausers, err := client.ListUsers(context.Background(), &jsonapi.ListRequest{Pagesize: 5, Pagenum: 2})
	if err != nil {
		t.Fatalf("could not fetch all users %s\n", err)
	}
	if len(ausers.Data) != 5 {
		t.Fatalf("expected entries does not match %d\n", len(ausers.Data))
	}
	if m, _ := regexp.MatchString("pagenum=2&pagesize=5", ausers.Links.Self); !m {
		t.Fatalf("expected link %s does not contain include query parameter", ausers.Links.Self)
	}
	if m, _ := regexp.MatchString("pagenum=3&pagesize=5", ausers.Links.Next); !m {
		t.Fatalf("expected link %s does not contain include query parameter", ausers.Links.Next)
	}
	if m, _ := regexp.MatchString("pagenum=6&pagesize=5", ausers.Links.Last); !m {
		t.Fatalf("expected link %s does not contain include query parameter", ausers.Links.Last)
	}
	apage := ausers.Meta.Pagination
	if apage.Number != 2 {
		t.Logf("expected page number does not match %d\n", apage.Number)
	}
	if apage.Size != 5 {
		t.Logf("expected page size does not match %d\n", apage.Size)
	}
	if apage.Total != 6 {
		t.Logf("expected no of pages does not match %d\n", apage.Total)
	}
	musers, err := client.ListUsers(context.Background(), &jsonapi.ListRequest{Pagesize: 5, Pagenum: 6})
	if err != nil {
		t.Fatalf("could not fetch all users %s\n", err)
	}
	if len(musers.Data) != 3 {
		t.Fatalf("expected entries does not match %d\n", len(musers.Data))
	}
}

func TestGetAllUsersWithFilter(t *testing.T) {
	defer tearDownTest(t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()

	client := pb.NewUserServiceClient(conn)
	for i := 0; i < 20; i++ {
		_, err := client.CreateUser(
			context.Background(),
			NewUser(fmt.Sprintf("%s@seinfeld.com", RandString(10))),
		)
		if err != nil {
			t.Fatalf("could not store the user %s\n", err)
		}
	}
	for i := 0; i < 20; i++ {
		_, err := client.CreateUser(
			context.Background(),
			NewUser(fmt.Sprintf("%s@kramer.com", RandString(10))),
		)
		if err != nil {
			t.Fatalf("could not store the user %s\n", err)
		}
	}
	fusers, err := client.ListUsers(
		context.Background(),
		&jsonapi.ListRequest{
			Pagesize: 5,
			Filter:   "email=@kramer",
		})
	if err != nil {
		t.Fatalf("could not fetch all users %s\n", err)
	}
	if len(fusers.Data) != 5 {
		t.Fatalf("expected 5, retrieved %d\n", len(fusers.Data))
	}
	if m, _ := regexp.MatchString("pagenum=4&pagesize=5", fusers.Links.Last); !m {
		t.Fatalf("expected last link does not match %s", fusers.Links.Last)
	}
	page := fusers.Meta.Pagination
	if page.Records != 20 {
		t.Logf("expected total no of records does not match %d\n", page.Records)
	}
	if page.Size != 5 {
		t.Logf("expected page size does not match %d\n", page.Size)
	}
	if page.Number != 1 {
		t.Logf("expected page number does not match %d\n", page.Number)
	}
	if page.Total != 4 {
		t.Logf("expected no of pages does not match %d\n", page.Total)
	}
}

func TestGetAllUsersWithRoles(t *testing.T) {
	defer tearDownTest(t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()

	roleClient := pb.NewRoleServiceClient(conn)
	nrole, err := roleClient.CreateRole(context.Background(), NewRole("editor"))
	if err != nil {
		t.Fatalf("could not store the role %s\n", err)
	}
	client := pb.NewUserServiceClient(conn)
	for i := 0; i < 20; i++ {
		_, err := client.CreateUser(
			context.Background(),
			NewUserWithRole(
				fmt.Sprintf("%s@seinfeld.com", RandString(10)),
				nrole,
			),
		)
		if err != nil {
			t.Fatalf("could not store the user %s\n", err)
		}
	}
	for i := 0; i < 20; i++ {
		_, err := client.CreateUser(
			context.Background(),
			NewUser(fmt.Sprintf("%s@kramer.com", RandString(10))),
		)
		if err != nil {
			t.Fatalf("could not store the user %s\n", err)
		}
	}
	fusers, err := client.ListUsers(
		context.Background(),
		&jsonapi.ListRequest{
			Pagesize: 5,
			Filter:   "email=@seinfeld",
			Include:  "roles",
		})
	if m, _ := regexp.MatchString("pagenum=4&pagesize=5", fusers.Links.Last); !m {
		t.Fatalf("expected last link does not match %s", fusers.Links.Last)
	}
	page := fusers.Meta.Pagination
	if page.Records != 20 {
		t.Logf("expected total no of records does not match %d\n", page.Records)
	}
	counter := 0
	for _, user := range fusers.Data {
		if user.Links.Self != fmt.Sprintf("/users/%d", user.Id) {
			t.Fatalf("expected link does not match %s\n", user.Links.Self)
		}
		if len(user.Relationships.Roles.Data) != 1 {
			t.Fatalf("expected included elements does not match %d\n", len(user.Relationships.Roles.Data))
		}
		if user.Relationships.Roles.Data[0].Id != nrole.Data.Id {
			t.Fatalf(
				"expected id %d does not match %d\n",
				user.Relationships.Roles.Data[0].Id,
				nrole.Data.Id,
			)
		}
		counter++
	}
	if counter != len(fusers.Included) {
		t.Fatalf("relationship resources %d does not match with %d included resources", counter, len(fusers.Included))
	}
	for _, a := range fusers.Included {
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

func TestGetRelatedRoles(t *testing.T) {
	defer tearDownTest(t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()

	roleClient := pb.NewRoleServiceClient(conn)
	role, err := roleClient.CreateRole(context.Background(), NewRole("fetcher"))
	if err != nil {
		t.Fatalf("could not store the role %s\n", err)
	}
	client := pb.NewUserServiceClient(conn)
	nuser, err := client.CreateUser(context.Background(), NewUserWithRole("bobsacamano@seinfeld.org", role))
	if err != nil {
		t.Fatalf("could not store the user %s\n", err)
	}
	nrole, err := client.GetRelatedRoles(
		context.Background(),
		&jsonapi.RelationshipRequest{
			Id: nuser.Data.Id,
		},
	)
	if err != nil {
		t.Fatalf("could not fetch role relationships %s\n", err)
	}
	if role.Data.Id != nrole.Data[0].Id {
		t.Fatalf("expected id %d does not match retrieved %d id", role.Data.Id, nrole.Data[0].Id)
	}
	if nuser.Data.Relationships.Roles.Links.Related != nrole.Links.Self {
		t.Fatalf("expected relationships link does not match %s", nrole.Links.Self)
	}
}

func TestUpdateRelatedRoles(t *testing.T) {
	defer tearDownTest(t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()

	roleClient := pb.NewRoleServiceClient(conn)
	role, err := roleClient.CreateRole(context.Background(), NewRole("fetcher"))
	if err != nil {
		t.Fatalf("could not store the role %s\n", err)
	}
	client := pb.NewUserServiceClient(conn)
	nuser, err := client.CreateUser(context.Background(), NewUserWithRole("bobsacamano@seinfeld.org", role))
	if err != nil {
		t.Fatalf("could not store the user %s\n", err)
	}
	urole, err := roleClient.CreateRole(context.Background(), NewRole("updater"))
	if err != nil {
		t.Fatalf("could not store the role %s\n", err)
	}
	_, err = client.UpdateRoleRelationship(
		context.Background(),
		&jsonapi.DataCollection{
			Id: nuser.Data.Id,
			Data: []*jsonapi.Data{
				&jsonapi.Data{
					Type: "roles",
					Id:   urole.Data.Id,
				},
			},
		})
	if err != nil {
		t.Fatalf("could not update the relationship with role %s\n", err)
	}
	guser, err := client.GetUser(
		context.Background(),
		&jsonapi.GetRequest{
			Id:      nuser.Data.Id,
			Include: "roles",
		})
	if err != nil {
		t.Fatalf("could not fetch the user %s\n", err)
	}
	for _, a := range guser.Included {
		roleData := &pb.RoleData{}
		if err := ptypes.UnmarshalAny(a, roleData); err != nil {
			t.Fatalf("error in unmarshaling any types %s\n", err)
		} else {
			if roleData.Id != urole.Data.Id {
				t.Fatalf("expected id does not match with %s\n", roleData.Id)
			}
			if roleData.Links.Self != urole.Links.Self {
				t.Fatalf("expected link does not match with %s\n", roleData.Links.Self)
			}
			if roleData.Attributes.Role != urole.Data.Attributes.Role {
				t.Fatalf("expected permission does not match with %s\n", roleData.Attributes.Role)
			}
		}
	}
}
