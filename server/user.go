package server

import (
	"fmt"

	"github.com/dictyBase/apihelpers/aphgrpc"
	"github.com/dictyBase/go-genproto/dictybaseapis/api/jsonapi"
	"github.com/dictyBase/go-genproto/dictybaseapis/user"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
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
	usrRoleJoin = `
			auth_user user
			JOIN auth_user_info uinfo
			ON user.auth_user_id = uinfo.auth_user_id
	`
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
		uattr, err := s.getSelectedRows(params, r.Id)
		if err != nil {
			return &user.User{}, aphgrpc.handleError(ctx, err)
		}
		roles, err := s.getRoles(id)
		if err != nil {
			return &user.User{}, aphgrpc.handleError(ctx, err)
		}
		// included relationships
		incRoles, err := convertAllToAny(roles)
		if err != nil {
			return &user.User{}, aphgrpc.handleError(ctx, err)
		}
		u := s.getSingleUserResource(r.Id, uattr)
		u.Included = incRoles
		u.Relationships.Roles.Data = s.getRoleResourceIdentifiers(roles)
		return u, nil
	case params.HasFields:
		s.fieldsStr = r.Fields
		uattr, err := s.getSelectedRows(params, r.Id)
		if err != nil {
			return &user.User{}, aphgrpc.handleError(ctx, err)
		}
		return s.getSingleUserResource(r.Id, uattr), nil
	case params.HasInclude:
		s.includeStr = r.Include
		uattr, err := s.getRow(r.Id)
		if err != nil {
			return &user.User{}, aphgrpc.handleError(ctx, err)
		}
		roles, err := s.getRoles(id)
		if err != nil {
			return &user.User{}, aphgrpc.handleError(ctx, err)
		}
		// included relationships
		incRoles, err := convertAllToAny(roles)
		if err != nil {
			return &user.User{}, aphgrpc.handleError(ctx, err)
		}
		u := s.getSingleUserResource(r.Id, uattr)
		u.Included = incRoles
		u.Relationships.Roles.Data = s.getRoleResourceIdentifiers(roles)
		return u, nil
	default:
		uattr, err := s.getRow(r.Id)
		if err != nil {
			return &user.User{}, aphgrpc.handleError(ctx, err)
		}
		return s.getSingleUserResource(r.Id, uattr), nil
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
			count, err := s.getAllFilteredCount(usrRoleJoin)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			dbUsers, err := s.getAllSelectedFilteredRowsWithPaging(params, r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.getAllUsersWithRelationsAndPagination(count, dbUsers, r.Pagenum, r.Pagesize)
		// fields and includes
		case params.HasFields && params.HasInclude:
			s.fieldsStr = r.Fields
			s.includeStr = r.Include
			count, err := s.getCount("auth_user")
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			dbUsers, err := s.getAllSelectedRowsWithPaging(params, r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.getAllUsersWithRelationsAndPagination(count, dbUsers, r.Pagenum, r.Pagesize)
		// fields and filters
		case params.HasFields && params.HasFilter:
			s.fieldsStr = r.Fields
			s.filterStr = r.Filter
			count, err := s.getAllFilteredCount(usrRoleJoin)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			dbUsers, err := s.getAllSelectedFilteredRowsWithPaging(params, r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.getAllUsersWithPagination(count, dbUsers, r.Pagenum, r.Pagesize)
		// include and filter
		case params.HasInclude && params.HasFilter:
			s.includeStr = r.Include
			s.filterStr = r.Filter
			count, err := s.getAllFilteredCount(usrRoleJoin)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			dbUsers, err := s.getAllFilteredRowsWithPaging(params, r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.getAllUsersWithRelationsAndPagination(count, dbUsers, r.Pagenum, r.Pagesize)
		// only pagination
		default:
			count, err := s.getCount("auth_user")
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			dbUsers, err := s.getAllRowsWithPaging(r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.getAllUsersWithPagination(count, dbUsers, r.Pagenum, r.Pagesize)
		}
	}
	// request without any pagination query parameters
	switch {
	case params.HasFields && params.HasFilter && params.HasInclude:
		s.fieldsStr = r.Fields
		s.filterStr = r.Filter
		s.includeStr = r.Include
		count, err := s.getAllFilteredCount(usrRoleJoin)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			dbUsers, err := s.getAllSelectedFilteredRowsWithPaging(params, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.getAllUsersWithRelationsAndPagination(count, dbUsers, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		}
		return s.getAllDefaultUsers()
	case params.HasFields && params.HasFilter:
		s.fieldsStr = r.Fields
		s.filterStr = r.Filter
		count, err := s.getAllFilteredCount(usrRoleJoin)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			dbUsers, err := s.getAllSelectedFilteredRowsWithPaging(params, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.getAllUsersWithPagination(count, dbUsers, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		}
		return s.getAllDefaultUsers()
	case params.HasFields && params.HasInclude:
		s.fieldsStr = r.Fields
		s.includeStr = r.Include
		count, err := s.getCount("auth_user")
		if err != nil {
			return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			dbUsers, err := s.getAllSelectedRowsWithPaging(params, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.getAllUsersWithRelationsAndPagination(count, dbUsers, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		}
		return s.getAllDefaultUsers()
	case params.HasFilter && params.HasInclude:
		s.includeStr = r.Include
		s.filterStr = r.Filter
		count, err := s.getAllFilteredCount(usrRoleJoin)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			dbUsers, err := s.getAllFilteredRowsWithPaging(params, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
			}
			return s.getAllUsersWithRelationsAndPagination(count, dbUsers, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		}
		return s.getAllDefaultUsers()
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
			return s.getAllUsersWithPagination(count, dbUsers, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		}
		return s.getAllDefaultUsers()
	}
}

func (s *UserService) CreateUser(ctx context.Context, r *user.CreateUserRequest) (*user.User, error) {
	var userId int64
	dbcuser := s.mapAttrTodbCoreUser(r.Data.Attributes)
	_, err := s.Dbh.InsertInto("auth_user").
		Columns(coreUserCols...).
		Record(dbcuser).
		Returning("auth_user_id").QueryScalar(&userId)
	if err != nil {
		grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseInsert)
		return &user.User{}, status.Error(codes.Internal, err.Error())
	}
	dbusrInfo := s.mapAttrTodbUserInfo(r.Data.Attributes)
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
	if err := s.hasUser(r.Data.Id); err != nil {
		return &user.User{}, aphgrpc.handleError(ctx, err)
	}
	dbcuser := s.mapAttrTodbCoreUser(r.Data.Attributes)
	usrMap := aphgrpc.GetDefinedTagsWithValue(dbcuser, "db")
	if len(usrMap) > 0 {
		_, err := s.Dbh.Update("auth_user").SetMap(usrMap).
			Where("auth_user_id = $1", r.Data.Id).Exec()
		if err != nil {
			grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseUpdate)
			return &user.User{}, status.Error(codes.Internal, err.Error())
		}
	}
	dbusrInfo := s.mapAttrTodbUserInfo(r.Data.Attributes)
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
	return getSingleUserData(r.Data.Id, r.Data.Attributes), nil
}

func (s *UserService) DeleteUser(ctx context.Context, r *jsonapi.DeleteRequest) (*empty.Empty, error) {
	if err := s.hasUser(r.Data.Id); err != nil {
		return &empty.Empty{}, aphgrpc.handleError(ctx, err)
	}
	_, err := s.Dbh.DeleteFrom("auth_user").Where("auth_user_id = $1", r.Id).Exec()
	if err != nil {
		grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseDelete)
		return &empty.Empty{}, status.Error(codes.Internal, err.Error())
	}
	return &empty.Empty{}, nil
}

func (s *UserService) getSelectedRows(id int64) (*user.UserAttributes, error) {
	dusr := new(dbUser)
	columns := s.fieldsToColumns(s.params.Fields)
	err := s.Dbh.Select(columns...).From(`
			auth_user user
			JOIN auth_user_info uinfo
			ON user.auth_user_id = uinfo.auth_user_id
		`).Where("user.auth_user_id = $1", id).QueryStruct(dusr)
	if err != nil {
		return &user.UserAttributes{}, err
	}
	return mapUserAttributes(dusr), nil
}

func (s *UserService) hasUser(id int64) error {
	return s.Dbh.Select("auth_user_id").From("auth_user").
		Where("auth_user_id = $1", id).Exec()
}

func (s *UserService) getRow(id int64) (*user.UserAttributes, error) {
	dusr := new(dbUser)
	err := s.Dbh.Select("user.*", "uinfo.*").
		From(usrRoleJoin).
		Where("user.auth_user_id = $1", id).
		QueryStruct(dusr)
	if err != nil {
		return &user.UserAttributes{}, err
	}
	return mapUserAttributes(dusr), nil
}

func (s *UserService) getAllRows() ([]*dbUser, error) {
	var dusrRows []*dbUser
	err := s.Dbh.Select("user.*", "uinfo.*").
		From(usrRoleJoin).
		QueryStructs(dusrRows)
	return dusrRows, err
}

func (s *UserService) getAllRowsWithPaging(pageNum int64, pageSize int64) ([]*dbUser, error) {
	var dusrRows []*dbUser
	err := s.Dbh.Select("user.*", "uinfo.*").
		From(usrRoleJoin).
		Paginate(uint64(pageNum), uint64(pageSize)).
		QueryStructs(dusrRows)
	return dusrRows, err
}

func (s *UserService) getAllSelectedRowsWithPaging(pagenum, pagesize int64) ([]*dbUser, error) {
	var dusrRows []*dbUser
	columns := s.MapFieldsToColumns(s.params.Fields)
	err := s.Dbh.Select(columns...).
		From(usrRoleJoin).
		Paginate(uint64(pageNum), uint64(pageSize)).
		QueryStructs(dusrRows)
	return dusrRows, err
}

func (s *UserService) getAllFilteredRowsWithPaging(pagenum, pagesize int64) ([]*dbUser, error) {
	var dusrRows []*dbUser
	err := s.Dbh.Select("user.*", "uinfo.*").
		From(usrRoleJoin).
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
		From(usrRoleJoin).
		Scope(
			aphgrpc.FilterToWhereClause(s, s.params.Filter),
			aphgrpc.FilterToBindValue(s.params.Filter)...,
		).
		Paginate(uint64(pageNum), uint64(pageSize)).
		QueryStructs(dusrRows)
	return dusrRows, err
}

func (s *UserService) getRoles(id int64) ([]*user.RoleData, error) {
	var drole []*dbRole
	err := s.Dbh.Select("role.*").From(`
			auth_user_role
			JOIN auth_role role
			ON auth_user_role.auth_role_id = role.auth_role_id
		`).Where("auth_user_role.auth_user_id = $1", id).QueryStructs(drole)
	if err != nil {
		return &user.RoleAttributes, err
	}
	rsrv := NewRoleService(s.Dbh, s.GetPathPrefix(), s.GetBaseURL())
	var rdata []*user.RoleData
	for _, dr := range drole {
		rd := &User.RoleData{
			Type:       rsrv.GetResourceName(),
			Id:         dr.AuthRoleId,
			Attributes: mapRoleAttributes(dr),
			Links: &jsonapi.Links{
				Self: aphgrpc.GenSingleResourceLink(rsrv, dr.AuthRoleId),
			},
		}
		rdata = append(rdata, rd)
	}
	return rdata, nil
}

func (s *UserService) getRoleResourceIdentifiers(roles []*user.Role) []*jsonapi.Data {
	jdata := make([]*jsonapi.Data, len(roles))
	for i, r := range roles {
		jdata[i] = &jsonapi.Data{
			Type: r.Type,
			Id:   r.Id,
		}
	}
	return jdata
}

func (s *UserService) getSingleUserData(id int64, uattr *user.UserAttributes) *user.UserData {
	links := aphgrpc.GenSingleResourceLink(s, id)
	if !s.IsListMethod() && s.params != nil {
		params := s.params
		switch {
		case params.HasFields && params.HasIncludes:
			links += fmt.Sprintf("?fields=%s&include=%s", s.fieldsStr, s.includeStr)
		case params.HasFields:
			links += fmt.Sprintf("?fields=%s", s.fieldsStr)
		case params.HasIncludes:
			links += fmt.Sprintf("?include=%s", s.includeStr)
		}
	}
	return &user.UserData{
		Type:       s.GetResourceName(),
		Id:         id,
		Attributes: uattr,
		Relationships: &user.ExistingUserRelationships{
			Roles: &user.ExistingUserRelationships_Roles{
				Links: &jsonapi.Links{
					Self:    aphgrpc.GenSelfRelationshipLink(s, "roles", r.Id),
					Related: aphgrpc.GenRelatedRelationshipLink(s, "roles", r.Id),
				},
			},
		},
		Links: &jsonapi.Links{
			Self: links,
		},
	}
}

func (s *UserService) getSingleUserResource(id int64, uattr *user.UserAttributes) *user.User {
	return &user.User{
		Data: s.getSingleUserData(id, uattr),
	}
}

func (s *UserService) getAllUserData([]*dbUser) []*user.UserData {
	var udata []*user.UserData
	for _, dusr := range dbUsers {
		udata = append(udata, s.getSingleUserData(dusr.AuthUserId, mapUserAttributes(dusr)))
	}
	return udata

}

func (s *Service) getAllDefaultUsers(dbUsers []*dbUser) ([]*user.UserCollection, error) {
	dbUsers, err := s.getAllRows()
	if err != nil {
		return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
	}
	return &user.UserCollection{
		Data: s.getAllUserData(dbUsers),
		Links: &jsonapi.PaginationLinks{
			Self: aphgrpc.GenMultiResourceLink(s),
		},
	}, nil
}

func (s *UserService) getAllUsers(dbUsers []*dbUser) ([]*user.UserCollection, error) {
	link := aphgrpc.GenMultiResourceLink(s)
	params := s.params
	switch {
	case params.HasFields && params.HasFilter && params.HasInclude:
		link += fmt.Sprintf("%s&fields=%s&include=%s&filter=%s", s.fieldsStr, s.includeStr, s.filterStr)
	case params.HasFields && params.HasFilter:
		link += fmt.Sprintf("%s&fields=%s&filter=%s", s.fieldsStr, s.filterStr)
	case params.HasFields && params.HasInclude:
		link += fmt.Sprintf("%s&fields=%s&include=%s", s.fieldsStr, s.includeStr)
	case params.HasFilter && params.HasInclude:
		link += fmt.Sprintf("%s&fiter=%s&include=%s", s.filterStr, s.includeStr)
	}
	return &user.UserCollection{
		Data: s.getAllUserData(dbUsers),
		Links: &jsonapi.PaginationLinks{
			Self: link,
		},
	}, nil
}

func (s *UserService) getAllUsersWithPagination(count int64, dbUsers []*dbUser, pagenum, pagesize int64) ([]*user.UserCollection, err) {
	udata := s.getAllUserData(dbUsers)
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

func (s *UserService) getAllUsersWithRelationsAndPagination(count int64, dbUsers []*dbUser, pagenum, pagesize int64) ([]*user.UserCollection, err) {
	udata := s.getAllUserData(dbUsers)
	var allRoles []*user.Role
	for i, _ := range udata {
		roles, err := s.getRoles(dbUsers[i].AuthUserId)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.handleError(ctx, err)
		}
		udata[i].Relationships.Roles.Data = s.getRoleResourceIdentifiers(roles)
		allRoles = append(allRoles, roles...)
	}
	incRoles, err := convertAllToAny(allRoles)
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

func (s *UserService) mapAttrTodbCoreUser(attr *user.UserAttributes) *dbCoreUser {
	return &dbCoreUser{
		FirstName: attr.FirstName,
		LastName:  attr.LastName,
		Email:     attr.Email,
		IsActive:  attr.IsActive,
	}
}

func (s *UserService) mapAttrTodbUserInfo(attr *user.UserAttributes) *dbUserInfo {
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

func mapUserAttributes(dusr *dbUser) *user.UserAttributes {
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

func convertAllToAny(msg []proto.Message) ([]*any.Any, error) {
	as := make([]*any.Any, len(msg))
	for i, p := range msg {
		pkg, err := ptypes.MarshalAny(p)
		if err != nil {
			return as, err
		}
		as[i] = pkg
	}
	return as, nil
}
