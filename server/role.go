package server

import (
	"github.com/dictyBase/apihelpers/aphgrpc"
	"github.com/dictyBase/go-genproto/dictybaseapis/api/jsonapi"
	"github.com/dictyBase/go-genproto/dictybaseapis/user"
	"github.com/golang/protobuf/ptypes/any"
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

func (s *RoleService) buildResource(id int64, attr *user.RoleAttributes) *user.Role {
	return &user.Role{
		Data: s.buildResourceData(id, attr),
	}
}

func (s *RoleService) dbToResourceAttributes(r *dbRole) *user.RoleAttributes {
	return &user.RoleAttributes{
		Role:        r.Role,
		Description: r.Description,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

func (s *RoleService) dbToCollResourceData(dbrows []*dbRole) []*user.RoleData {
	var rdata []*user.RoleData
	for _, r := range dbrows {
		rdata = append(rdata, s.buildResourceData(r.AuthRoleId, s.dbToResourceAttributes(r)))
	}
	return rdata
}

func (s *RoleService) dbToCollResource(dbrows []*dbRole) (*user.RoleCollection, error) {
	return &user.RoleCollection{
		Data: s.dbToCollResourceData(dbrows),
		Links: &jsonapi.PaginationLinks{
			Self: s.genCollResourceSelfLink(),
		},
	}, nil
}

func (s *RoleService) dbToCollResourceWithPagination(count int64, dbrows []*dbRole, pagenum, pagesize int64) (*user.RoleCollection, err) {
	rdata := s.dbToCollResourceData(dbrows)
	jsLinks, pages := s.getPagination(count, pagenum, pagesize)
	return &user.RoleCollection{
		Data:  rdata,
		Links: jsLinks,
		Meta: &jsonapi.Meta{
			Pagination: &jsonapi.Pagination{
				Records: count,
				Total:   pages,
				Size:    pagesize,
				Number:  pagenum,
			},
		},
	}, nil
}

func (s *RoleService) dbToCollResourceWithRelAndPagination(count int64, dbrows []*dbRole, pagenum, pagesize int64) (*user.RoleCollection, err) {
	rdata := s.dbToCollResourceData(dbrows)
	// related users
	var users []*user.User
	for i, _ := range rdata {
		u, err := s.getUserResource(dbrows[i].AuthRoleId)
		if err != nil {
			return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
		}
		rdata[i].Relationships.Users.Data = s.buildUserResourceIdentifiers(u)
		users = append(users, u...)
	}
	incUsers, err := aphgrpc.ConvertAllToAny(users)
	if err != nil {
		return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
	}
	// related permissions
	var perms []*user.Permission
	for i, _ := range rdata {
		p, err := s.getPermissionResource(dbrows[i].AuthRoleId)
		if err != nil {
			return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
		}
		rdata[i].Relationships.Permissions.Data = s.buildPermissionResourceIdentifiers(p)
		perms = append(perms, p...)
	}
	incPerms, err := aphgrpc.ConvertAllToAny(perms)
	if err != nil {
		return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
	}
	var allInc []*any.Any
	allInc = append(allInc, incUsers...)
	allInc = append(allInc, incPerms...)
	jsLinks, pages := s.getPagination(count, pagenum, pagesize)
	return &user.RoleCollection{
		Data:     rdata,
		Links:    jsLinks,
		Included: allInc,
		Meta: &jsonapi.Meta{
			Pagination: &jsonapi.Pagination{
				Records: count,
				Total:   pages,
				Size:    pagesize,
				Number:  pagenum,
			},
		},
	}, nil
}

func (s *RoleService) attrTodbRole(attr *user.RoleAttributes) *dbRole {
	return &dbRole{
		Role:        attr.Role,
		Description: attr.Description,
		CreatedAt:   attr.CreatedAt,
		UpdatedAt:   attr.UpdatedAt,
	}
}
