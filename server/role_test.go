package server

import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"testing"
	"time"

	"github.com/dictyBase/go-genproto/dictybaseapis/api/jsonapi"
	pb "github.com/dictyBase/go-genproto/dictybaseapis/user"
	"github.com/dictyBase/modware-user/testutils"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/grpc"
)

const (
	charSet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

var seedRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

func stringWithCharset(length int, charset string) string {
	var b []byte
	for i := 0; i < length; i++ {
		b = append(
			b,
			charset[seedRand.Intn(len(charset))],
		)
	}
	return string(b)
}

func RandString(length int) string {
	return stringWithCharset(length, charSet)
}

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

func NewRoleWithMultiUsers(role string, allUsers []*pb.User) *pb.CreateRoleRequest {
	var allData []*jsonapi.Data
	for _, u := range allUsers {
		allData = append(
			allData,
			&jsonapi.Data{Id: u.Data.Id, Type: u.Data.Type},
		)
	}
	return &pb.CreateRoleRequest{
		Data: &pb.CreateRoleRequest_Data{
			Type: "roles",
			Attributes: &pb.RoleAttributes{
				Role:        role,
				Description: fmt.Sprintf("Ability to do %s", role),
			},
			Relationships: &pb.NewRoleRelationships{
				Users: &pb.NewRoleRelationships_Users{
					Data: allData,
				},
			},
		},
	}
}

func NewRoleWithUser(role string, user *pb.User) *pb.CreateRoleRequest {
	return &pb.CreateRoleRequest{
		Data: &pb.CreateRoleRequest_Data{
			Type: "roles",
			Attributes: &pb.RoleAttributes{
				Role:        role,
				Description: fmt.Sprintf("Ability to do %s", role),
			},
			Relationships: &pb.NewRoleRelationships{
				Users: &pb.NewRoleRelationships_Users{
					Data: []*jsonapi.Data{
						&jsonapi.Data{Id: user.Data.Id, Type: user.Data.Type},
					},
				},
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
	defer testutils.TearDownTest(db, t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()

	permClient := pb.NewPermissionServiceClient(conn)
	perm, err := permClient.CreatePermission(context.Background(), NewPermission("create", "genome"))
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
	if nrole.Data.Relationships.Permissions.Links.Self != fmt.Sprintf("/roles/%d/relationships/permissions", nrole.Data.Id) {
		t.Fatalf("permission's self relationship %s does not match", nrole.Data.Relationships.Permissions.Links.Self)
	}
	if nrole.Data.Relationships.Permissions.Links.Related != fmt.Sprintf("/roles/%d/permissions", nrole.Data.Id) {
		t.Fatalf("permission's self relationship %s does not match", nrole.Data.Relationships.Permissions.Links.Related)
	}
}

func TestRoleUpdateWithPermission(t *testing.T) {
	defer testutils.TearDownTest(db, t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()

	permClient := pb.NewPermissionServiceClient(conn)
	cperm, err := permClient.CreatePermission(context.Background(), NewPermission("create", "phenotype"))
	if err != nil {
		t.Fatalf("could not store the permission %s\n", err)
	}
	dperm, err := permClient.CreatePermission(context.Background(), NewPermission("destroy", "genotype"))
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
	defer testutils.TearDownTest(db, t)
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

func TestRoleGet(t *testing.T) {
	defer testutils.TearDownTest(db, t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()

	client := pb.NewRoleServiceClient(conn)
	nrole, err := client.CreateRole(context.Background(), NewRole("fetcher"))
	if err != nil {
		t.Fatalf("could not store the role %s\n", err)
	}
	grole, err := client.GetRole(context.Background(), &jsonapi.GetRequest{Id: nrole.Data.Id})
	if err != nil {
		t.Fatalf("could not fetch the role %s\n", err)
	}
	if grole.Data.Id != nrole.Data.Id {
		t.Fatalf("expected id %d does not match %d\n", nrole.Data.Id, grole.Data.Id)
	}
	if grole.Data.Attributes.Role != "fetcher" {
		t.Fatalf("expected role %s does not match %s\n", grole.Data.Attributes.Role, "fetcher")
	}
}

func TestRoleGetWithFields(t *testing.T) {
	defer testutils.TearDownTest(db, t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()

	client := pb.NewRoleServiceClient(conn)
	nrole, err := client.CreateRole(context.Background(), NewRole("fetcher"))
	if err != nil {
		t.Fatalf("could not store the role %s\n", err)
	}
	grole, err := client.GetRole(
		context.Background(),
		&jsonapi.GetRequest{Id: nrole.Data.Id, Fields: "role"},
	)
	if err != nil {
		t.Fatalf("could not delete the role %s\n", err)
	}
	if len(grole.Data.Attributes.Description) != 0 {
		t.Fatalf("expecting nil but retrieved %s\n", grole.Data.Attributes.Description)
	}
	if m, _ := regexp.MatchString("fields=role", grole.Links.Self); !m {
		t.Fatalf("expected link %s does not contain fields query parameter", grole.Links.Self)
	}
}

func TestRoleGetWithFieldsAndInclude(t *testing.T) {
	defer testutils.TearDownTest(db, t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()

	permClient := pb.NewPermissionServiceClient(conn)
	perm, err := permClient.CreatePermission(context.Background(), NewPermission("fetch", "notes"))
	if err != nil {
		t.Fatalf("could not store the permission %s\n", err)
	}

	client := pb.NewRoleServiceClient(conn)
	nrole, err := client.CreateRole(context.Background(), NewRoleWithPermission("fetcher", perm))
	if err != nil {
		t.Fatalf("could not store the role %s\n", err)
	}
	grole, err := client.GetRole(
		context.Background(),
		&jsonapi.GetRequest{Id: nrole.Data.Id, Fields: "role", Include: "permissions"},
	)
	if err != nil {
		t.Fatalf("could not fetch the role %s\n", err)
	}
	if len(grole.Data.Attributes.Description) != 0 {
		t.Fatalf("expecting nil but retrieved %s\n", grole.Data.Attributes.Description)
	}
	if m, _ := regexp.MatchString("fields=role&include=permissions", grole.Links.Self); !m {
		t.Fatalf("expected link %s does not contain fields query parameter", grole.Links.Self)
	}
	for _, a := range grole.Included {
		permData := &pb.PermissionData{}
		if err := ptypes.UnmarshalAny(a, permData); err != nil {
			t.Fatalf("error in unmarshaling any types %s\n", err)
		} else {
			if permData.Id != perm.Data.Id {
				t.Fatalf("expected id does not match with %d\n", permData.Id)
			}
			if permData.Links.Self != perm.Links.Self {
				t.Fatalf("expected link does not match with %s\n", permData.Links.Self)
			}
			if permData.Attributes.Permission != perm.Data.Attributes.Permission {
				t.Fatalf("expected permission does not match with %s\n", permData.Attributes.Permission)
			}
		}
	}
}

func TestRoleGetAll(t *testing.T) {
	defer testutils.TearDownTest(db, t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()

	client := pb.NewRoleServiceClient(conn)
	for i := 0; i < 28; i++ {
		_, err := client.CreateRole(context.Background(), NewRole(RandString(6)))
		if err != nil {
			t.Fatalf("could not store the role %s\n", err)
		}
	}
	lroles, err := client.ListRoles(context.Background(), &jsonapi.SimpleListRequest{})
	if err != nil {
		t.Fatalf("could not fetch all roles %s\n", err)
	}
	if len(lroles.Data) != 28 {
		t.Fatalf("expected entries does not match %d\n", len(lroles.Data))
	}
	for _, role := range lroles.Data {
		if role.Id < 1 {
			t.Fatalf("expected id does not match %d\n", role.Id)
		}
		if role.Links.Self != fmt.Sprintf("/roles/%d", role.Id) {
			t.Fatalf("expected link does not match %s\n", role.Links.Self)
		}
	}
}

func TestRoleGetAllWithFields(t *testing.T) {
	defer testutils.TearDownTest(db, t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()

	client := pb.NewRoleServiceClient(conn)
	for _, r := range []string{"curator", "manager", "admin", "staff", "user"} {
		_, err := client.CreateRole(context.Background(), NewRole(r))
		if err != nil {
			t.Fatalf("could not store the role %s\n", err)
		}
	}
	lroles, err := client.ListRoles(
		context.Background(),
		&jsonapi.SimpleListRequest{
			Fields: "role",
		})
	if err != nil {
		t.Fatalf("could not fetch all roles %s\n", err)
	}
	if len(lroles.Data) != 5 {
		t.Fatalf("expected entries does not match %d\n", len(lroles.Data))
	}
	if m, _ := regexp.MatchString("fields=role", lroles.Links.Self); !m {
		t.Fatalf("expected link %s does not contain fields query parameter", lroles.Links.Self)
	}
	for _, role := range lroles.Data {
		if len(role.Attributes.Description) != 0 {
			t.Fatalf("expecting nil but retrieved %s\n", role.Attributes.Description)
		}
		if role.Links.Self != fmt.Sprintf("/roles/%d", role.Id) {
			t.Fatalf("expected link does not match %s\n", role.Links.Self)
		}
	}
}

func TestRoleGetAllWithFieldsAndFilter(t *testing.T) {
	defer testutils.TearDownTest(db, t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()

	client := pb.NewRoleServiceClient(conn)
	for _, r := range []string{"curator", "manager", "admin", "staff", "user"} {
		_, err := client.CreateRole(context.Background(), NewRole(r))
		if err != nil {
			t.Fatalf("could not store the role %s\n", err)
		}
	}
	lroles, err := client.ListRoles(
		context.Background(),
		&jsonapi.SimpleListRequest{
			Fields: "role",
			Filter: "role!@er",
		})
	if err != nil {
		t.Fatalf("could not fetch all roles %s\n", err)
	}
	if len(lroles.Data) != 3 {
		t.Fatalf("expected entries does not match %d\n", len(lroles.Data))
	}
	if m, _ := regexp.MatchString("fields=role&filter=role!@er", lroles.Links.Self); !m {
		t.Fatalf("expected link %s does not contain fields query parameter", lroles.Links.Self)
	}
	for _, role := range lroles.Data {
		if len(role.Attributes.Description) != 0 {
			t.Fatalf("expecting nil but retrieved %s\n", role.Attributes.Description)
		}
		if role.Links.Self != fmt.Sprintf("/roles/%d", role.Id) {
			t.Fatalf("expected link does not match %s\n", role.Links.Self)
		}
	}
}

func TestRoleGetAllWithIncludeAndFilter(t *testing.T) {
	defer testutils.TearDownTest(db, t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()

	permClient := pb.NewPermissionServiceClient(conn)
	perm, err := permClient.CreatePermission(context.Background(), NewPermission("fetch", "strain"))
	if err != nil {
		t.Fatalf("could not store the permission %s\n", err)
	}

	client := pb.NewRoleServiceClient(conn)
	for _, r := range []string{"curator", "manager", "admin", "staff", "user"} {
		_, err := client.CreateRole(context.Background(), NewRoleWithPermission(r, perm))
		if err != nil {
			t.Fatalf("could not store the role %s\n", err)
		}
	}
	lroles, err := client.ListRoles(
		context.Background(),
		&jsonapi.SimpleListRequest{
			Include: "permissions",
			Filter:  "role!@er",
		})
	if err != nil {
		t.Fatalf("could not fetch all roles %s\n", err)
	}
	if len(lroles.Data) != 3 {
		t.Fatalf("expected entries does not match %d\n", len(lroles.Data))
	}
	if m, _ := regexp.MatchString("filter=role!@er&include=permissions", lroles.Links.Self); !m {
		t.Fatalf("expected link %s does not contain fields query parameter", lroles.Links.Self)
	}
	counter := 0
	for _, role := range lroles.Data {
		if role.Links.Self != fmt.Sprintf("/roles/%d", role.Id) {
			t.Fatalf("expected link does not match %s\n", role.Links.Self)
		}
		if len(role.Relationships.Permissions.Data) != 1 {
			t.Fatalf("expected included elements does not match %d\n", len(role.Relationships.Permissions.Data))
		}
		if role.Relationships.Permissions.Data[0].Id != perm.Data.Id {
			t.Fatalf(
				"expected id %d does not match %d\n",
				role.Relationships.Permissions.Data[0].Id,
				perm.Data.Id,
			)
		}
		counter++
	}
	if counter != len(lroles.Included) {
		t.Fatalf("relationship resources %d does not match with %d included resources", counter, len(lroles.Included))
	}
	for _, a := range lroles.Included {
		permData := &pb.PermissionData{}
		if err := ptypes.UnmarshalAny(a, permData); err != nil {
			t.Fatalf("error in unmarshaling any types %s\n", err)
		} else {
			if permData.Id != perm.Data.Id {
				t.Fatalf("expected id does not match with %d\n", permData.Id)
			}
			if permData.Links.Self != perm.Links.Self {
				t.Fatalf("expected link does not match with %s\n", permData.Links.Self)
			}
			if permData.Attributes.Permission != perm.Data.Attributes.Permission {
				t.Fatalf("expected permission does not match with %s\n", permData.Attributes.Permission)
			}
		}
	}
}

func TestRoleCreateUserRelationship(t *testing.T) {
	defer testutils.TearDownTest(db, t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()
	uclient := pb.NewUserServiceClient(conn)
	nuser, err := uclient.CreateUser(context.Background(), NewUser("todd@gad.org"))
	if err != nil {
		t.Fatalf("could not store the user %s\n", err)
	}

	client := pb.NewRoleServiceClient(conn)
	nrole, err := client.CreateRole(context.Background(), NewRole("fetcher"))
	if err != nil {
		t.Fatalf("could not store the role %s\n", err)
	}
	_, err = client.CreateUserRelationship(
		context.Background(),
		&jsonapi.DataCollection{
			Id:   nrole.Data.Id,
			Data: []*jsonapi.Data{&jsonapi.Data{Type: "users", Id: nuser.Data.Id}},
		},
	)
	if err != nil {
		t.Fatalf("could not create the relationship with user %s\n", err)
	}
	grole, err := client.GetRole(
		context.Background(),
		&jsonapi.GetRequest{Id: nrole.Data.Id, Include: "users"},
	)
	if err != nil {
		t.Fatalf("could not fetch the role %s\n", err)
	}
	for _, a := range grole.Included {
		uData := &pb.UserData{}
		if err := ptypes.UnmarshalAny(a, uData); err != nil {
			t.Fatalf("error in unmarshaling any types %s\n", err)
		} else {
			if uData.Id != nuser.Data.Id {
				t.Fatalf("expected id does not match with %d\n", uData.Id)
			}
			if uData.Links.Self != nuser.Links.Self {
				t.Fatalf("expected link does not match with %s\n", uData.Links.Self)
			}
			if uData.Attributes.Email != nuser.Data.Attributes.Email {
				t.Fatalf("expected permission does not match with %s\n", uData.Attributes.Email)
			}
		}
	}
}

func TestRoleCreatePermissionRelationship(t *testing.T) {
	defer testutils.TearDownTest(db, t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()

	permClient := pb.NewPermissionServiceClient(conn)
	perm, err := permClient.CreatePermission(context.Background(), NewPermission("fetch", "order"))
	if err != nil {
		t.Fatalf("could not store the permission %s\n", err)
	}

	client := pb.NewRoleServiceClient(conn)
	nrole, err := client.CreateRole(context.Background(), NewRole("fetcher"))
	if err != nil {
		t.Fatalf("could not store the role %s\n", err)
	}
	_, err = client.CreatePermissionRelationship(
		context.Background(),
		&jsonapi.DataCollection{
			Id:   nrole.Data.Id,
			Data: []*jsonapi.Data{&jsonapi.Data{Type: "permissions", Id: perm.Data.Id}},
		},
	)
	if err != nil {
		t.Fatalf("could not create the relationship with permission %s\n", err)
	}
	grole, err := client.GetRole(
		context.Background(),
		&jsonapi.GetRequest{Id: nrole.Data.Id, Include: "permissions"},
	)
	if err != nil {
		t.Fatalf("could not fetch the role %s\n", err)
	}
	for _, a := range grole.Included {
		permData := &pb.PermissionData{}
		if err := ptypes.UnmarshalAny(a, permData); err != nil {
			t.Fatalf("error in unmarshaling any types %s\n", err)
		} else {
			if permData.Id != perm.Data.Id {
				t.Fatalf("expected id does not match with %d\n", permData.Id)
			}
			if permData.Links.Self != perm.Links.Self {
				t.Fatalf("expected link does not match with %s\n", permData.Links.Self)
			}
			if permData.Attributes.Permission != perm.Data.Attributes.Permission {
				t.Fatalf("expected permission does not match with %s\n", permData.Attributes.Permission)
			}
		}
	}
}

func TestRoleUpdatePermissionRelationship(t *testing.T) {
	defer testutils.TearDownTest(db, t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()

	permClient := pb.NewPermissionServiceClient(conn)
	perm, err := permClient.CreatePermission(context.Background(), NewPermission("create", "gene"))
	if err != nil {
		t.Fatalf("could not store the permission %s\n", err)
	}

	client := pb.NewRoleServiceClient(conn)
	nrole, err := client.CreateRole(context.Background(), NewRoleWithPermission("creator", perm))
	if err != nil {
		t.Fatalf("could not store the role %s\n", err)
	}
	uperm, err := permClient.CreatePermission(context.Background(), NewPermission("update", "pathway"))
	if err != nil {
		t.Fatalf("could not store the permission %s\n", err)
	}
	_, err = client.UpdatePermissionRelationship(
		context.Background(),
		&jsonapi.DataCollection{
			Id: nrole.Data.Id,
			Data: []*jsonapi.Data{
				&jsonapi.Data{
					Type: "permissions",
					Id:   uperm.Data.Id,
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("could not update the relationship with permission %s\n", err)
	}

	grole, err := client.GetRole(
		context.Background(),
		&jsonapi.GetRequest{Id: nrole.Data.Id, Include: "permissions"},
	)
	if err != nil {
		t.Fatalf("could not fetch the role %s\n", err)
	}
	for _, a := range grole.Included {
		permData := &pb.PermissionData{}
		if err := ptypes.UnmarshalAny(a, permData); err != nil {
			t.Fatalf("error in unmarshaling any types %s\n", err)
		} else {
			if permData.Id != uperm.Data.Id {
				t.Fatalf("expected id does not match with %d\n", permData.Id)
			}
			if permData.Links.Self != uperm.Links.Self {
				t.Fatalf("expected link does not match with %s\n", permData.Links.Self)
			}
			if permData.Attributes.Permission != uperm.Data.Attributes.Permission {
				t.Fatalf("expected permission does not match with %s\n", permData.Attributes.Permission)
			}
		}
	}
}

func TestRoleDeletePermissionRelationship(t *testing.T) {
	defer testutils.TearDownTest(db, t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()

	permClient := pb.NewPermissionServiceClient(conn)
	perm, err := permClient.CreatePermission(context.Background(), NewPermission("delete", "frontpage"))
	if err != nil {
		t.Fatalf("could not store the permission %s\n", err)
	}

	client := pb.NewRoleServiceClient(conn)
	nrole, err := client.CreateRole(context.Background(), NewRoleWithPermission("deleter", perm))
	if err != nil {
		t.Fatalf("could not store the role %s\n", err)
	}
	_, err = client.DeletePermissionRelationship(
		context.Background(),
		&jsonapi.DataCollection{
			Id: nrole.Data.Id,
			Data: []*jsonapi.Data{
				&jsonapi.Data{
					Type: "permissions",
					Id:   perm.Data.Id,
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("could not delete the relationship with permission %s\n", err)
	}
}

func TestRoleGetPermissionRelationship(t *testing.T) {
	defer testutils.TearDownTest(db, t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()

	permClient := pb.NewPermissionServiceClient(conn)
	perm, err := permClient.CreatePermission(context.Background(), NewPermission("get", "wiki"))
	if err != nil {
		t.Fatalf("could not store the permission %s\n", err)
	}

	client := pb.NewRoleServiceClient(conn)
	nrole, err := client.CreateRole(context.Background(), NewRoleWithPermission("getter", perm))
	if err != nil {
		t.Fatalf("could not store the role %s\n", err)
	}
	nperm, err := client.GetRelatedPermissions(
		context.Background(),
		&jsonapi.RelationshipRequest{
			Id: nrole.Data.Id,
		},
	)
	if err != nil {
		t.Fatalf("could not get relationship permission %s\n", err)
	}
	if perm.Data.Id != nperm.Data[0].Id {
		t.Fatalf("expected id %d does not match retrieved %d id", perm.Data.Id, nperm.Data[0].Id)
	}
	if nperm.Links.Self != nrole.Data.Relationships.Permissions.Links.Related {
		t.Fatalf("expected relationships link does not match %s", nrole.Data.Relationships.Permissions.Links.Related)
	}
}

func TestGetRelatedUsers(t *testing.T) {
	defer testutils.TearDownTest(db, t)
	conn, err := grpc.Dial("localhost"+port, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("could not connect to grpc server %s\n", err)
	}
	defer conn.Close()

	userClient := pb.NewUserServiceClient(conn)
	var allUsers []*pb.User
	for i := 0; i < 40; i++ {
		u, err := userClient.CreateUser(
			context.Background(),
			NewUser(fmt.Sprintf("%s@kramer.com", RandString(10))),
		)
		if err != nil {
			t.Fatalf("could not store the user %s\n", err)
		}
		allUsers = append(allUsers, u)
	}

	client := pb.NewRoleServiceClient(conn)
	nrole, err := client.CreateRole(context.Background(), NewRoleWithMultiUsers("genome-curator", allUsers))
	if err != nil {
		t.Fatalf("could not store the role %s\n", err)
	}
	rusers, err := client.GetRelatedUsers(
		context.Background(),
		&jsonapi.RelationshipRequestWithPagination{
			Id: nrole.Data.Id,
		},
	)
	if err != nil {
		t.Fatalf("could not get related users %s\n", err)
	}
	if len(rusers.Data) != 10 {
		t.Fatalf("expected entries does not match %d\n", len(rusers.Data))
	}
	if rusers.Links.Self != rusers.Links.First {
		t.Fatalf("self should match with first link %s", rusers.Links.First)
	}
	if m, _ := regexp.MatchString(`roles\/\d+\/users\?pagenum=2&pagesize=10`, rusers.Links.Next); !m {
		t.Fatalf("expected next link does not match %s", rusers.Links.Next)
	}
	if m, _ := regexp.MatchString(`roles\/\d+\/users\?pagenum=4&pagesize=10`, rusers.Links.Last); !m {
		t.Fatalf("expected last link does not match %s", rusers.Links.Last)
	}
	page := rusers.Meta.Pagination
	if page.Records != 40 {
		t.Logf("expected total no of records does not match %d\n", page.Records)
	}
	if page.Size != 10 {
		t.Logf("expected page size does not match %d\n", page.Size)
	}
	if page.Number != 1 {
		t.Logf("expected page number does not match %d\n", page.Number)
	}
	if page.Total != 4 {
		t.Logf("expected no of pages does not match %d\n", page.Total)
	}

	musers, err := client.GetRelatedUsers(
		context.Background(),
		&jsonapi.RelationshipRequestWithPagination{
			Id:       nrole.Data.Id,
			Pagesize: 5,
			Pagenum:  4,
		},
	)
	if err != nil {
		t.Fatalf("could not get related users %s\n", err)
	}
	if len(musers.Data) != 5 {
		t.Fatalf("expected entries does not match %d\n", len(musers.Data))
	}
	if m, _ := regexp.MatchString(`roles\/\d+\/users\?pagenum=5&pagesize=5`, musers.Links.Next); !m {
		t.Fatalf("expected next link does not match %s", musers.Links.Next)
	}
	if m, _ := regexp.MatchString(`roles\/\d+\/users\?pagenum=8&pagesize=5`, musers.Links.Last); !m {
		t.Fatalf("expected last link does not match %s", musers.Links.Last)
	}
	if m, _ := regexp.MatchString(`roles\/\d+\/users\?pagenum=3&pagesize=5`, musers.Links.Prev); !m {
		t.Fatalf("expected last link does not match %s", musers.Links.Last)
	}
	mpage := musers.Meta.Pagination
	if mpage.Size != 5 {
		t.Logf("expected page size does not match %d\n", mpage.Size)
	}
	if mpage.Number != 4 {
		t.Logf("expected page number does not match %d\n", mpage.Number)
	}
	if mpage.Total != 8 {
		t.Logf("expected no of pages does not match %d\n", mpage.Total)
	}
}
