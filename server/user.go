package server

import (
	"github.com/dictyBase/apihelpers/aphgrpc"
	"github.com/dictyBase/go-genproto/dictybaseapis/api/jsonapi"
	"github.com/dictyBase/go-genproto/dictybaseapis/user"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	dat "gopkg.in/mgutz/dat.v1"
	"gopkg.in/mgutz/dat.v1/sqlx-runner"

	context "golang.org/x/net/context"
)

const (
	usrTablesJoin = `
			auth_user user
			JOIN auth_user_info uinfo
			ON user.auth_user_id = uinfo.auth_user_id
	`
	userDbTable = "auth_user user"
)

var coreUserCols = []string{"first_name", "last_name", "email"}
var userInfoCols = []string{
	"auth_user_id",
	"organization",
	"group_name",
	"first_address",
	"second_address",
	"city",
	"state",
	"zipcode",
	"country",
	"phone",
	"created_at",
	"updated_at",
}

type dbUser struct {
	AuthUserId    int64          `db:"auth_user_id"`
	FirstName     string         `db:"first_name"`
	LastName      string         `db:"last_name"`
	Email         string         `db:"email"`
	IsActive      bool           `db:"is_active"`
	Organization  dat.NullString `db:"organization"`
	GroupName     dat.NullString `db:"group_name"`
	FirstAddress  dat.NullString `db:"first_address"`
	SecondAddress dat.NullString `db:"second_address"`
	City          dat.NullString `db:"city"`
	State         dat.NullString `db:"state"`
	Zipcode       dat.NullString `db:"zipcode"`
	Country       dat.NullString `db:"country"`
	Phone         dat.NullString `db:"phone"`
	CreatedAt     dat.NullTime   `db:"created_at"`
	UpdatedAt     dat.NullTime   `db:"updated_at"`
}

type dbCoreUser struct {
	FirstName string `db:"first_name"`
	LastName  string `db:"last_name"`
	Email     string `db:"email"`
	IsActive  bool   `db:"is_active"`
}

type dbUserInfo struct {
	AuthUserId    int64          `db:"auth_user_id"`
	Organization  dat.NullString `db:"organization"`
	GroupName     dat.NullString `db:"group_name"`
	FirstAddress  dat.NullString `db:"first_address"`
	SecondAddress dat.NullString `db:"second_address"`
	City          dat.NullString `db:"city"`
	State         dat.NullString `db:"state"`
	Zipcode       dat.NullString `db:"zipcode"`
	Country       dat.NullString `db:"country"`
	Phone         dat.NullString `db:"phone"`
	CreatedAt     dat.NullTime   `db:"created_at"`
	UpdatedAt     dat.NullTime   `db:"updated_at"`
}

type UserService struct {
	*aphgrpc.Service
}

func NewUserService(dbh *runner.DB, pathPrefix string, baseURL string) *UserService {
	return &UserService{
		&aphgrpc.Service{
			baseURL:    baseURL,
			resource:   "users",
			Dbh:        dbh,
			pathPrefix: pathPrefix,
			include:    []string{"roles"},
			filterToColumns: map[string]string{
				"first_name": "user.first_name",
				"last_name":  "user.last_name",
				"email":      "user.email",
			},
			fieldsToColumns: map[string]string{
				"first_name":     "user.first_name",
				"last_name":      "user.last_name",
				"email":          "user.email",
				"created_at":     "user.created_at",
				"updated_at":     "user.updated_at",
				"organization":   "uinfo.organization",
				"group_name":     "uinfo.group_name",
				"first_address":  "uinfo.first_address",
				"second_address": "uinfo.second_address",
				"city":           "uinfo.city",
				"state":          "uinfo.state",
				"zipcode":        "uinfo.zipcode",
				"country":        "uinfo.country",
				"phone":          "uinfo.phone",
				"is_active":      "uinfo.is_active",
			},
			requiredAttrs: []string{"FirstName", "LastName", "Email"},
		},
	}
}

func (s *UserService) GetUser(ctx context.Context, r *jsonapi.GetRequest) (*user.User, error) {
	params, md, err := aphgrpc.ValidateAndParseGetParams(s, r)
	if err != nil {
		grpc.SetTrailer(ctx, md)
		return new(user.User), status.Error(codes.InvalidArgument, err.Error())
	}
	s.params = params
	s.listMethod = false
	switch {
	case params.HasFields && params.HasInclude:
		s.includeStr = r.Include
		s.fieldsStr = r.Fields
		u, err := s.getResourceWithSelectedAttr(params, r.Id)
		if err != nil {
			return &user.User{}, aphgrpc.handleError(ctx, err)
		}
		roles, err := s.getRoleResourceData(id)
		if err != nil {
			return &user.User{}, aphgrpc.handleError(ctx, err)
		}
		// included relationships
		incRoles, err := aphgrpc.ConvertAllToAny(roles)
		if err != nil {
			return &user.User{}, aphgrpc.handleError(ctx, err)
		}
		u.Included = incRoles
		u.Relationships.Roles.Data = s.buildRoleResourceIdentifiers(roles)
		return u, nil
	case params.HasFields:
		s.fieldsStr = r.Fields
		u, err := s.getUserWithSelectedAttr(params, r.Id)
		if err != nil {
			return &user.User{}, aphgrpc.handleError(ctx, err)
		}
		return u, nil
	case params.HasInclude:
		s.includeStr = r.Include
		u, err := s.getResource(r.Id)
		if err != nil {
			return &user.User{}, aphgrpc.handleError(ctx, err)
		}
		roles, err := s.getRoleResourceData(id)
		if err != nil {
			return &user.User{}, aphgrpc.handleError(ctx, err)
		}
		// included relationships
		incRoles, err := aphgrpc.ConvertAllToAny(roles)
		if err != nil {
			return &user.User{}, aphgrpc.handleError(ctx, err)
		}
		u.Included = incRoles
		u.Relationships.Roles.Data = s.buildRoleResourceIdentifiers(roles)
		return u, nil
	default:
		u, err := s.getResource(r.Id)
		if err != nil {
			return &user.User{}, aphgrpc.handleError(ctx, err)
		}
		return u, nil
	}
}

func (s *UserService) ListUsers(ctx context.Context, r *jsonapi.ListRequest) (*user.UserCollection, error) {
	params, md, err := aphgrpc.ValidateAndParseListParams(s, r)
	if err != nil {
		grpc.SetTrailer(ctx, md)
		return new(user.User), status.Error(codes.InvalidArgument, err.Error())
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
			count, err := s.getAllFilteredCount(usrTablesJoin)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			dbUsers, err := s.getAllSelectedFilteredRowsWithPaging(r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.dbToCollResourceWithRelAndPagination(count, dbUsers, r.Pagenum, r.Pagesize)
		// fields and includes
		case params.HasFields && params.HasInclude:
			s.fieldsStr = r.Fields
			s.includeStr = r.Include
			count, err := s.getCount(userDbTable)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			dbUsers, err := s.getAllSelectedRowsWithPaging(r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.dbToCollResourceWithRelAndPagination(count, dbUsers, r.Pagenum, r.Pagesize)
		// fields and filters
		case params.HasFields && params.HasFilter:
			s.fieldsStr = r.Fields
			s.filterStr = r.Filter
			count, err := s.getAllFilteredCount(usrTablesJoin)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			dbUsers, err := s.getAllSelectedFilteredRowsWithPaging(params, r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.dbToCollResourceWithPagination(count, dbUsers, r.Pagenum, r.Pagesize)
		// include and filter
		case params.HasInclude && params.HasFilter:
			s.includeStr = r.Include
			s.filterStr = r.Filter
			count, err := s.getAllFilteredCount(usrTablesJoin)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			dbUsers, err := s.getAllFilteredRowsWithPaging(r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.dbToCollResourceWithRelAndPagination(count, dbUsers, r.Pagenum, r.Pagesize)
		case params.HasFields:
			s.fieldsStr = r.Fields
			count, err := s.getCount(userDbTable)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			dbUsers, err := s.getAllSelectedRowsWithPaging(r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.dbToCollResourceWithPagination(count, dbUsers, r.Pagenum, r.Pagesize)
		case params.HasFilter:
			s.filterStr = r.Filter
			count, err := s.getAllFilteredCount(usrTablesJoin)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			dbUsers, err := s.getAllFilteredRowsWithPaging(r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.dbToCollResourceWithPagination(count, dbUsers, r.Pagenum, r.Pagesize)
		case params.HasInclude:
			s.includeStr = r.Include
			count, err := s.getCount(userDbTable)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			dbUsers, err := s.getAllRowsWithPaging(r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.dbToCollResourceWithRelAndPagination(count, dbUsers, r.Pagenum, r.Pagesize)
		// only pagination
		default:
			count, err := s.getCount(userDbTable)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			dbUsers, err := s.getAllSelectedRowsWithPaging(r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.dbToCollResourceWithPagination(count, dbUsers, r.Pagenum, r.Pagesize), nil
		}
	}
	// request without any pagination query parameters
	switch {
	case params.HasFields && params.HasFilter && params.HasInclude:
		s.fieldsStr = r.Fields
		s.filterStr = r.Filter
		s.includeStr = r.Include
		count, err := s.getAllFilteredCount(usrTablesJoin)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			dbUsers, err := s.getAllSelectedFilteredRowsWithPaging(params, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.dbToCollResourceWithRelAndPagination(count, dbUsers, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		}
		return s.dbToCollResource(dbUsers), nil
	case params.HasFields && params.HasFilter:
		s.fieldsStr = r.Fields
		s.filterStr = r.Filter
		count, err := s.getAllFilteredCount(usrTablesJoin)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			dbUsers, err := s.getAllSelectedFilteredRowsWithPaging(params, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.dbToCollResourceWithPagination(count, dbUsers, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		}
		return s.dbToCollResource(dbUsers), nil
	case params.HasFields && params.HasInclude:
		s.fieldsStr = r.Fields
		s.includeStr = r.Include
		count, err := s.getCount("auth_user")
		if err != nil {
			return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			dbUsers, err := s.getAllSelectedRowsWithPaging(aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.dbToCollResourceWithRelAndPagination(count, dbUsers, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		}
		return s.dbToCollResource(dbUsers), nil
	case params.HasFilter && params.HasInclude:
		s.includeStr = r.Include
		s.filterStr = r.Filter
		count, err := s.getAllFilteredCount(usrTablesJoin)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			dbUsers, err := s.getAllFilteredRowsWithPaging(aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.dbToCollResourceWithRelAndPagination(count, dbUsers, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		}
		return s.dbToCollResource(dbUsers), nil
	case params.HasFields:
		s.fieldsStr = r.Fields
		count, err := s.getCount(userDbTable)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			dbUsers, err := s.getAllSelectedRowsWithPaging(aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.dbToCollResourceWithPagination(count, dbUsers, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		}
		return s.dbToCollResource(dbUsers), nil
	case params.HasFilter:
		s.filterStr = r.Filter
		count, err := s.getAllFilteredCount(usrTablesJoin)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			dbUsers, err := s.getAllFilteredRowsWithPaging(aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.dbToCollResourceWithPagination(count, dbUsers, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		}
		return s.dbToCollResource(dbUsers), nil
	case params.HasInclude:
		s.includeStr = r.Include
		count, err := s.getCount(userDbTable)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			dbUsers, err := s.getAllRowsWithPaging(aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.dbToCollResourceWithRelAndPagination(count, dbUsers, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		}
		return s.dbToCollResource(dbUsers), nil
	default:
		count, err := s.getCount("auth_user")
		if err != nil {
			return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			dbUsers, err := s.getAllPaginatedRows(aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.dbToCollResourceWithPagination(count, dbUsers, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		}
		return s.dbToCollResource(dbUsers), nil
	}
}

func (s *UserService) CreateUser(ctx context.Context, r *user.CreateUserRequest) (*user.User, error) {
	var userId int64
	dbcuser := s.attrTodbCoreUser(r.Data.Attributes)
	_, err := s.Dbh.InsertInto("auth_user").
		Columns(coreUserCols...).
		Record(dbcuser).
		Returning("auth_user_id").QueryScalar(&userId)
	if err != nil {
		grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseInsert)
		return &user.User{}, status.Error(codes.Internal, err.Error())
	}
	dbusrInfo := s.attrTodbUserInfo(r.Data.Attributes)
	usrInfoCols := aphgrpc.GetDefinedTags(dbusrInfo, "db")
	dbusrInfo.AuthUserId = userId
	if len(usrInfoCols) > 0 {
		_, err = s.Dbh.InsertInto("auth_user_info").
			Columns(userInfoCols...).
			Record(dbusrInfo).Exec()
		if err != nil {
			grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseInsert)
			return &user.User{}, status.Error(codes.Internal, err.Error())
		}
	}
	for _, role := range r.Data.Relationships.Roles.Data {
		_, err = s.Dbh.InsertInto("auth_user_role").
			Columns("auth_user_id", "auth_role_id").
			Values(userId, role.Id).Exec()
		if err != nil {
			grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseInsert)
			return &user.User{}, status.Error(codes.Internal, err.Error())
		}
	}
	grpc.SetTrailer(ctx, metadata.Pairs("method", "POST"))
	return getSingleUserData(userId, r.Data.Attributes), nil
}

func (s *UserService) UpdateUser(ctx context.Context, r *user.UpdateUserRequest) (*user.User, error) {
	if err := s.existsResource(r.Data.Id); err != nil {
		return &user.User{}, aphgrpc.handleError(ctx, err)
	}
	dbcuser := s.attrTodbCoreUser(r.Data.Attributes)
	usrMap := aphgrpc.GetDefinedTagsWithValue(dbcuser, "db")
	if len(usrMap) > 0 {
		_, err := s.Dbh.Update("auth_user").SetMap(usrMap).
			Where("auth_user_id = $1", r.Data.Id).Exec()
		if err != nil {
			grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseUpdate)
			return &user.User{}, status.Error(codes.Internal, err.Error())
		}
	}
	dbusrInfo := s.attrTodbUserInfo(r.Data.Attributes)
	usrInfoMap := aphgrpc.GetDefinedTagsWithValue(dbusrInfo, "db")
	if len(usrInfoMap) > 0 {
		_, err := s.Dbh.Update("auth_user_info").SetMap(usrInfoMap).
			Where("auth_user_id = $1", r.Data.Id).Exec()
		if err != nil {
			grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseUpdate)
			return &user.User{}, status.Error(codes.Internal, err.Error())
		}
	}
	for _, role := range r.Data.Relationships.Roles.Data {
		_, err = s.Dbh.Update("auth_user_role").
			Set("auth_role_id", role.Id).
			Where("auth_user_id = $1", r.Data.Id).Exec()
		if err != nil {
			grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseUpdate)
			return &user.User{}, status.Error(codes.Internal, err.Error())
		}
	}
	return s.buildResource(r.Data.Id, r.Data.Attributes), nil
}

func (s *UserService) DeleteUser(ctx context.Context, r *jsonapi.DeleteRequest) (*empty.Empty, error) {
	if err := s.existsResource(r.Data.Id); err != nil {
		return &empty.Empty{}, aphgrpc.handleError(ctx, err)
	}
	_, err := s.Dbh.DeleteFrom(userDbTable).Where("user.auth_user_id = $1", r.Id).Exec()
	if err != nil {
		grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseDelete)
		return &empty.Empty{}, status.Error(codes.Internal, err.Error())
	}
	return &empty.Empty{}, nil
}

// All helper functions

func (s *UserService) existsResource(id int64) error {
	return s.Dbh.Select("user.auth_user_id").From(userDbTable).
		Where("user.auth_user_id = $1", id).Exec()
}

// -- Functions that queries the storage and generates an user resource object

func (s *UserService) getResourceWithSelectedAttr(id int64) (*user.User, error) {
	dusr := new(dbUser)
	columns := s.fieldsToColumns(s.params.Fields)
	err := s.Dbh.Select(columns...).From(usrTablesJoin).
		Where("user.auth_user_id = $1", id).QueryStruct(dusr)
	if err != nil {
		return &user.User{}, err
	}
	return s.buildResource(id, s.dbToResourceAttributes(dusr)), nil
}

func (s *UserService) getResource(id int64) (*user.User, error) {
	dusr := new(dbUser)
	err := s.Dbh.Select("user.*", "uinfo.*").
		From(usrTablesJoin).
		Where("user.auth_user_id = $1", id).
		QueryStruct(dusr)
	if err != nil {
		return &user.User{}, err
	}
	return s.buildResource(id, s.dbToResourceAttributes(dusr)), nil
}

// -- Functions that queries the storage and generates a database user object

func (s *UserService) getAllRows() ([]*dbUser, error) {
	var dusrRows []*dbUser
	err := s.Dbh.Select("user.*", "uinfo.*").
		From(usrTablesJoin).
		QueryStructs(dusrRows)
	return dusrRows, err
}

func (s *UserService) getAllRowsWithPaging(pagenum int64, pagesize int64) ([]*dbUser, error) {
	var dusrRows []*dbUser
	err := s.Dbh.Select("user.*", "uinfo.*").
		From(usrTablesJoin).
		Paginate(uint64(pagenum), uint64(pagesize)).
		QueryStructs(dusrRows)
	return dusrRows, err
}

func (s *UserService) getAllSelectedRowsWithPaging(pagenum, pagesize int64) ([]*dbUser, error) {
	var dusrRows []*dbUser
	columns := s.MapFieldsToColumns(s.params.Fields)
	err := s.Dbh.Select(columns...).
		From(usrTablesJoin).
		Paginate(uint64(pageNum), uint64(pageSize)).
		QueryStructs(dusrRows)
	return dusrRows, err
}

func (s *UserService) getAllFilteredRowsWithPaging(pagenum, pagesize int64) ([]*dbUser, error) {
	var dusrRows []*dbUser
	err := s.Dbh.Select("user.*", "uinfo.*").
		From(usrTablesJoin).
		Scope(
			aphgrpc.FilterToWhereClause(s, s.params.Filter),
			aphgrpc.FilterToBindValue(s.params.Filter)...,
		).
		Paginate(uint64(pageNum), uint64(pageSize)).
		QueryStructs(dusrRows)
	return dusrRows, err
}

func (s *UserService) getAllSelectedFilteredRowsWithPaging(pagenum, pagesize int64) ([]*dbUser, error) {
	var dusrRows []*dbUser
	columns := s.MapFieldsToColumns(s.params.Fields)
	err := s.Dbh.Select(columns...).
		From(usrTablesJoin).
		Scope(
			aphgrpc.FilterToWhereClause(s, s.params.Filter),
			aphgrpc.FilterToBindValue(s.params.Filter)...,
		).
		Paginate(uint64(pageNum), uint64(pageSize)).
		QueryStructs(dusrRows)
	return dusrRows, err
}

// -- Functions that returns relationship resource objects

func (s *UserService) getRoleResourceData(id int64) ([]*user.RoleData, error) {
	var drole []*dbRole
	var rdata []*user.RoleData
	err := s.Dbh.Select("role.*").From(`
			auth_user_role
			JOIN auth_role role
			ON auth_user_role.auth_role_id = role.auth_role_id
		`).Where("auth_user_role.auth_user_id = $1", id).QueryStructs(drole)
	if err != nil {
		return &user.RoleAttributes, err
	}
	return NewRoleService(s.Dbh, s.GetPathPrefix(), s.GetBaseURL()).dbToCollResourceData(drole), nil
}

func (s *UserService) buildRoleResourceIdentifiers(roles []*user.Role) []*jsonapi.Data {
	jdata := make([]*jsonapi.Data, len(roles))
	for i, r := range roles {
		jdata[i] = &jsonapi.Data{
			Type: r.Type,
			Id:   r.Id,
		}
	}
	return jdata
}

// -- Functions that builds up the various parts of the final user resource objects

func (s *UserService) buildResourceData(id int64, uattr *user.UserAttributes) *user.UserData {
	return &user.UserData{
		Type:       s.GetResourceName(),
		Id:         id,
		Attributes: uattr,
		Relationships: &user.ExistingUserRelationships{
			Roles: &user.ExistingUserRelationships_Roles{
				Links: &jsonapi.Links{
					Self:    aphgrpc.GenSelfRelationshipLink(s, "roles", id),
					Related: aphgrpc.GenRelatedRelationshipLink(s, "roles", id),
				},
			},
		},
		Links: &jsonapi.Links{
			Self: s.genSingularResSelfLink(id),
		},
	}
}

func (s *UserService) buildResource(id int64, uattr *user.UserAttributes) *user.User {
	return &user.User{
		Data: s.buildResourceData(id, uattr),
	}
}

// -- Functions that generates various user resource objects from
//    database user object.

func (s *UserService) dbToResourceAttributes(dusr *dbUser) *user.UserAttributes {
	return &user.UserAttributes{
		FirstName:     dusr.FirstName,
		LastName:      dusr.LastName,
		Email:         dusr.Email,
		IsActive:      dusr.IsActive,
		Organization:  dusr.Organization,
		GroupName:     dusr.GroupName,
		FirstAddress:  dusr.FirstAddress,
		SecondAddress: dusr.SecondAddress,
		City:          dusr.City,
		State:         dusr.State,
		Zipcode:       dusr.Zipcode,
		Country:       dusr.Country,
		Phone:         dusr.Phone,
		CreatedAt:     dusr.CreatedAt,
		UpdatedAt:     dusr.UpdatedAt,
	}
}

func (s *UserService) dbToCollResourceData(dbUsers []*dbUser) []*user.UserData {
	var udata []*user.UserData
	for _, dusr := range dbUsers {
		udata = append(udata, s.buildResourceData(dusr.AuthUserId, s.dbToResourceAttributes(dusr)))
	}
	return udata

}

func (s *UserService) dbToCollResource(dbUsers []*dbUser) (*user.UserCollection, error) {
	return &user.UserCollection{
		Data: s.dbToCollResourceData(dbUsers),
		Links: &jsonapi.PaginationLinks{
			Self: s.genCollResourceSelfLink(),
		},
	}, nil
}

func (s *UserService) dbToCollResourceWithPagination(count int64, dbUsers []*dbUser, pagenum, pagesize int64) (*user.UserCollection, err) {
	udata := s.dbToCollResourceData(dbUsers)
	jsLinks, pages := s.getPagination(count, pagenum, pagesize)
	return &user.UserCollection{
		Data:  udata,
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

func (s *UserService) dbToCollResourceWithRelAndPagination(count int64, dbUsers []*dbUser, pagenum, pagesize int64) (*user.UserCollection, err) {
	udata := s.dbToCollResourceData(dbUsers)
	var allRoles []*user.Role
	for i, _ := range udata {
		roles, err := s.getRoleResourceData(dbUsers[i].AuthUserId)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
		}
		udata[i].Relationships.Roles.Data = s.buildRoleResourceIdentifiers(roles)
		allRoles = append(allRoles, roles...)
	}
	incRoles, err := aphgrpc.ConvertAllToAny(allRoles)
	if err != nil {
		return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
	}
	jsLinks, pages := s.getPagination(count, pagenum, pagesize)
	return &user.UserCollection{
		Data:     udata,
		Links:    jsLinks,
		Included: incRoles,
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

// -- Various utility functions

func (s *UserService) attrTodbCoreUser(attr *user.UserAttributes) *dbCoreUser {
	return &dbCoreUser{
		FirstName: attr.FirstName,
		LastName:  attr.LastName,
		Email:     attr.Email,
		IsActive:  attr.IsActive,
	}
}

func (s *UserService) attrTodbUserInfo(attr *user.UserAttributes) *dbUserInfo {
	return &dbUserInfo{
		Organization:  attr.Organization,
		GroupName:     attr.GroupName,
		FirstAddress:  attr.FirstAddress,
		SecondAddress: attr.SecondAddress,
		City:          attr.City,
		State:         attr.State,
		Zipcode:       attr.Zipcode,
		Country:       attr.Country,
		CreatedAt:     attr.CreatedAt,
		UpdatedAt:     attr.UpdatedAt,
	}
}
