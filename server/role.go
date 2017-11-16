package server

import (
	"github.com/dictyBase/apihelpers/aphgrpc"
	"github.com/dictyBase/go-genproto/dictybaseapis/api/jsonapi"
	"github.com/dictyBase/go-genproto/dictybaseapis/user"
	dat "gopkg.in/mgutz/dat.v1"
	runner "gopkg.in/mgutz/dat.v1/sqlx-runner"
)

const (
	roleDbTable = "auth_role"
)

type dbRole struct {
	AuthRoleId  string       `db:"auth_role_id"`
	Role        string       `db:"role"`
	Description string       `db:"description"`
	CreatedAt   dat.NullTime `db:"created_at"`
	UpdatedAt   dat.NullTime `db:"updated_at"`
}

type RoleService struct {
	*aphgrpc.Service
}

func NewRoleService(dbh *runner.DB, pathPrefix string, baseURL string) *RoleService {
	return &RoleService{
		&aphgrpc.Service{
			baseURL:    baseURL,
			resource:   "roles",
			Dbh:        dbh,
			pathPrefix: pathPrefix,
			include:    []string{"users", "permissions"},
			filterToColumns: map[string]string{
				"role":        "role.role",
				"description": "role.description",
			},
			fieldsToColumns: map[string]string{
				"role":        "role.role",
				"description": "role.description",
				"created_at":  "role.created_at",
				"updated_at":  "role.updated_at",
			},
			requiredAttrs: []string{"Role"},
		},
	}
}

func (s *RoleService) buildResourceData(id int64, attr *user.RoleAttributes) *user.RoleData {
	return &user.RoleData{
		Type:       s.GetResourceName(),
		Id:         id,
		Attributes: attr,
		Relationships: &user.ExistingRoleRelationships{
			Users: &user.ExistingRoleRelationships_Users{
				Links: &jsonapi.Links{
					Self:    aphgrpc.GenSelfRelationshipLink(s, "users", id),
					Related: aphgrpc.GenRelatedRelationshipLink(s, "users", id),
				},
			},
			Permissions: &user.ExistingRoleRelationships_Permissions{
				Links: &jsonapi.Links{
					Self:    aphgrpc.GenSelfRelationshipLink(s, "permissions", id),
					Related: aphgrpc.GenRelatedRelationshipLink(s, "permissions", id),
				},
			},
		},
		Links: &jsonapi.Links{
			Self: s.genSingularResSelfLink(id),
		},
	}
}

func (s *RoleService) dbToResourceAttributes(drole *dbRole) *user.RoleAttributes {
	return &user.RoleAttributes{
		Role:        drole.Role,
		Description: drole.Description,
		CreatedAt:   drole.CreatedAt,
		UpdatedAt:   drole.UpdatedAt,
	}
}

func (s *RoleService) attrTodbRole(attr *user.RoleAttributes) *dbRole {
	return &dbRole{
		Role:        attr.Role,
		Description: attr.Description,
		CreatedAt:   attr.CreatedAt,
		UpdatedAt:   attr.UpdatedAt,
	}
}
