package server

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/dictyBase/apihelpers/aphgrpc"
	"github.com/dictyBase/go-genproto/dictybaseapis/api/jsonapi"
	"github.com/dictyBase/go-genproto/dictybaseapis/user"
	"github.com/fatih/structs"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/empty"
	dat "gopkg.in/mgutz/dat.v2/dat"
	runner "gopkg.in/mgutz/dat.v2/sqlx-runner"
)

const (
	roleDbTable    = "auth_role"
	roleDbTblAlias = "auth_role role"
)

var roleCols = []string{"auth_role_id", "role", "created_at", "updated_at"}

type dbRole struct {
	AuthRoleId  int64        `db:"auth_role_id"`
	Role        string       `db:"role"`
	Description string       `db:"description"`
	CreatedAt   dat.NullTime `db:"created_at"`
	UpdatedAt   dat.NullTime `db:"updated_at"`
}

type RoleService struct {
	*aphgrpc.Service
}

func NewRoleService(dbh *runner.DB, pathPrefix string) *RoleService {
	return &RoleService{
		&aphgrpc.Service{
			Resource:   "roles",
			Dbh:        dbh,
			PathPrefix: pathPrefix,
			Include:    []string{"users", "permissions"},
			FilToColumns: map[string]string{
				"role":        "role.role",
				"description": "role.description",
			},
			FieldsToColumns: map[string]string{
				"role":        "role.role",
				"description": "role.description",
				"created_at":  "role.created_at",
				"updated_at":  "role.updated_at",
			},
			ReqAttrs: []string{"Role"},
		},
	}
}

func (s *RoleService) GetRole(ctx context.Context, r *jsonapi.GetRequest) (*user.Role, error) {
	params, md, err := aphgrpc.ValidateAndParseGetParams(s, r)
	if err != nil {
		grpc.SetTrailer(ctx, md)
		return &user.Role{}, status.Error(codes.InvalidArgument, err.Error())
	}
	s.Params = params
	s.ListMethod = false
	s.SetBaseURL(ctx)
	switch {
	case params.HasFields && params.HasInclude:
		s.IncludeStr = r.Include
		s.FieldsStr = r.Fields
		role, err := s.getResourceWithSelectedAttr(r.Id)
		if err != nil {
			return &user.Role{}, aphgrpc.HandleError(ctx, err)
		}
		err = s.buildResourceRelationships(r.Id, role)
		if err != nil {
			return &user.Role{}, aphgrpc.HandleError(ctx, err)
		}
		return role, nil
	case params.HasFields:
		s.FieldsStr = r.Fields
		role, err := s.getResourceWithSelectedAttr(r.Id)
		if err != nil {
			return &user.Role{}, aphgrpc.HandleError(ctx, err)
		}
		return role, nil
	case params.HasInclude:
		s.IncludeStr = r.Include
		role, err := s.getResource(r.Id)
		if err != nil {
			return &user.Role{}, aphgrpc.HandleError(ctx, err)
		}
		err = s.buildResourceRelationships(r.Id, role)
		if err != nil {
			return &user.Role{}, aphgrpc.HandleError(ctx, err)
		}
		return role, nil
	default:
		role, err := s.getResource(r.Id)
		if err != nil {
			return &user.Role{}, aphgrpc.HandleError(ctx, err)
		}
		return role, nil
	}
}

func (s *RoleService) GetRelatedUsers(ctx context.Context, r *jsonapi.RelationshipRequestWithPagination) (*user.UserCollection, error) {
	// For pagination based data retreival
	// 1. Get count of all rows
	count, err := s.getRelatedUsersCount(r.Id)
	if err != nil {
		return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
	}
	// 2. Has pagination query paramters
	var pagenum, pagesize int64
	if aphgrpc.HasRelatedPagination(r) {
		if r.Pagenum == 0 {
			pagenum = aphgrpc.DefaultPagenum
		} else if r.Pagesize == 0 {
			pagesize = aphgrpc.DefaultPagesize
		} else {
			pagenum = r.Pagenum
			pagesize = r.Pagesize
		}
		// 3. Without any pagination parameters(use default page parameters)
	} else {
		pagenum = aphgrpc.DefaultPagenum
		pagesize = aphgrpc.DefaultPagesize
	}
	udata, err := s.getUserResourceDataWithPagination(r.Id, pagenum, pagesize)
	if err != nil {
		return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
	}
	s.SetBaseURL(ctx)
	pageLinks, pages := s.GetRelatedPagination(r.Id, count, pagenum, pagesize, "users")
	return &user.UserCollection{
		Data:  udata,
		Links: pageLinks,
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

func (s *RoleService) GetRelatedPermissions(ctx context.Context, r *jsonapi.RelationshipRequest) (*user.PermissionCollection, error) {
	pdata, err := s.getPermissionResourceData(r.Id)
	if err != nil {
		return &user.PermissionCollection{}, aphgrpc.HandleError(ctx, err)
	}
	return &user.PermissionCollection{
		Data: pdata,
		Links: &jsonapi.Links{
			Self: s.GenCollResourceRelSelfLink(r.Id, "permissions"),
		},
	}, nil
}

func (s *RoleService) ListRoles(ctx context.Context, r *jsonapi.SimpleListRequest) (*user.RoleCollection, error) {
	params, md, err := aphgrpc.ValidateAndParseSimpleListParams(s, r)
	if err != nil {
		grpc.SetTrailer(ctx, md)
		return &user.RoleCollection{}, status.Error(codes.InvalidArgument, err.Error())
	}
	s.Params = params
	s.ListMethod = true
	s.SetBaseURL(ctx)
	// request without any pagination query parameters
	switch {
	case params.HasFields && params.HasFilter && params.HasInclude:
		s.FieldsStr = r.Fields
		s.FilterStr = r.Filter
		s.IncludeStr = r.Include
		dbRoles, err := s.getAllSelectedFilteredRows()
		if err != nil {
			return &user.RoleCollection{}, aphgrpc.HandleError(ctx, err)
		}
		r, err := s.dbToCollResourceWithRel(dbRoles)
		if err != nil {
			return &user.RoleCollection{}, aphgrpc.HandleError(ctx, err)
		}
		return r, nil
	case params.HasFields && params.HasFilter:
		s.FieldsStr = r.Fields
		s.FilterStr = r.Filter
		dbRoles, err := s.getAllSelectedFilteredRows()
		if err != nil {
			return &user.RoleCollection{}, aphgrpc.HandleError(ctx, err)
		}
		return s.dbToCollResource(dbRoles), nil
	case params.HasFields && params.HasInclude:
		s.FieldsStr = r.Fields
		s.IncludeStr = r.Include
		dbRoles, err := s.getAllSelectedRows()
		if err != nil {
			return &user.RoleCollection{}, aphgrpc.HandleError(ctx, err)
		}
		r, err := s.dbToCollResourceWithRel(dbRoles)
		if err != nil {
			return &user.RoleCollection{}, aphgrpc.HandleError(ctx, err)
		}
		return r, nil
	case params.HasFilter && params.HasInclude:
		s.IncludeStr = r.Include
		s.FilterStr = r.Filter
		dbRoles, err := s.getAllFilteredRows()
		if err != nil {
			return &user.RoleCollection{}, aphgrpc.HandleError(ctx, err)
		}
		r, err := s.dbToCollResourceWithRel(dbRoles)
		if err != nil {
			return &user.RoleCollection{}, aphgrpc.HandleError(ctx, err)
		}
		return r, nil
	case params.HasFields:
		s.FieldsStr = r.Fields
		dbRoles, err := s.getAllSelectedRows()
		if err != nil {
			return &user.RoleCollection{}, aphgrpc.HandleError(ctx, err)
		}
		return s.dbToCollResource(dbRoles), nil
	case params.HasFilter:
		s.FilterStr = r.Filter
		dbRoles, err := s.getAllFilteredRows()
		if err != nil {
			return &user.RoleCollection{}, aphgrpc.HandleError(ctx, err)
		}
		return s.dbToCollResource(dbRoles), nil
	case params.HasInclude:
		s.IncludeStr = r.Include
		dbRoles, err := s.getAllRows()
		if err != nil {
			return &user.RoleCollection{}, aphgrpc.HandleError(ctx, err)
		}
		r, err := s.dbToCollResourceWithRel(dbRoles)
		if err != nil {
			return &user.RoleCollection{}, aphgrpc.HandleError(ctx, err)
		}
		return r, nil
	default:
		dbRoles, err := s.getAllRows()
		if err != nil {
			return &user.RoleCollection{}, aphgrpc.HandleError(ctx, err)
		}
		return s.dbToCollResource(dbRoles), nil
	}
}

func (s *RoleService) CreateRole(ctx context.Context, r *user.CreateRoleRequest) (*user.Role, error) {
	dbrole := s.attrTodbRole(r.Data.Attributes)
	rcolumns := aphgrpc.GetDefinedTags(dbrole, "db")
	if len(rcolumns) > 0 {
		err := s.Dbh.InsertInto("auth_role").
			Columns(rcolumns...).
			Record(dbrole).
			Returning(roleCols...).
			QueryStruct(dbrole)
		if err != nil {
			grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseInsert)
			return &user.Role{}, status.Error(codes.Internal, err.Error())
		}
	}
	roleId := dbrole.AuthRoleId
	rstruct := structs.New(r).Field("Data").Field("Relationships")
	if !rstruct.IsZero() {
		if !rstruct.Field("Users").IsZero() {
			for _, u := range r.Data.Relationships.Users.Data {
				_, err := s.Dbh.InsertInto("auth_user_role").
					Columns("auth_user_id", "auth_role_id").
					Values(u.Id, roleId).Exec()
				if err != nil {
					grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseInsert)
					return &user.Role{}, status.Error(codes.Internal, err.Error())
				}
			}
		}
		if !rstruct.Field("Permissions").IsZero() {
			for _, p := range r.Data.Relationships.Permissions.Data {
				_, err := s.Dbh.InsertInto("auth_role_permission").
					Columns("auth_role_id", "auth_permission_id").
					Values(roleId, p.Id).Exec()
				if err != nil {
					grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseInsert)
					return &user.Role{}, status.Error(codes.Internal, err.Error())
				}
			}
		}
	}
	s.SetBaseURL(ctx)
	grpc.SetTrailer(ctx, metadata.Pairs("method", "POST"))
	return s.buildResource(roleId, s.dbToResourceAttributes(dbrole)), nil
}

func (s *RoleService) CreateUserRelationship(ctx context.Context, r *jsonapi.DataCollection) (*empty.Empty, error) {
	result, err := s.existsResource(r.Id)
	if err != nil {
		return &empty.Empty{}, aphgrpc.HandleError(ctx, err)
	}
	if !result {
		grpc.SetTrailer(ctx, aphgrpc.ErrNotFound)
		return &empty.Empty{}, status.Error(codes.NotFound, fmt.Sprintf("id %d not found", r.Id))
	}
	for _, ud := range r.Data {
		res, err := s.Dbh.Select("aurole.auth_user_role_id").
			From("auth_user_role aurole").
			Where("aurole.auth_role_id = $1 AND aurole.auth_user_role_id = $2", r.Id, ud.Id).
			Exec()
		if err != nil {
			grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseInsert)
			return &empty.Empty{}, status.Error(codes.Internal, err.Error())
		}
		if res.RowsAffected != 1 {
			_, err := s.Dbh.InsertInto("auth_user_role").
				Columns("auth_role_id", "auth_user_id").
				Values(r.Id, ud.Id).Exec()
			if err != nil {
				grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseInsert)
				return &empty.Empty{}, status.Error(codes.Internal, err.Error())

			}
		}
	}
	grpc.SetTrailer(ctx, metadata.Pairs("method", "POST_NO_CONTENT"))
	return &empty.Empty{}, nil
}

func (s *RoleService) CreatePermissionRelationship(ctx context.Context, r *jsonapi.DataCollection) (*empty.Empty, error) {
	result, err := s.existsResource(r.Id)
	if err != nil {
		return &empty.Empty{}, aphgrpc.HandleError(ctx, err)
	}
	if !result {
		grpc.SetTrailer(ctx, aphgrpc.ErrNotFound)
		return &empty.Empty{}, status.Error(codes.NotFound, fmt.Sprintf("id %d not found", r.Id))
	}
	for _, pd := range r.Data {
		res, err := s.Dbh.Select("auth_role_permission.auth_role_permission_id").
			From("auth_role_permission").
			Where("auth_role_permission.auth_role_id = $1 AND auth_role_permission.auth_permission_id = $2", r.Id, pd.Id).
			Exec()
		if err != nil {
			grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseInsert)
			return &empty.Empty{}, status.Error(codes.Internal, err.Error())
		}
		if res.RowsAffected != 1 {
			_, err := s.Dbh.InsertInto("auth_role_permission").
				Columns("auth_role_id", "auth_permission_id").
				Values(r.Id, pd.Id).Exec()
			if err != nil {
				grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseInsert)
				return &empty.Empty{}, status.Error(codes.Internal, err.Error())

			}
		}
	}
	grpc.SetTrailer(ctx, metadata.Pairs("method", "POST_NO_CONTENT"))
	return &empty.Empty{}, nil
}

func (s *RoleService) UpdateRole(ctx context.Context, r *user.UpdateRoleRequest) (*user.Role, error) {
	result, err := s.existsResource(r.Id)
	if err != nil {
		return &user.Role{}, aphgrpc.HandleError(ctx, err)
	}
	if !result {
		grpc.SetTrailer(ctx, aphgrpc.ErrNotFound)
		return &user.Role{}, status.Error(codes.NotFound, fmt.Sprintf("id %d not found", r.Id))
	}
	dbrole := s.attrTodbRole(r.Data.Attributes)
	rmap := aphgrpc.GetDefinedTagsWithValue(dbrole, "db")
	if len(rmap) > 0 {
		err := s.Dbh.Update(roleDbTable).SetMap(rmap).
			Where("auth_role_id = $1", r.Data.Id).Returning(roleCols...).
			QueryStruct(dbrole)
		if err != nil {
			grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseUpdate)
			return &user.Role{}, status.Error(codes.Internal, err.Error())
		}
	}
	rstruct := structs.New(r).Field("Data").Field("Relationships")
	if !rstruct.IsZero() {
		if !rstruct.Field("Users").IsZero() {
			for _, u := range r.Data.Relationships.Users.Data {
				_, err := s.Dbh.Update("auth_user_role").
					Set("auth_user_id", u.Id).
					Where("auth_role_id = $1", r.Data.Id).Exec()
				if err != nil {
					grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseUpdate)
					return &user.Role{}, status.Error(codes.Internal, err.Error())
				}
			}
		}
		if !rstruct.Field("Permissions").IsZero() {
			for _, p := range r.Data.Relationships.Permissions.Data {
				_, err := s.Dbh.Update("auth_role_permission").
					Set("auth_permission_id", p.Id).
					Where("auth_role_id = $1", r.Data.Id).Exec()
				if err != nil {
					grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseUpdate)
					return &user.Role{}, status.Error(codes.Internal, err.Error())
				}
			}
		}
	}
	s.SetBaseURL(ctx)
	return s.buildResource(dbrole.AuthRoleId, s.dbToResourceAttributes(dbrole)), nil
}

func (s *RoleService) UpdateUserRelationship(ctx context.Context, r *jsonapi.DataCollection) (*empty.Empty, error) {
	result, err := s.existsResource(r.Id)
	if err != nil {
		return &empty.Empty{}, aphgrpc.HandleError(ctx, err)
	}
	if !result {
		grpc.SetTrailer(ctx, aphgrpc.ErrNotFound)
		return &empty.Empty{}, status.Error(codes.NotFound, fmt.Sprintf("id %d not found", r.Id))
	}
	_, err = s.Dbh.DeleteFrom("auth_user_role").
		Where("auth_user_role.auth_role_id = $1", r.Id).
		Exec()
	if err != nil {
		grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseUpdate)
		return &empty.Empty{}, status.Error(codes.Internal, err.Error())
	}
	for _, ud := range r.Data {
		_, err := s.Dbh.InsertInto("auth_user_role").
			Columns("auth_role_id", "auth_user_id").
			Values(r.Id, ud.Id).Exec()
		if err != nil {
			grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseUpdate)
			return &empty.Empty{}, status.Error(codes.Internal, err.Error())
		}
	}
	return &empty.Empty{}, nil
}

func (s *RoleService) UpdatePermissionRelationship(ctx context.Context, r *jsonapi.DataCollection) (*empty.Empty, error) {
	result, err := s.existsResource(r.Id)
	if err != nil {
		return &empty.Empty{}, aphgrpc.HandleError(ctx, err)
	}
	if !result {
		grpc.SetTrailer(ctx, aphgrpc.ErrNotFound)
		return &empty.Empty{}, status.Error(codes.NotFound, fmt.Sprintf("id %d not found", r.Id))
	}
	_, err = s.Dbh.DeleteFrom("auth_role_permission").
		Where("auth_role_permission.auth_role_id = $1", r.Id).
		Exec()
	if err != nil {
		grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseUpdate)
		return &empty.Empty{}, status.Error(codes.Internal, err.Error())
	}
	for _, pd := range r.Data {
		_, err := s.Dbh.InsertInto("auth_role_permission").
			Columns("auth_role_id", "auth_permission_id").
			Values(r.Id, pd.Id).Exec()
		if err != nil {
			grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseUpdate)
			return &empty.Empty{}, status.Error(codes.Internal, err.Error())
		}
	}
	return &empty.Empty{}, nil
}

func (s *RoleService) DeleteRole(ctx context.Context, r *jsonapi.DeleteRequest) (*empty.Empty, error) {
	result, err := s.existsResource(r.Id)
	if err != nil {
		return &empty.Empty{}, aphgrpc.HandleError(ctx, err)
	}
	if !result {
		grpc.SetTrailer(ctx, aphgrpc.ErrNotFound)
		return &empty.Empty{}, status.Error(codes.NotFound, fmt.Sprintf("id %d not found", r.Id))
	}
	_, err = s.Dbh.DeleteFrom(roleDbTable).Where("auth_role_id = $1", r.Id).Exec()
	if err != nil {
		grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseDelete)
		return &empty.Empty{}, status.Error(codes.Internal, err.Error())
	}
	return &empty.Empty{}, nil
}

func (s *RoleService) DeleteUserRelationship(ctx context.Context, r *jsonapi.DataCollection) (*empty.Empty, error) {
	result, err := s.existsResource(r.Id)
	if err != nil {
		return &empty.Empty{}, aphgrpc.HandleError(ctx, err)
	}
	if !result {
		grpc.SetTrailer(ctx, aphgrpc.ErrNotFound)
		return &empty.Empty{}, status.Error(codes.NotFound, fmt.Sprintf("id %d not found", r.Id))
	}
	for _, ud := range r.Data {
		_, err := s.Dbh.DeleteFrom("auth_user_role").
			Where("auth_user_role.auth_role_id = $1 AND auth_user_role.auth_user_id = $2", r.Id, ud.Id).
			Exec()
		if err != nil {
			grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseDelete)
			return &empty.Empty{}, status.Error(codes.Internal, err.Error())
		}
	}
	return &empty.Empty{}, nil
}

func (s *RoleService) DeletePermissionRelationship(ctx context.Context, r *jsonapi.DataCollection) (*empty.Empty, error) {
	result, err := s.existsResource(r.Id)
	if err != nil {
		return &empty.Empty{}, aphgrpc.HandleError(ctx, err)
	}
	if !result {
		grpc.SetTrailer(ctx, aphgrpc.ErrNotFound)
		return &empty.Empty{}, status.Error(codes.NotFound, fmt.Sprintf("id %d not found", r.Id))
	}
	for _, pd := range r.Data {
		_, err := s.Dbh.DeleteFrom("auth_role_permission").
			Where("auth_role_permission.auth_role_id = $1 AND auth_role_permission.auth_permission_id = $2", r.Id, pd.Id).
			Exec()
		if err != nil {
			grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseDelete)
			return &empty.Empty{}, status.Error(codes.Internal, err.Error())
		}
	}
	return &empty.Empty{}, nil
}

func (s *RoleService) existsResource(id int64) (bool, error) {
	r, err := s.Dbh.Select("auth_role_id").From("auth_role").
		Where("auth_role_id = $1", id).Exec()
	if err != nil {
		return false, err
	}
	if r.RowsAffected != 1 {
		return false, nil
	}
	return true, nil
}

func (s *RoleService) getResourceWithSelectedAttr(id int64) (*user.Role, error) {
	drole := &dbRole{}
	columns := s.MapFieldsToColumns(s.Params.Fields)
	err := s.Dbh.Select(columns...).From(roleDbTblAlias).
		Where("role.auth_role_id = $1", id).QueryStruct(drole)
	if err != nil {
		return &user.Role{}, err
	}
	return s.buildResource(id, s.dbToResourceAttributes(drole)), nil
}

func (s *RoleService) getResource(id int64) (*user.Role, error) {
	drole := &dbRole{}
	err := s.Dbh.Select("role.*").
		From(roleDbTblAlias).
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
		From(roleDbTblAlias).
		QueryStructs(&dbrows)
	return dbrows, err
}

func (s *RoleService) getAllSelectedRows() ([]*dbRole, error) {
	var dbrows []*dbRole
	columns := s.MapFieldsToColumns(s.Params.Fields)
	err := s.Dbh.Select(columns...).
		From(roleDbTblAlias).
		QueryStructs(&dbrows)
	return dbrows, err
}

func (s *RoleService) getAllFilteredRows() ([]*dbRole, error) {
	var dbrows []*dbRole
	err := s.Dbh.Select("role.*").
		From(roleDbTblAlias).
		Scope(
			aphgrpc.FilterToWhereClause(s, s.Params.Filters),
			aphgrpc.FilterToBindValue(s.Params.Filters)...,
		).
		QueryStructs(&dbrows)
	return dbrows, err
}

func (s *RoleService) getAllSelectedFilteredRows() ([]*dbRole, error) {
	var dbrows []*dbRole
	columns := s.MapFieldsToColumns(s.Params.Fields)
	err := s.Dbh.Select(columns...).
		From(roleDbTblAlias).
		Scope(
			aphgrpc.FilterToWhereClause(s, s.Params.Filters),
			aphgrpc.FilterToBindValue(s.Params.Filters)...,
		).
		QueryStructs(&dbrows)
	return dbrows, err
}

func (s *RoleService) getPermissionResourceData(id int64) ([]*user.PermissionData, error) {
	var dbrows []*dbPermission
	var pdata []*user.PermissionData
	err := s.Dbh.Select("perm.*").From(`
			auth_role_permission
			JOIN auth_permission perm
			ON auth_role_permission.auth_permission_id = perm.auth_permission_id
		`).Where("auth_role_permission.auth_role_id = $1", id).QueryStructs(&dbrows)
	if err != nil {
		return pdata, err
	}
	return NewPermissionService(
		s.Dbh,
		"permissions",
	).dbToCollResourceData(dbrows), nil
}

func (s *RoleService) getRelatedUsersCount(id int64) (int64, error) {
	var count int64
	err := s.Dbh.Select("COUNT(*)").From(`
		auth_user_role
		JOIN auth_user
		ON auth_user_role.auth_user_id = auth_user.auth_user_id
		JOIN auth_user_info
		ON auth_user_info.auth_user_id = auth_user.auth_user_id
		`).Where("auth_user_role.auth_role_id = $1", id).QueryScalar(&count)
	return count, err
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
	`).Where("auth_user_role.auth_role_id = $1", id).
		QueryStructs(&dbrows)
	if err != nil {
		return udata, err
	}
	return NewUserService(
		s.Dbh,
		"users",
	).dbToCollResourceData(dbrows), nil
}

func (s *RoleService) getUserResourceDataWithPagination(id, pagenum, pagesize int64) ([]*user.UserData, error) {
	var dbrows []*dbUser
	var udata []*user.UserData
	err := s.Dbh.SQL(
		fmt.Sprintf(
			"%s LIMIT %d OFFSET %d",
			`SELECT auth_user.auth_user_id,
				CAST(auth_user.email AS TEXT),
				auth_user.first_name,
				auth_user.last_name,
				auth_user.is_active,
				auth_user_info.*
				FROM auth_user_role
				JOIN auth_user
				ON auth_user_role.auth_user_id = auth_user.auth_user_id
				JOIN auth_user_info
				ON auth_user_info.auth_user_id = auth_user.auth_user_id
				WHERE auth_user_role.auth_role_id = $1`,
			pagesize,
			(pagenum-1)*pagesize,
		), id).QueryStructs(&dbrows)
	if err != nil {
		return udata, err
	}
	return NewUserService(
		s.Dbh,
		"users",
	).dbToCollResourceData(dbrows), nil
}

func (s *RoleService) buildUserResourceIdentifiers(users []*user.UserData) []*jsonapi.Data {
	jdata := make([]*jsonapi.Data, len(users))
	for i, r := range users {
		jdata[i] = &jsonapi.Data{
			Type: r.Type,
			Id:   r.Id,
		}
	}
	return jdata
}

func (s *RoleService) buildPermissionResourceIdentifiers(perms []*user.PermissionData) []*jsonapi.Data {
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
			Self: s.GenResourceSelfLink(id),
		},
	}
}

func (s *RoleService) buildResource(id int64, attr *user.RoleAttributes) *user.Role {
	return &user.Role{
		Data: s.buildResourceData(id, attr),
		Links: &jsonapi.Links{
			Self: s.GenResourceSelfLink(id),
		},
	}
}

func (s *RoleService) buildResourceRelationships(id int64, role *user.Role) error {
	var allInc []*any.Any
	for _, inc := range s.Params.Includes {
		switch inc {
		case "users":
			users, err := s.getUserResourceData(id)
			if err != nil {
				return err
			}
			// included relationships
			incUsers, err := NewUserService(s.Dbh, "users").convertAllToAny(users)
			if err != nil {
				return err
			}
			allInc = append(allInc, incUsers...)
			role.Data.Relationships.Users.Data = s.buildUserResourceIdentifiers(users)
		case "permissions":
			perms, err := s.getPermissionResourceData(id)
			if err != nil {
				return err
			}
			incPerms, err := NewPermissionService(s.Dbh, "permissions").convertAllToAny(perms)
			if err != nil {
				return err
			}
			allInc = append(allInc, incPerms...)
			role.Data.Relationships.Permissions.Data = s.buildPermissionResourceIdentifiers(perms)
		}
		role.Included = allInc
	}
	return nil
}

func (s *RoleService) dbToResourceAttributes(r *dbRole) *user.RoleAttributes {
	return &user.RoleAttributes{
		Role:        r.Role,
		Description: r.Description,
		CreatedAt:   aphgrpc.NullToTime(r.CreatedAt),
		UpdatedAt:   aphgrpc.NullToTime(r.UpdatedAt),
	}
}

func (s *RoleService) dbToCollResourceData(dbrows []*dbRole) []*user.RoleData {
	var rdata []*user.RoleData
	for _, r := range dbrows {
		rdata = append(rdata, s.buildResourceData(r.AuthRoleId, s.dbToResourceAttributes(r)))
	}
	return rdata
}

func (s *RoleService) dbToCollResource(dbrows []*dbRole) *user.RoleCollection {
	return &user.RoleCollection{
		Data: s.dbToCollResourceData(dbrows),
		Links: &jsonapi.Links{
			Self: s.GenCollResourceSelfLink(),
		},
	}
}

func (s *RoleService) dbToCollResourceWithRel(dbrows []*dbRole) (*user.RoleCollection, error) {
	rdata := s.dbToCollResourceData(dbrows)
	var allInc []*any.Any
	// related users
	for _, inc := range s.Params.Includes {
		switch inc {
		case "users":
			var users []*user.UserData
			for i, _ := range rdata {
				u, err := s.getUserResourceData(dbrows[i].AuthRoleId)
				if err != nil {
					return &user.RoleCollection{}, err
				}
				rdata[i].Relationships.Users.Data = s.buildUserResourceIdentifiers(u)
				users = append(users, u...)
			}
			incUsers, err := NewUserService(s.Dbh, "users").convertAllToAny(users)
			if err != nil {
				return &user.RoleCollection{}, err
			}
			allInc = append(allInc, incUsers...)
		case "permissions":
			var perms []*user.PermissionData
			for i, _ := range rdata {
				p, err := s.getPermissionResourceData(dbrows[i].AuthRoleId)
				if err != nil {
					return &user.RoleCollection{}, err
				}
				rdata[i].Relationships.Permissions.Data = s.buildPermissionResourceIdentifiers(p)
				perms = append(perms, p...)
			}
			incPerms, err := NewPermissionService(s.Dbh, "permissions").convertAllToAny(perms)
			if err != nil {
				return &user.RoleCollection{}, err
			}
			allInc = append(allInc, incPerms...)
		}
	}
	return &user.RoleCollection{
		Data: rdata,
		Links: &jsonapi.Links{
			Self: s.GenCollResourceSelfLink(),
		},
		Included: allInc,
	}, nil
}

func (s *RoleService) attrTodbRole(attr *user.RoleAttributes) *dbRole {
	return &dbRole{
		Role:        attr.Role,
		Description: attr.Description,
		CreatedAt:   dat.NullTimeFrom(aphgrpc.ProtoTimeStamp(attr.CreatedAt)),
		UpdatedAt:   dat.NullTimeFrom(aphgrpc.ProtoTimeStamp(attr.UpdatedAt)),
	}
}

func (s *RoleService) convertAllToAny(roles []*user.RoleData) ([]*any.Any, error) {
	aslice := make([]*any.Any, len(roles))
	for i, r := range roles {
		pkg, err := ptypes.MarshalAny(r)
		if err != nil {
			return aslice, err
		}
		aslice[i] = pkg
	}
	return aslice, nil
}
