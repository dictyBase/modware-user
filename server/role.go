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
	roleDbTable = "auth_role role"
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

func (s *RoleService) existsResource(id int64) error {
	return s.Dbh.Select("auth_role_id").From("auth_role").
		Where("auth_role_id = $1", id).Exec()
}

func (s *RoleService) getResourceWithSelectedAttr(id int64) (*user.Role, error) {
	drole := &dbRole{}
	columns := s.fieldsToColumns(s.params.Fields)
	err := s.Dbh.Select(columns...).From(roleDbTable).
		Where("role.auth_role_id = $1", id).QueryStruct(drole)
	if err != nil {
		return &user.Role{}, err
	}
	return s.buildResource(id, s.dbToResourceAttributes(drole)), nil
}

func (s *RoleService) getResource(id int64) (*user.Role, error) {
	drole := &dbRole{}
	err := s.Dbh.Select("role.*").
		From(roleDbTable).
		Where("role.auth_role_id = $1", id).
		QueryStruct(drole)
	if err != nil {
		return &user.Role{}, err
	}
	return s.buildResource(id, s.dbToResourceAttributes(drole)), nil
}

func (s *RoleService) getAllRows() ([]*dbRole, error) {
	var dbrows []*dbRole
	err := s.Dbh.Select("role.*").
		From(roleDbTable).
		QueryStructs(dbrows)
	return dbrows, err
}

func (s *RoleService) getAllRowsWithPaging(pagenum int64, pagesize int64) ([]*dbRole, error) {
	var dbrows []*dbRole
	err := s.Dbh.Select("role.*").
		From(roleDbTable).
		Paginate(uint64(pagenum), uint64(pagesize)).
		QueryStructs(dbrows)
	return dbrows, err
}

func (s *RoleService) getAllSelectedRowsWithPaging(pagenum, pagesize int64) ([]*dbRole, error) {
	var dbrows []*dbRole
	columns := s.MapFieldsToColumns(s.params.Fields)
	err := s.Dbh.Select(columns...).
		From(roleDbTable).
		Paginate(uint64(pageNum), uint64(pageSize)).
		QueryStructs(dbrows)
	return dbrows, err
}

func (s *RoleService) getAllFilteredRowsWithPaging(pagenum, pagesize int64) ([]*dbRole, error) {
	var dbrows []*dbRole
	err := s.Dbh.Select("role.*").
		From(roleDbTable).
		Scope(
			aphgrpc.FilterToWhereClause(s, s.params.Filter),
			aphgrpc.FilterToBindValue(s.params.Filter)...,
		).
		Paginate(uint64(pageNum), uint64(pageSize)).
		QueryStructs(dbrows)
	return dbrows, err
}

func (s *RoleService) getAllSelectedFilteredRowsWithPaging(pagenum, pagesize int64) ([]*dbRole, error) {
	var dbrows []*dbRole
	columns := s.MapFieldsToColumns(s.params.Fields)
	err := s.Dbh.Select(columns...).
		From(roleDbTable).
		Scope(
			aphgrpc.FilterToWhereClause(s, s.params.Filter),
			aphgrpc.FilterToBindValue(s.params.Filter)...,
		).
		Paginate(uint64(pageNum), uint64(pageSize)).
		QueryStructs(dbrows)
	return dbrows, err
}

func (s *RoleService) getPermissionResourceData(id int64) ([]*user.PermissionData, error) {
	var dbrows []*dbPermission
	var pdata []*user.PermissionData
	err := s.Dbh.Select("perm.*").From(`
			auth_role_permission
			JOIN permission perm
			ON auth_role_permission.auth_permission_id = perm.permission_id
		`).Where("auth_role_permission.auth_role_id = $1", id).
		QueryStruct(dbrows)
	if err != nil {
		return pdata, err
	}
	return NewPermissionService(
		s.Dbh,
		s.GetPathPrefix(),
		s.GetBaseURL(),
	).dbToCollResourceData(dbrows), nil
}

func (s *RoleService) getUserResourceData(id int64) ([]*user.UserData, error) {
	var dbrows []*dbUser
	var udata []*user.UserData
	err := s.Dbh.Select("user.*", "uinfo.*").From(`
		auth_user_role
		JOIN auth_user user
		ON auth_user_role.auth_user_id = user.auth_user_id
		JOIN auth_user_info uinfo
		ON uinfo.auth_user_id = user.auth_user_id
	`).Where("auth_user_role.auth_role_id = $1", id).QueryStruct(dbrows)
	if err != nil {
		return udata, err
	}
	return NewUserService(
		s.Dbh,
		s.GetPathPrefix(),
		s.GetBaseURL(),
	).dbToCollResourceData(dbrows), nil
}

func (s *RoleService) buildUserResourceIdentifiers(users []*user.User) []*jsonapi.Data {
	jdata := make([]*jsonapi.Data, len(users))
	for i, r := range users {
		jdata[i] = &jsonapi.Data{
			Type: r.Type,
			Id:   r.Id,
		}
	}
	return jdata
}

func (s *RoleService) buildPermissionResourceIdentifiers(perms []*user.Permission) []*jsonapi.Data {
	jdata := make([]*jsonapi.Data, len(perms))
	for i, r := range perms {
		jdata[i] = &jsonapi.Data{
			Type: r.Type,
			Id:   r.Id,
		}
	}
	return jdata
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
		u, err := s.getUserResourceData(dbrows[i].AuthRoleId)
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
		p, err := s.getPermissionResourceData(dbrows[i].AuthRoleId)
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
