package server

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/dictyBase/apihelpers/aphgrpc"
	"github.com/dictyBase/go-genproto/dictybaseapis/api/jsonapi"
	"github.com/dictyBase/go-genproto/dictybaseapis/user"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/empty"
	dat "gopkg.in/mgutz/dat.v1"
	runner "gopkg.in/mgutz/dat.v1/sqlx-runner"
)

const (
	roleDbTable    = "auth_role"
	roleDbTblAlias = "auth_role role"
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

func (s *RoleService) GetRole(ctx context.Context, r *jsonapi.GetRequest) (*user.Role, error) {
	params, md, err := aphgrpc.ValidateAndParseGetParams(s, r)
	if err != nil {
		grpc.SetTrailer(ctx, md)
		return &user.Role{}, status.Error(codes.InvalidArgument, err.Error())
	}
	s.params = params
	s.listMethod = false
	switch {
	case params.HasFields && params.HasInclude:
		s.includeStr = r.Include
		s.fieldsStr = r.Fields
		role, err := s.getResourceWithSelectedAttr(params, r.Id)
		if err != nil {
			return &user.Role{}, aphgrpc.handleError(ctx, err)
		}
		err := s.buildResourceRelationships(id, role)
		if err != nil {
			return &user.Role{}, aphgrpc.handleError(ctx, err)
		}
		return role, nil
	case params.HasFields:
		s.fieldsStr = r.Fields
		role, err := s.getResourceWithSelectedAttr(params, r.Id)
		if err != nil {
			return &user.Role{}, aphgrpc.handleError(ctx, err)
		}
		return role, nil
	case params.HasInclude:
		s.includeStr = r.Include
		role, err := s.getResource(r.Id)
		if err != nil {
			return &user.Role{}, aphgrpc.handleError(ctx, err)
		}
		err := s.buildResourceRelationships(id, role)
		if err != nil {
			return &user.Role{}, aphgrpc.handleError(ctx, err)
		}
		return role, nil
	default:
		role, err := s.getResource(r.Id)
		if err != nil {
			return &user.Role{}, aphgrpc.handleError(ctx, err)
		}
		return role, nil
	}
}

func (s *RoleService) GetRelatedUsers(ctx context.Context, r *jsonapi.RelationshipRequest) (*UserCollection, error) {
	udata, err := s.getUserResourceData(id)
	if err != nil {
		return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
	}
	return &user.UserCollection{
		Data: udata,
		Links: &jsonapi.PaginationLinks{
			Self: NewUserService(
				s.Dbh,
				"users",
				s.GetBaseURL(),
			).genCollResourceSelfLink(),
		},
	}, nil
}

func (s *RoleService) GetRelatedPermissions(ctx context.Context, r *jsonapi.RelationshipRequest) (*PermissionCollection, error) {
	pdata, err := s.getPermissionResourceData(r.Id)
	if err != nil {
		return &user.PermissionCollection{}, aphgrpc.handleError(ctx, err)
	}
	return &user.PermissionCollection{
		Data: pdata,
		Links: &jsonapi.PaginationLinks{
			Self: NewPermissionService(
				s.Dbh,
				"permissions",
				s.GetBaseURL(),
			).genCollResourceSelfLink(),
		},
	}, nil
}

func (s *RoleService) ListRoles(ctx context.Context, r *jsonapi.ListRequest) (*user.RoleCollection, error) {
	params, md, err := aphgrpc.ValidateAndParseListParams(s, r)
	if err != nil {
		grpc.SetTrailer(ctx, md)
		return &user.RoleCollection{}, status.Error(codes.InvalidArgument, err.Error())
	}
	s.params = params
	s.listMethod = true
	// has pagination query parameters
	if aphgrpc.HasPagination {
		switch {
		// filter, fields and include parameters
		case params.HasFields && params.HasInclude && params.HasFilter:
			s.fieldsStr = r.Fields
			s.filterStr = r.Filter
			s.includeStr = r.Include
			count, err := s.getAllFilteredCount(roleDbTable)
			if err != nil {
				return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
			}
			dbRoles, err := s.getAllSelectedFilteredRowsWithPaging(r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.dbToCollResourceWithRelAndPagination(count, dbRoles, r.Pagenum, r.Pagesize)
		// fields and includes
		case params.HasFields && params.HasInclude:
			s.fieldsStr = r.Fields
			s.includeStr = r.Include
			count, err := s.getCount(userDbTable)
			if err != nil {
				return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
			}
			dbRoles, err := s.getAllSelectedRowsWithPaging(r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.dbToCollResourceWithRelAndPagination(count, dbRoles, r.Pagenum, r.Pagesize)
		// fields and filters
		case params.HasFields && params.HasFilter:
			s.fieldsStr = r.Fields
			s.filterStr = r.Filter
			count, err := s.getAllFilteredCount(roleDbTable)
			if err != nil {
				return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
			}
			dbRoles, err := s.getAllSelectedFilteredRowsWithPaging(params, r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.dbToCollResourceWithPagination(count, dbRoles, r.Pagenum, r.Pagesize)
		// include and filter
		case params.HasInclude && params.HasFilter:
			s.includeStr = r.Include
			s.filterStr = r.Filter
			count, err := s.getAllFilteredCount(roleDbTable)
			if err != nil {
				return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
			}
			dbRoles, err := s.getAllFilteredRowsWithPaging(r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.dbToCollResourceWithRelAndPagination(count, dbRoles, r.Pagenum, r.Pagesize)
		case params.HasFields:
			s.fieldsStr = r.Fields
			count, err := s.getCount(roleDbTable)
			if err != nil {
				return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
			}
			dbRoles, err := s.getAllSelectedRowsWithPaging(r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.dbToCollResourceWithPagination(count, dbRoles, r.Pagenum, r.Pagesize)
		case params.HasFilter:
			s.filterStr = r.Filter
			count, err := s.getAllFilteredCount(roleDbTable)
			if err != nil {
				return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
			}
			dbRoles, err := s.getAllFilteredRowsWithPaging(r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.dbToCollResourceWithPagination(count, dbRoles, r.Pagenum, r.Pagesize)
		case params.HasInclude:
			s.includeStr = r.Include
			count, err := s.getCount(userDbTable)
			if err != nil {
				return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
			}
			dbRoles, err := s.getAllRowsWithPaging(r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.dbToCollResourceWithRelAndPagination(count, dbRoles, r.Pagenum, r.Pagesize)
		// only pagination
		default:
			count, err := s.getCount(userDbTable)
			if err != nil {
				return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
			}
			dbRoles, err := s.getAllSelectedRowsWithPaging(r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.dbToCollResourceWithPagination(count, dbRoles, r.Pagenum, r.Pagesize), nil
		}
	}
	// request without any pagination query parameters
	switch {
	case params.HasFields && params.HasFilter && params.HasInclude:
		s.fieldsStr = r.Fields
		s.filterStr = r.Filter
		s.includeStr = r.Include
		count, err := s.getAllFilteredCount(roleDbTable)
		if err != nil {
			return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			dbRoles, err := s.getAllSelectedFilteredRowsWithPaging(params, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
			if err != nil {
				return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.dbToCollResourceWithRelAndPagination(count, dbRoles, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		}
		return s.dbToCollResource(dbRoles), nil
	case params.HasFields && params.HasFilter:
		s.fieldsStr = r.Fields
		s.filterStr = r.Filter
		count, err := s.getAllFilteredCount(roleDbTable)
		if err != nil {
			return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			dbRoles, err := s.getAllSelectedFilteredRowsWithPaging(params, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
			if err != nil {
				return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.dbToCollResourceWithPagination(count, dbRoles, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		}
		return s.dbToCollResource(dbRoles), nil
	case params.HasFields && params.HasInclude:
		s.fieldsStr = r.Fields
		s.includeStr = r.Include
		count, err := s.getCount(roleDbTable)
		if err != nil {
			return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			dbRoles, err := s.getAllSelectedRowsWithPaging(aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
			if err != nil {
				return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.dbToCollResourceWithRelAndPagination(count, dbRoles, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		}
		return s.dbToCollResource(dbRoles), nil
	case params.HasFilter && params.HasInclude:
		s.includeStr = r.Include
		s.filterStr = r.Filter
		count, err := s.getAllFilteredCount(roleDbTable)
		if err != nil {
			return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			dbRoles, err := s.getAllFilteredRowsWithPaging(aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
			if err != nil {
				return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.dbToCollResourceWithRelAndPagination(count, dbRoles, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		}
		return s.dbToCollResource(dbRoles), nil
	case params.HasFields:
		s.fieldsStr = r.Fields
		count, err := s.getCount(roleDbTable)
		if err != nil {
			return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			dbRoles, err := s.getAllSelectedRowsWithPaging(aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
			if err != nil {
				return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.dbToCollResourceWithPagination(count, dbRoles, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		}
		return s.dbToCollResource(dbRoles), nil
	case params.HasFilter:
		s.filterStr = r.Filter
		count, err := s.getAllFilteredCount(roleDbTable)
		if err != nil {
			return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			dbRoles, err := s.getAllFilteredRowsWithPaging(aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
			if err != nil {
				return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.dbToCollResourceWithPagination(count, dbRoles, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		}
		return s.dbToCollResource(dbRoles), nil
	case params.HasInclude:
		s.includeStr = r.Include
		count, err := s.getCount(roleDbTable)
		if err != nil {
			return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			dbRoles, err := s.getAllRowsWithPaging(aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
			if err != nil {
				return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.dbToCollResourceWithRelAndPagination(count, dbRoles, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		}
		return s.dbToCollResource(dbRoles), nil
	default:
		count, err := s.getCount(roleDbTable)
		if err != nil {
			return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			dbRoles, err := s.getAllPaginatedRows(aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
			if err != nil {
				return &user.RoleCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.dbToCollResourceWithPagination(count, dbRoles, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		}
		return s.dbToCollResource(dbRoles), nil
	}
}

func (s *RoleService) CreateRole(ctx context.Context, r *user.CreateRoleRequest) (*user.Role, error) {
	var roleId int64
	dbrole := s.attrTodbRole(r.Data.Attributes)
	rcolumns := aphgrpc.GetDefinedTags(dbrole, "db")
	if len(rcolumns) > 0 {
		_, err := s.Dbh.InsertInto("auth_role").
			Columns(rcolumns...).
			Record(dbrole).
			Returning("auth_role_id").QueryScalar(&roleId)
		if err != nil {
			grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseInsert)
			return &user.User{}, status.Error(codes.Internal, err.Error())
		}
	}
	for _, u := range r.Data.Relationships.Users.Data {
		_, err = s.Dbh.InsertInto("auth_user_role").
			Columns("auth_user_id", "auth_role_id").
			Values(u.Id, roleId).Exec()
		if err != nil {
			grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseInsert)
			return &user.User{}, status.Error(codes.Internal, err.Error())
		}
	}
	for _, p := range r.Data.Relationships.Permissions.Data {
		_, err := s.Dbh.InsertInto("auth_role_permission").
			Columns("auth_role_id", "auth_permission_id").
			Values(roleId, p.Id).Exec()
		if err != nil {
			grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseInsert)
			return &user.User{}, status.Error(codes.Internal, err.Error())
		}
	}
	grpc.SetTrailer(ctx, metadata.Pairs("method", "POST"))
	return s.buildResource(roleId, r.Data.Attributes), nil
}

func (s *RoleService) CreateUserRelationship(ctx context.Context, r *jsonapi.DataCollection) (*empty.Empty, error) {
	if err := s.existsResource(r.Id); err != nil {
		return &empty.Empty{}, aphgrpc.handleError(ctx, err)
	}
	for _, ud := range r.Data {
		err := s.Dbh.Select("aurole.auth_user_role_id").
			From("auth_user_role aurole").
			Where("aurole.auth_role_id = $1 AND aurole.auth_user_role_id = $2", r.Id, ud.Id).
			Exec()
		if err != nil {
			if err == dat.ErrNotFound {
				err := s.Dbh.InsertInto("auth_user_role").
					Columns("auth_role_id", "auth_user_id").
					Values(r.Id, ud.Id).Exec()
				if err != nil {
					grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseInsert)
					return &empty.Empty{}, status.Error(codes.Internal, err.Error())

				}
			}
		}
	}
	grpc.SetTrailer(ctx, metadata.Pairs("method", "POST_NO_CONTENT"))
	return &empty.Empty{}, nil
}

func (s *RoleService) CreatePermissionRelationship(ctx context.Context, r *jsonapi.DataCollection) (*empty.Empty, error) {
	if err := s.existsResource(r.Id); err != nil {
		return &empty.Empty{}, aphgrpc.handleError(ctx, err)
	}
	for _, pd := range r.Data {
		err := s.Dbh.Select("auth_role_permission.auth_role_permission_id").
			From("auth_role_permission").
			Where("auth_role_permission.auth_role_id = $1 AND auth_role_permission.auth_permission_id = $2", r.Id, pd.Id).
			Exec()
		if err != nil {
			if err == dat.ErrNotFound {
				err := s.Dbh.InsertInto("auth_role_permission").
					Columns("auth_role_id", "auth_permission_id").
					Values(r.Id, pd.Id).Exec()
				if err != nil {
					grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseInsert)
					return &empty.Empty{}, status.Error(codes.Internal, err.Error())

				}
			}
		}
	}
	grpc.SetTrailer(ctx, metadata.Pairs("method", "POST_NO_CONTENT"))
	return &empty.Empty{}, nil
}

func (s *RoleService) UpdateRole(ctx context.Context, r *user.UpdateRoleRequest) (*user.Role, error) {
	if err := s.existsResource(r.Data.Id); err != nil {
		return &user.Role{}, aphgrpc.handleError(ctx, err)
	}
	dbrole := s.attrTodbRole(r.Data.Attributes)
	rmap := aphgrpc.GetDefinedTagsWithValue(dbrole, "db")
	if len(rmap) > 0 {
		_, err := s.Dbh.Update(roleDbTable).SetMap(rmap).
			Where("auth_role_id = $1", r.Data.Id).Exec()
		if err != nil {
			grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseUpdate)
			return &user.Role{}, status.Error(codes.Internal, err.Error())
		}
	}
	for _, u := range r.Data.Relationships.Users.Data {
		_, err = s.Dbh.Update("auth_user_role").
			Set("auth_user_id", u.Id).
			Where("auth_role_id = $1", r.Data.Id).Exec()
		if err != nil {
			grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseUpdate)
			return &user.Role{}, status.Error(codes.Internal, err.Error())
		}
	}
	return s.buildResource(r.Data.Id, r.Data.Attributes), nil
}

func (s *RoleService) UpdateUserRelationship(ctx context.Context, r *jsonapi.DataCollection) (*empty.Empty, error) {
	if err := s.existsResource(r.Id); err != nil {
		return &empty.Empty{}, aphgrpc.handleError(ctx, err)
	}
	_, err := s.Dbh.DeleteFrom("auth_user_role").
		Where("auth_user_role.auth_role_id = $1", r.Id).
		Exec()
	if err != nil {
		grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseUpdate)
		return &empty.Empty{}, status.Error(codes.Internal, err.Error())
	}
	for _, ud := range r.Data {
		err := s.Dbh.InsertInto("auth_user_role").
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
	if err := s.existsResource(r.Id); err != nil {
		return &empty.Empty{}, aphgrpc.handleError(ctx, err)
	}
	_, err := s.Dbh.DeleteFrom("auth_role_permission").
		Where("auth_role_permission.auth_role_id = $1", r.Id).
		Exec()
	if err != nil {
		grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseUpdate)
		return &empty.Empty{}, status.Error(codes.Internal, err.Error())
	}
	for _, pd := range r.Data {
		err := s.Dbh.InsertInto("auth_role_permission").
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
	if err := s.existsResource(r.Data.Id); err != nil {
		return &empty.Empty{}, aphgrpc.handleError(ctx, err)
	}
	_, err := s.Dbh.DeleteFrom(roleDbTable).Where("auth_role_id = $1", r.Id).Exec()
	if err != nil {
		grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseDelete)
		return &empty.Empty{}, status.Error(codes.Internal, err.Error())
	}
	return &empty.Empty{}, nil
}

func (s *RoleService) DeleteUserRelationship(ctx context.Context, r *jsonapi.DataCollection) (*empty.Empty, error) {
	if err := s.existsResource(r.Id); err != nil {
		return &empty.Empty{}, aphgrpc.handleError(ctx, err)
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
	if err := s.existsResource(r.Id); err != nil {
		return &empty.Empty{}, aphgrpc.handleError(ctx, err)
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

func (s *RoleService) existsResource(id int64) error {
	return s.Dbh.Select("auth_role_id").From("auth_role").
		Where("auth_role_id = $1", id).Exec()
}

func (s *RoleService) getResourceWithSelectedAttr(id int64) (*user.Role, error) {
	drole := &dbRole{}
	columns := s.fieldsToColumns(s.params.Fields)
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
		QueryStructs(dbrows)
	return dbrows, err
}

func (s *RoleService) getAllRowsWithPaging(pagenum int64, pagesize int64) ([]*dbRole, error) {
	var dbrows []*dbRole
	err := s.Dbh.Select("role.*").
		From(roleDbTblAlias).
		Paginate(uint64(pagenum), uint64(pagesize)).
		QueryStructs(dbrows)
	return dbrows, err
}

func (s *RoleService) getAllSelectedRowsWithPaging(pagenum, pagesize int64) ([]*dbRole, error) {
	var dbrows []*dbRole
	columns := s.MapFieldsToColumns(s.params.Fields)
	err := s.Dbh.Select(columns...).
		From(roleDbTblAlias).
		Paginate(uint64(pageNum), uint64(pageSize)).
		QueryStructs(dbrows)
	return dbrows, err
}

func (s *RoleService) getAllFilteredRowsWithPaging(pagenum, pagesize int64) ([]*dbRole, error) {
	var dbrows []*dbRole
	err := s.Dbh.Select("role.*").
		From(roleDbTblAlias).
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
		From(roleDbTblAlias).
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

func (s *RoleService) buildUserResourceIdentifiers(users []*user.Data) []*jsonapi.Data {
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
			Self: s.genSingularResSelfLink(id),
		},
	}
}

func (s *RoleService) buildResource(id int64, attr *user.RoleAttributes) *user.Role {
	return &user.Role{
		Data: s.buildResourceData(id, attr),
	}
}

func (s *RoleService) buildResourceRelationships(id int64, role *user.Role) error {
	var allInc []*any.Any
	for _, inc := range s.parse.Includes {
		switch inc {
		case "users":
			users, err := s.getUserResourceData(id)
			if err != nil {
				return err
			}
			// included relationships
			incUsers, err := aphgrpc.ConvertAllToAny(users)
			if err != nil {
				return err
			}
			allInc = append(allInc, incUsers...)
			role.Relationships.Users.Data = s.buildUserResourceIdentifiers(users)
		case "permissions":
			perms, err := s.getPermissionResourceData(id)
			if err != nil {
				return err
			}
			incPerms, err := aphgrpc.ConvertAllToAny(perms)
			if err != nil {
				return err
			}
			allInc = append(allInc, incPerms...)
			role.Relationships.Permissions.Data = s.buildPermissionResourceIdentifiers(perms)
		}
		role.Included = allInc
	}
	return nil
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
