package server

import (
	"context"
	"fmt"

	"github.com/dictyBase/apihelpers/aphgrpc"
	"github.com/dictyBase/go-genproto/dictybaseapis/api/jsonapi"
	"github.com/dictyBase/go-genproto/dictybaseapis/user"
	"github.com/fatih/structs"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	dat "gopkg.in/mgutz/dat.v2/dat"
	runner "gopkg.in/mgutz/dat.v2/sqlx-runner"
)

const (
	usrTablesJoin = `
			auth_user user
			JOIN auth_user_info uinfo
			ON user.auth_user_id = uinfo.auth_user_id
	`
	userDbTable = "auth_user user"
)

var coreUserCols = []string{
	"first_name",
	"last_name",
	"email",
	"is_active",
}
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
	AuthUserId int64        `db:"auth_user_id"`
	FirstName  string       `db:"first_name"`
	LastName   string       `db:"last_name"`
	Email      string       `db:"email"`
	IsActive   bool         `db:"is_active"`
	CreatedAt  dat.NullTime `db:"created_at"`
	UpdatedAt  dat.NullTime `db:"updated_at"`
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
}

type UserService struct {
	*aphgrpc.Service
}

func NewUserService(dbh *runner.DB, pathPrefix string) *UserService {
	return &UserService{
		&aphgrpc.Service{
			Resource:   "users",
			Dbh:        dbh,
			PathPrefix: pathPrefix,
			Include:    []string{"roles"},
			FilToColumns: map[string]string{
				"first_name": "user.first_name",
				"last_name":  "user.last_name",
				"email":      "user.email",
			},
			FieldsToColumns: map[string]string{
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
			ReqAttrs: []string{"FirstName", "LastName", "Email"},
		},
	}
}

func (s *UserService) GetUser(ctx context.Context, r *jsonapi.GetRequest) (*user.User, error) {
	params, md, err := aphgrpc.ValidateAndParseGetParams(s, r)
	if err != nil {
		grpc.SetTrailer(ctx, md)
		return new(user.User), status.Error(codes.InvalidArgument, err.Error())
	}
	s.Params = params
	s.ListMethod = false
	s.SetBaseURL(ctx)
	switch {
	case params.HasFields && params.HasInclude:
		s.IncludeStr = r.Include
		s.FieldsStr = r.Fields
		u, err := s.getResourceWithSelectedAttr(r.Id)
		if err != nil {
			return &user.User{}, aphgrpc.HandleError(ctx, err)
		}
		err = s.buildResourceRelationships(r.Id, u)
		if err != nil {
			return &user.User{}, aphgrpc.HandleError(ctx, err)
		}
		return u, nil
	case params.HasFields:
		s.FieldsStr = r.Fields
		u, err := s.getResourceWithSelectedAttr(r.Id)
		if err != nil {
			return &user.User{}, aphgrpc.HandleError(ctx, err)
		}
		return u, nil
	case params.HasInclude:
		s.IncludeStr = r.Include
		u, err := s.getResource(r.Id)
		if err != nil {
			return &user.User{}, aphgrpc.HandleError(ctx, err)
		}
		err = s.buildResourceRelationships(r.Id, u)
		if err != nil {
			return &user.User{}, aphgrpc.HandleError(ctx, err)
		}
		return u, nil
	default:
		u, err := s.getResource(r.Id)
		if err != nil {
			return &user.User{}, aphgrpc.HandleError(ctx, err)
		}
		return u, nil
	}
}

func (s *UserService) GetRelatedRoles(ctx context.Context, r *jsonapi.RelationshipRequest) (*user.RoleCollection, error) {
	rdata, err := s.getRoleResourceData(r.Id)
	if err != nil {
		return &user.RoleCollection{}, aphgrpc.HandleError(ctx, err)
	}
	return &user.RoleCollection{
		Data: rdata,
		Links: &jsonapi.Links{
			Self: s.GenCollResourceRelSelfLink(r.Id, "roles"),
		},
	}, nil
}

func (s *UserService) ListUsers(ctx context.Context, r *jsonapi.ListRequest) (*user.UserCollection, error) {
	params, md, err := aphgrpc.ValidateAndParseListParams(s, r)
	if err != nil {
		grpc.SetTrailer(ctx, md)
		return &user.UserCollection{}, status.Error(codes.InvalidArgument, err.Error())
	}
	s.Params = params
	s.ListMethod = true
	s.SetBaseURL(ctx)
	// has pagination query parameters
	if aphgrpc.HasPagination(r) {
		switch {
		// filter, fields and include parameters
		case params.HasFields && params.HasInclude && params.HasFilter:
			s.FieldsStr = r.Fields
			s.FilterStr = r.Filter
			s.IncludeStr = r.Include
			count, err := s.GetAllFilteredCount(usrTablesJoin)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
			}
			dbUsers, err := s.getAllSelectedFilteredRowsWithPaging(r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
			}
			return s.dbToCollResourceWithRelAndPagination(count, dbUsers, r.Pagenum, r.Pagesize)
		// fields and includes
		case params.HasFields && params.HasInclude:
			s.FieldsStr = r.Fields
			s.IncludeStr = r.Include
			count, err := s.GetCount(userDbTable)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
			}
			dbUsers, err := s.getAllSelectedRowsWithPaging(r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
			}
			return s.dbToCollResourceWithRelAndPagination(count, dbUsers, r.Pagenum, r.Pagesize)
		// fields and filters
		case params.HasFields && params.HasFilter:
			s.FieldsStr = r.Fields
			s.FilterStr = r.Filter
			count, err := s.GetAllFilteredCount(usrTablesJoin)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
			}
			dbUsers, err := s.getAllSelectedFilteredRowsWithPaging(r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
			}
			return s.dbToCollResourceWithPagination(count, dbUsers, r.Pagenum, r.Pagesize), nil
		// include and filter
		case params.HasInclude && params.HasFilter:
			s.IncludeStr = r.Include
			s.FilterStr = r.Filter
			count, err := s.GetAllFilteredCount(usrTablesJoin)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
			}
			dbUsers, err := s.getAllFilteredRowsWithPaging(r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
			}
			return s.dbToCollResourceWithRelAndPagination(count, dbUsers, r.Pagenum, r.Pagesize)
		case params.HasFields:
			s.FieldsStr = r.Fields
			count, err := s.GetCount(userDbTable)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
			}
			dbUsers, err := s.getAllSelectedRowsWithPaging(r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
			}
			return s.dbToCollResourceWithPagination(count, dbUsers, r.Pagenum, r.Pagesize), nil
		case params.HasFilter:
			s.FilterStr = r.Filter
			count, err := s.GetAllFilteredCount(usrTablesJoin)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
			}
			dbUsers, err := s.getAllFilteredRowsWithPaging(r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
			}
			return s.dbToCollResourceWithPagination(count, dbUsers, r.Pagenum, r.Pagesize), nil
		case params.HasInclude:
			s.IncludeStr = r.Include
			count, err := s.GetCount(userDbTable)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
			}
			dbUsers, err := s.getAllRowsWithPaging(r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
			}
			return s.dbToCollResourceWithRelAndPagination(count, dbUsers, r.Pagenum, r.Pagesize)
		// only pagination
		default:
			count, err := s.GetCount(userDbTable)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
			}
			dbUsers, err := s.getAllSelectedRowsWithPaging(r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
			}
			return s.dbToCollResourceWithPagination(count, dbUsers, r.Pagenum, r.Pagesize), nil
		}
	}
	// request without any pagination query parameters
	switch {
	case params.HasFields && params.HasFilter && params.HasInclude:
		s.FieldsStr = r.Fields
		s.FilterStr = r.Filter
		s.IncludeStr = r.Include
		count, err := s.GetAllFilteredCount(usrTablesJoin)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
		}
		dbUsers, err := s.getAllSelectedFilteredRowsWithPaging(aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			return s.dbToCollResourceWithRelAndPagination(count, dbUsers, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		}
		return s.dbToCollResource(dbUsers), nil
	case params.HasFields && params.HasFilter:
		s.FieldsStr = r.Fields
		s.FilterStr = r.Filter
		count, err := s.GetAllFilteredCount(usrTablesJoin)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
		}
		dbUsers, err := s.getAllSelectedFilteredRowsWithPaging(aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			return s.dbToCollResourceWithPagination(count, dbUsers, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize), nil
		}
		return s.dbToCollResource(dbUsers), nil
	case params.HasFields && params.HasInclude:
		s.FieldsStr = r.Fields
		s.IncludeStr = r.Include
		count, err := s.GetCount("auth_user")
		if err != nil {
			return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
		}
		dbUsers, err := s.getAllSelectedRowsWithPaging(aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			return s.dbToCollResourceWithRelAndPagination(count, dbUsers, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		}
		return s.dbToCollResource(dbUsers), nil
	case params.HasFilter && params.HasInclude:
		s.IncludeStr = r.Include
		s.FilterStr = r.Filter
		count, err := s.GetAllFilteredCount(usrTablesJoin)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
		}
		dbUsers, err := s.getAllFilteredRowsWithPaging(aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			return s.dbToCollResourceWithRelAndPagination(count, dbUsers, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		}
		return s.dbToCollResource(dbUsers), nil
	case params.HasFields:
		s.FieldsStr = r.Fields
		count, err := s.GetCount(userDbTable)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
		}
		dbUsers, err := s.getAllSelectedRowsWithPaging(aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			return s.dbToCollResourceWithPagination(count, dbUsers, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize), nil
		}
		return s.dbToCollResource(dbUsers), nil
	case params.HasFilter:
		s.FilterStr = r.Filter
		count, err := s.GetAllFilteredCount(usrTablesJoin)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
		}
		dbUsers, err := s.getAllFilteredRowsWithPaging(aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			return s.dbToCollResourceWithPagination(count, dbUsers, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize), nil
		}
		return s.dbToCollResource(dbUsers), nil
	case params.HasInclude:
		s.IncludeStr = r.Include
		count, err := s.GetCount(userDbTable)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
		}
		dbUsers, err := s.getAllRowsWithPaging(aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			return s.dbToCollResourceWithRelAndPagination(count, dbUsers, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		}
		return s.dbToCollResource(dbUsers), nil
	default:
		count, err := s.GetCount("auth_user")
		if err != nil {
			return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
		}
		dbUsers, err := s.getAllRowsWithPaging(aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			return s.dbToCollResourceWithPagination(count, dbUsers, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize), nil
		}
		return s.dbToCollResource(dbUsers), nil
	}
}

func (s *UserService) CreateUser(ctx context.Context, r *user.CreateUserRequest) (*user.User, error) {
	dbcuser := s.attrTodbCoreUser(r.Data.Attributes)
	retcols := []string{"auth_user_id", "created_at", "updated_at"}
	err := s.Dbh.InsertInto("auth_user").
		Columns(coreUserCols...).
		Record(dbcuser).
		Returning(retcols...).
		QueryStruct(dbcuser)
	if err != nil {
		grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseInsert)
		return &user.User{}, status.Error(codes.Internal, err.Error())
	}
	dbusrInfo := s.attrTodbUserInfo(r.Data.Attributes)
	dbusrInfo.AuthUserId = dbcuser.AuthUserId
	defUsrInfoCols := aphgrpc.GetDefinedTags(dbusrInfo, "db")
	if len(defUsrInfoCols) > 0 {
		err = s.Dbh.InsertInto("auth_user_info").
			Columns(defUsrInfoCols...).
			Record(dbusrInfo).
			Returning(userInfoCols...).
			QueryStruct(dbusrInfo)
		if err != nil {
			grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseInsert)
			return &user.User{}, status.Error(codes.Internal, err.Error())
		}
	}
	rstruct := structs.New(r).Field("Data").Field("Relationships")
	if !rstruct.IsZero() {
		if !rstruct.Field("Roles").IsZero() {
			for _, role := range r.Data.Relationships.Roles.Data {
				_, err = s.Dbh.InsertInto("auth_user_role").
					Columns("auth_user_id", "auth_role_id").
					Values(dbcuser.AuthUserId, role.Id).Exec()
				if err != nil {
					grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseInsert)
					return &user.User{}, status.Error(codes.Internal, err.Error())
				}
			}
		}
	}
	s.SetBaseURL(ctx)
	grpc.SetTrailer(ctx, metadata.Pairs("method", "POST"))
	return s.buildResource(
		dbcuser.AuthUserId,
		s.dbToResourceAttributes(s.mergeTodbUser(dbcuser, dbusrInfo)),
	), nil
}

func (s *UserService) CreateRoleRelationship(ctx context.Context, r *jsonapi.DataCollection) (*empty.Empty, error) {
	result, err := s.existsResource(r.Id)
	if err != nil {
		return &empty.Empty{}, aphgrpc.HandleError(ctx, err)
	}
	if !result {
		grpc.SetTrailer(ctx, aphgrpc.ErrNotFound)
		return &empty.Empty{}, status.Error(codes.NotFound, fmt.Sprintf("id %d not found", r.Id))
	}
	for _, rd := range r.Data {
		res, err := s.Dbh.Select("aurole.auth_user_role_id").
			From("auth_user_role aurole").
			Where("aurole.auth_user_id = $1 AND aurole.auth_role_id = $2", r.Id, rd.Id).
			Exec()
		if err != nil {
			grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseInsert)
			return &empty.Empty{}, status.Error(codes.Internal, err.Error())
		}
		if res.RowsAffected != 1 {
			_, err := s.Dbh.InsertInto("auth_user_role").
				Columns("auth_user_id", "auth_role_id").
				Values(r.Id, rd.Id).Exec()
			if err != nil {
				grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseInsert)
				return &empty.Empty{}, status.Error(codes.Internal, err.Error())
			}
		}
	}
	grpc.SetTrailer(ctx, metadata.Pairs("method", "POST_NO_CONTENT"))
	return &empty.Empty{}, nil
}

func (s *UserService) UpdateUser(ctx context.Context, r *user.UpdateUserRequest) (*user.User, error) {
	result, err := s.existsResource(r.Id)
	if err != nil {
		return &user.User{}, aphgrpc.HandleError(ctx, err)
	}
	if !result {
		grpc.SetTrailer(ctx, aphgrpc.ErrNotFound)
		return &user.User{}, status.Error(codes.NotFound, fmt.Sprintf("id %d not found", r.Id))
	}
	dbcuser := s.attrTodbCoreUser(r.Data.Attributes)
	usrMap := aphgrpc.GetDefinedTagsWithValue(dbcuser, "db")
	if len(usrMap) > 0 {
		err := s.Dbh.Update("auth_user").
			SetMap(usrMap).
			Where("auth_user_id = $1", r.Data.Id).
			Returning([]string{"created_at", "updated_at"}...).
			QueryStruct(dbcuser)
		if err != nil {
			grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseUpdate)
			return &user.User{}, status.Error(codes.Internal, err.Error())
		}
	}
	dbusrInfo := s.attrTodbUserInfo(r.Data.Attributes)
	usrInfoMap := aphgrpc.GetDefinedTagsWithValue(dbusrInfo, "db")
	if len(usrInfoMap) > 0 {
		err := s.Dbh.Update("auth_user_info").
			SetMap(usrInfoMap).
			Where("auth_user_id = $1", r.Data.Id).
			Returning(userInfoCols...).
			QueryStruct(dbusrInfo)
		if err != nil {
			grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseUpdate)
			return &user.User{}, status.Error(codes.Internal, err.Error())
		}
	}
	rstruct := structs.New(r).Field("Data").Field("Relationships")
	if !rstruct.IsZero() {
		if !rstruct.Field("Roles").IsZero() {
			for _, role := range r.Data.Relationships.Roles.Data {
				_, err := s.Dbh.Update("auth_user_role").
					Set("auth_role_id", role.Id).
					Where("auth_user_id = $1", r.Data.Id).Exec()
				if err != nil {
					grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseUpdate)
					return &user.User{}, status.Error(codes.Internal, err.Error())
				}
			}
		}
	}
	s.SetBaseURL(ctx)
	return s.buildResource(
		r.Data.Id,
		s.dbToResourceAttributes(s.mergeTodbUser(dbcuser, dbusrInfo)),
	), nil
}

func (s *UserService) UpdateRoleRelationship(ctx context.Context, r *jsonapi.DataCollection) (*empty.Empty, error) {
	result, err := s.existsResource(r.Id)
	if err != nil {
		return &empty.Empty{}, aphgrpc.HandleError(ctx, err)
	}
	if !result {
		grpc.SetTrailer(ctx, aphgrpc.ErrNotFound)
		return &empty.Empty{}, status.Error(codes.NotFound, fmt.Sprintf("id %d not found", r.Id))
	}
	_, err = s.Dbh.DeleteFrom("auth_user_role").
		Where("auth_user_role.auth_user_id = $1", r.Id).
		Exec()
	if err != nil {
		grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseUpdate)
		return &empty.Empty{}, status.Error(codes.Internal, err.Error())
	}
	for _, rd := range r.Data {
		_, err := s.Dbh.InsertInto("auth_user_role").
			Columns("auth_user_id", "auth_role_id").
			Values(r.Id, rd.Id).Exec()
		if err != nil {
			grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseUpdate)
			return &empty.Empty{}, status.Error(codes.Internal, err.Error())
		}
	}
	return &empty.Empty{}, nil
}

func (s *UserService) DeleteUser(ctx context.Context, r *jsonapi.DeleteRequest) (*empty.Empty, error) {
	result, err := s.existsResource(r.Id)
	if err != nil {
		return &empty.Empty{}, aphgrpc.HandleError(ctx, err)
	}
	if !result {
		grpc.SetTrailer(ctx, aphgrpc.ErrNotFound)
		return &empty.Empty{}, status.Error(codes.NotFound, fmt.Sprintf("id %d not found", r.Id))
	}
	_, err = s.Dbh.DeleteFrom("auth_user").Where("auth_user_id = $1", r.Id).Exec()
	if err != nil {
		grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseDelete)
		return &empty.Empty{}, status.Error(codes.Internal, err.Error())
	}
	return &empty.Empty{}, nil
}

func (s *UserService) DeleteRoleRelationship(ctx context.Context, r *jsonapi.DataCollection) (*empty.Empty, error) {
	result, err := s.existsResource(r.Id)
	if err != nil {
		return &empty.Empty{}, aphgrpc.HandleError(ctx, err)
	}
	if !result {
		grpc.SetTrailer(ctx, aphgrpc.ErrNotFound)
		return &empty.Empty{}, status.Error(codes.NotFound, fmt.Sprintf("id %d not found", r.Id))
	}
	for _, rd := range r.Data {
		_, err := s.Dbh.DeleteFrom("auth_user_role").
			Where("auth_user_role.auth_user_id = $1 AND auth_user_role.auth_role_id = $2", r.Id, rd.Id).
			Exec()
		if err != nil {
			grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseDelete)
			return &empty.Empty{}, status.Error(codes.Internal, err.Error())
		}
	}
	return &empty.Empty{}, nil
}

// All helper functions

func (s *UserService) existsResource(id int64) (bool, error) {
	r, err := s.Dbh.Select("auth_user_id").From("auth_user").
		Where("auth_user_id = $1", id).Exec()
	if err != nil {
		return false, err
	}
	if r.RowsAffected != 1 {
		return false, nil
	}
	return true, nil
}

// -- Functions that queries the storage and generates an user resource object

func (s *UserService) getResourceWithSelectedAttr(id int64) (*user.User, error) {
	dusr := new(dbUser)
	columns := s.MapFieldsToColumns(s.Params.Fields)
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
	columns := s.MapFieldsToColumns(s.Params.Fields)
	err := s.Dbh.Select(columns...).
		From(usrTablesJoin).
		Paginate(uint64(pagenum), uint64(pagesize)).
		QueryStructs(dusrRows)
	return dusrRows, err
}

func (s *UserService) getAllFilteredRowsWithPaging(pagenum, pagesize int64) ([]*dbUser, error) {
	var dusrRows []*dbUser
	err := s.Dbh.Select("user.*", "uinfo.*").
		From(usrTablesJoin).
		Scope(
			aphgrpc.FilterToWhereClause(s, s.Params.Filters),
			aphgrpc.FilterToBindValue(s.Params.Filters)...,
		).
		Paginate(uint64(pagenum), uint64(pagesize)).
		QueryStructs(dusrRows)
	return dusrRows, err
}

func (s *UserService) getAllSelectedFilteredRowsWithPaging(pagenum, pagesize int64) ([]*dbUser, error) {
	var dusrRows []*dbUser
	columns := s.MapFieldsToColumns(s.Params.Fields)
	err := s.Dbh.Select(columns...).
		From(usrTablesJoin).
		Scope(
			aphgrpc.FilterToWhereClause(s, s.Params.Filters),
			aphgrpc.FilterToBindValue(s.Params.Filters)...,
		).
		Paginate(uint64(pagenum), uint64(pagesize)).
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
		return rdata, err
	}
	return NewRoleService(s.Dbh, s.GetPathPrefix()).dbToCollResourceData(drole), nil
}

func (s *UserService) buildRoleResourceIdentifiers(roles []*user.RoleData) []*jsonapi.Data {
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
			Self: s.GenResourceSelfLink(id),
		},
	}
}

func (s *UserService) buildResource(id int64, uattr *user.UserAttributes) *user.User {
	return &user.User{
		Data: s.buildResourceData(id, uattr),
		Links: &jsonapi.Links{
			Self: s.GenResourceSelfLink(id),
		},
	}
}

func (s *UserService) buildResourceRelationships(id int64, user *user.User) error {
	var allInc []*any.Any
	roles, err := s.getRoleResourceData(id)
	if err != nil {
		return err
	}
	incRoles, err := NewRoleService(s.Dbh, "roles").convertAllToAny(roles)
	if err != nil {
		return err
	}
	allInc = append(allInc, incRoles...)
	user.Data.Relationships.Roles.Data = s.buildRoleResourceIdentifiers(roles)
	user.Included = allInc
	return nil
}

// -- Functions that generates various user resource objects from
//    database user object.

func (s *UserService) dbToResourceAttributes(dusr *dbUser) *user.UserAttributes {
	return &user.UserAttributes{
		FirstName:     dusr.FirstName,
		LastName:      dusr.LastName,
		Email:         dusr.Email,
		IsActive:      dusr.IsActive,
		Organization:  aphgrpc.NullToString(dusr.Organization),
		GroupName:     aphgrpc.NullToString(dusr.GroupName),
		FirstAddress:  aphgrpc.NullToString(dusr.FirstAddress),
		SecondAddress: aphgrpc.NullToString(dusr.SecondAddress),
		City:          aphgrpc.NullToString(dusr.City),
		State:         aphgrpc.NullToString(dusr.State),
		Zipcode:       aphgrpc.NullToString(dusr.Zipcode),
		Country:       aphgrpc.NullToString(dusr.Country),
		Phone:         aphgrpc.NullToString(dusr.Phone),
		CreatedAt:     aphgrpc.NullToTime(dusr.CreatedAt),
		UpdatedAt:     aphgrpc.NullToTime(dusr.UpdatedAt),
	}
}

func (s *UserService) dbToCollResourceData(dbUsers []*dbUser) []*user.UserData {
	var udata []*user.UserData
	for _, dusr := range dbUsers {
		udata = append(udata, s.buildResourceData(dusr.AuthUserId, s.dbToResourceAttributes(dusr)))
	}
	return udata

}

func (s *UserService) dbToCollResource(dbUsers []*dbUser) *user.UserCollection {
	return &user.UserCollection{
		Data: s.dbToCollResourceData(dbUsers),
		Links: &jsonapi.PaginationLinks{
			Self: s.GenCollResourceSelfLink(),
		},
	}
}

func (s *UserService) dbToCollResourceWithPagination(count int64, dbUsers []*dbUser, pagenum, pagesize int64) *user.UserCollection {
	udata := s.dbToCollResourceData(dbUsers)
	jsLinks, pages := s.GetPagination(count, pagenum, pagesize)
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
	}
}

func (s *UserService) dbToCollResourceWithRelAndPagination(count int64, dbUsers []*dbUser, pagenum, pagesize int64) (*user.UserCollection, error) {
	udata := s.dbToCollResourceData(dbUsers)
	var allRoles []*user.RoleData
	for i, _ := range udata {
		roles, err := s.getRoleResourceData(dbUsers[i].AuthUserId)
		if err != nil {
			return &user.UserCollection{}, err
		}
		udata[i].Relationships.Roles.Data = s.buildRoleResourceIdentifiers(roles)
		allRoles = append(allRoles, roles...)
	}
	incRoles, err := NewRoleService(s.Dbh, "roles").convertAllToAny(allRoles)
	if err != nil {
		return &user.UserCollection{}, err
	}
	jsLinks, pages := s.GetPagination(count, pagenum, pagesize)
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

func (s *UserService) mergeTodbUser(dbcuser *dbCoreUser, dbusrInfo *dbUserInfo) *dbUser {
	return &dbUser{
		AuthUserId:    dbcuser.AuthUserId,
		FirstName:     dbcuser.FirstName,
		LastName:      dbcuser.LastName,
		Email:         dbcuser.Email,
		IsActive:      dbcuser.IsActive,
		CreatedAt:     dbcuser.CreatedAt,
		UpdatedAt:     dbcuser.UpdatedAt,
		Organization:  dbusrInfo.Organization,
		GroupName:     dbusrInfo.GroupName,
		FirstAddress:  dbusrInfo.FirstAddress,
		SecondAddress: dbusrInfo.SecondAddress,
		City:          dbusrInfo.City,
		State:         dbusrInfo.State,
		Zipcode:       dbusrInfo.Zipcode,
		Country:       dbusrInfo.Country,
		Phone:         dbusrInfo.Phone,
	}
}

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
		Organization:  dat.NullStringFrom(attr.Organization),
		GroupName:     dat.NullStringFrom(attr.GroupName),
		FirstAddress:  dat.NullStringFrom(attr.FirstAddress),
		SecondAddress: dat.NullStringFrom(attr.SecondAddress),
		City:          dat.NullStringFrom(attr.City),
		State:         dat.NullStringFrom(attr.State),
		Zipcode:       dat.NullStringFrom(attr.Zipcode),
		Country:       dat.NullStringFrom(attr.Country),
	}
}

func (s *UserService) convertAllToAny(users []*user.UserData) ([]*any.Any, error) {
	aslice := make([]*any.Any, len(users))
	for i, u := range users {
		pkg, err := ptypes.MarshalAny(u)
		if err != nil {
			return aslice, err
		}
		aslice[i] = pkg
	}
	return aslice, nil
}
