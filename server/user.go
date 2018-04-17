package server

import (
	"context"
	"fmt"
	"strings"

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
	usrTableSel = `
			SELECT
				auth_user.auth_user_id,
				CAST(auth_user.email AS TEXT),
				auth_user.first_name,
				auth_user.last_name,
				auth_user.is_active,
				auth_user_info.*
			FROM auth_user
		`
	usrTablesJoin = `
			JOIN auth_user_info
			ON auth_user.auth_user_id = auth_user_info.auth_user_id
	`
	userDbTable = "auth_user"
)

var usrTableStmt = fmt.Sprintf("%s %s", usrTableSel, usrTablesJoin)

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
	AuthUserId     int64          `db:"auth_user_id"`
	FirstName      string         `db:"first_name"`
	LastName       string         `db:"last_name"`
	Email          string         `db:"email"`
	IsActive       bool           `db:"is_active"`
	Organization   dat.NullString `db:"organization"`
	GroupName      dat.NullString `db:"group_name"`
	FirstAddress   dat.NullString `db:"first_address"`
	SecondAddress  dat.NullString `db:"second_address"`
	City           dat.NullString `db:"city"`
	State          dat.NullString `db:"state"`
	Zipcode        dat.NullString `db:"zipcode"`
	Country        dat.NullString `db:"country"`
	Phone          dat.NullString `db:"phone"`
	CreatedAt      dat.NullTime   `db:"created_at"`
	UpdatedAt      dat.NullTime   `db:"updated_at"`
	AuthUserInfoId int64          `db:"auth_user_info_id"`
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

func userServiceOptions() *aphgrpc.ServiceOptions {
	return &aphgrpc.ServiceOptions{
		Resource:   "users",
		PathPrefix: "users",
		Include:    []string{"roles"},
		FilToColumns: map[string]string{
			"first_name": "auth_user.first_name",
			"last_name":  "auth_user.last_name",
			"email":      "auth_user.email",
		},
		FieldsToColumns: map[string]string{
			"first_name":     "auth_user.first_name",
			"last_name":      "auth_user.last_name",
			"email":          "auth_user.email",
			"created_at":     "auth_user.created_at",
			"updated_at":     "auth_user.updated_at",
			"organization":   "auth_user_info.organization",
			"group_name":     "auth_user_info.group_name",
			"first_address":  "auth_user_info.first_address",
			"second_address": "auth_user_info.second_address",
			"city":           "auth_user_info.city",
			"state":          "auth_user_info.state",
			"zipcode":        "auth_user_info.zipcode",
			"country":        "auth_user_info.country",
			"phone":          "auth_user_info.phone",
			"is_active":      "auth_user_info.is_active",
		},
		ReqAttrs: []string{"FirstName", "LastName", "Email"},
	}
}

func NewUserService(dbh *runner.DB, opt ...aphgrpc.Option) *UserService {
	so := userServiceOptions()
	for _, optfn := range opt {
		optfn(so)
	}
	srv := &aphgrpc.Service{Dbh: dbh}
	aphgrpc.AssignFieldsToStructs(so, srv)
	return &UserService{srv}
}

func (s *UserService) Healthz(ctx context.Context, r *jsonapi.HealthzIdRequest) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}

func (s *UserService) ExistUser(ctx context.Context, r *jsonapi.IdRequest) (*jsonapi.ExistResponse, error) {
	found, err := s.existsResource(r.Id)
	return &jsonapi.ExistResponse{Exist: found}, err
}

func (s *UserService) GetUserByEmail(ctx context.Context, r *jsonapi.GetEmailRequest) (*user.User, error) {
	id, err := s.emailToResourceId(r.Email)
	if err != nil {
		return &user.User{}, aphgrpc.HandleError(ctx, err)
	}
	if id == 0 { // user with that email does not exist
		return &user.User{}, status.Error(codes.NotFound, fmt.Sprintf("email %s not found", r.Email))
	}
	return s.GetUser(
		ctx,
		&jsonapi.GetRequest{
			Id:      id,
			Include: r.Include,
			Fields:  r.Fields,
		})
}

func (s *UserService) GetUser(ctx context.Context, r *jsonapi.GetRequest) (*user.User, error) {
	params, md, err := aphgrpc.ValidateAndParseGetParams(s, r)
	if err != nil {
		grpc.SetTrailer(ctx, md)
		return new(user.User), status.Error(codes.InvalidArgument, err.Error())
	}
	gctx := aphgrpc.GetReqCtx(params, r)
	switch {
	case params.HasFields && params.HasInclude:
		u, err := s.getResourceWithSelectedAttr(gctx, r.Id)
		if err != nil {
			return &user.User{}, aphgrpc.HandleError(ctx, err)
		}
		err = s.buildResourceRelationships(r.Id, u)
		if err != nil {
			return &user.User{}, aphgrpc.HandleError(ctx, err)
		}
		return u, nil
	case params.HasFields:
		u, err := s.getResourceWithSelectedAttr(gctx, r.Id)
		if err != nil {
			return &user.User{}, aphgrpc.HandleError(ctx, err)
		}
		return u, nil
	case params.HasInclude:
		u, err := s.getResource(gctx, r.Id)
		if err != nil {
			return &user.User{}, aphgrpc.HandleError(ctx, err)
		}
		err = s.buildResourceRelationships(r.Id, u)
		if err != nil {
			return &user.User{}, aphgrpc.HandleError(ctx, err)
		}
		return u, nil
	default:
		u, err := s.getResource(gctx, r.Id)
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
	lctx := aphgrpc.ListReqCtx(params, r)
	// has pagination query parameters
	if aphgrpc.HasPagination(r) {
		if r.Pagenum == 0 {
			r.Pagenum = aphgrpc.DefaultPagenum
		}
		if r.Pagesize == 0 {
			r.Pagesize = aphgrpc.DefaultPagesize
		}
		switch {
		// filter, fields and include parameters
		case params.HasFields && params.HasInclude && params.HasFilter:
			count, err := s.GetAllFilteredCount(lctx, fmt.Sprintf("%s %s", userDbTable, usrTablesJoin))
			if err != nil {
				return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
			}
			dbUsers, err := s.getAllSelectedFilteredRowsWithPaging(lctx, r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
			}
			return s.dbToCollResourceWithRelAndPagination(lctx, count, dbUsers, r.Pagenum, r.Pagesize)
		// fields and includes
		case params.HasFields && params.HasInclude:
			count, err := s.GetCount(lctx, userDbTable)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
			}
			dbUsers, err := s.getAllSelectedRowsWithPaging(lctx, r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
			}
			return s.dbToCollResourceWithRelAndPagination(lctx, count, dbUsers, r.Pagenum, r.Pagesize)
		// fields and filters
		case params.HasFields && params.HasFilter:
			count, err := s.GetAllFilteredCount(lctx, fmt.Sprintf("%s %s", userDbTable, usrTablesJoin))
			if err != nil {
				return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
			}
			dbUsers, err := s.getAllSelectedFilteredRowsWithPaging(lctx, r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
			}
			return s.dbToCollResourceWithPagination(lctx, count, dbUsers, r.Pagenum, r.Pagesize), nil
		// include and filter
		case params.HasInclude && params.HasFilter:
			count, err := s.GetAllFilteredCount(lctx, fmt.Sprintf("%s %s", userDbTable, usrTablesJoin))
			if err != nil {
				return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
			}
			dbUsers, err := s.getAllFilteredRowsWithPaging(lctx, r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
			}
			return s.dbToCollResourceWithRelAndPagination(lctx, count, dbUsers, r.Pagenum, r.Pagesize)
		case params.HasFields:
			count, err := s.GetCount(lctx, userDbTable)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
			}
			dbUsers, err := s.getAllSelectedRowsWithPaging(lctx, r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
			}
			return s.dbToCollResourceWithPagination(lctx, count, dbUsers, r.Pagenum, r.Pagesize), nil
		case params.HasFilter:
			count, err := s.GetAllFilteredCount(lctx, fmt.Sprintf("%s %s", userDbTable, usrTablesJoin))
			if err != nil {
				return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
			}
			dbUsers, err := s.getAllFilteredRowsWithPaging(lctx, r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
			}
			return s.dbToCollResourceWithPagination(lctx, count, dbUsers, r.Pagenum, r.Pagesize), nil
		case params.HasInclude:
			count, err := s.GetCount(lctx, userDbTable)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
			}
			dbUsers, err := s.getAllRowsWithPaging(lctx, r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
			}
			return s.dbToCollResourceWithRelAndPagination(lctx, count, dbUsers, r.Pagenum, r.Pagesize)
		// only pagination
		default:
			count, err := s.GetCount(lctx, userDbTable)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
			}
			dbUsers, err := s.getAllSelectedRowsWithPaging(lctx, r.Pagenum, r.Pagesize)
			if err != nil {
				return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
			}
			return s.dbToCollResourceWithPagination(lctx, count, dbUsers, r.Pagenum, r.Pagesize), nil
		}
	}
	// request without any pagination query parameters
	switch {
	case params.HasFields && params.HasFilter && params.HasInclude:
		count, err := s.GetAllFilteredCount(lctx, fmt.Sprintf("%s %s", userDbTable, usrTablesJoin))
		if err != nil {
			return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
		}
		dbUsers, err := s.getAllSelectedFilteredRowsWithPaging(lctx, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			return s.dbToCollResourceWithRelAndPagination(lctx, count, dbUsers, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		}
		return s.dbToCollResource(lctx, dbUsers), nil
	case params.HasFields && params.HasFilter:
		count, err := s.GetAllFilteredCount(lctx, fmt.Sprintf("%s %s", userDbTable, usrTablesJoin))
		if err != nil {
			return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
		}
		dbUsers, err := s.getAllSelectedFilteredRowsWithPaging(lctx, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			return s.dbToCollResourceWithPagination(lctx, count, dbUsers, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize), nil
		}
		return s.dbToCollResource(lctx, dbUsers), nil
	case params.HasFields && params.HasInclude:
		count, err := s.GetCount(lctx, "auth_user")
		if err != nil {
			return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
		}
		dbUsers, err := s.getAllSelectedRowsWithPaging(lctx, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			return s.dbToCollResourceWithRelAndPagination(lctx, count, dbUsers, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		}
		return s.dbToCollResource(lctx, dbUsers), nil
	case params.HasFilter && params.HasInclude:
		count, err := s.GetAllFilteredCount(lctx, fmt.Sprintf("%s %s", userDbTable, usrTablesJoin))
		if err != nil {
			return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
		}
		dbUsers, err := s.getAllFilteredRowsWithPaging(lctx, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			return s.dbToCollResourceWithRelAndPagination(lctx, count, dbUsers, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		}
		return s.dbToCollResource(lctx, dbUsers), nil
	case params.HasFields:
		count, err := s.GetCount(lctx, userDbTable)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
		}
		dbUsers, err := s.getAllSelectedRowsWithPaging(lctx, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			return s.dbToCollResourceWithPagination(lctx, count, dbUsers, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize), nil
		}
		return s.dbToCollResource(lctx, dbUsers), nil
	case params.HasFilter:
		count, err := s.GetAllFilteredCount(lctx, fmt.Sprintf("%s %s", userDbTable, usrTablesJoin))
		if err != nil {
			return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
		}
		dbUsers, err := s.getAllFilteredRowsWithPaging(lctx, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			return s.dbToCollResourceWithPagination(lctx, count, dbUsers, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize), nil
		}
		return s.dbToCollResource(lctx, dbUsers), nil
	case params.HasInclude:
		count, err := s.GetCount(lctx, userDbTable)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
		}
		dbUsers, err := s.getAllRowsWithPaging(lctx, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			return s.dbToCollResourceWithRelAndPagination(lctx, count, dbUsers, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		}
		return s.dbToCollResource(lctx, dbUsers), nil
	default:
		count, err := s.GetCount(lctx, userDbTable)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
		}
		dbUsers, err := s.getAllRowsWithPaging(lctx, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize)
		if err != nil {
			return &user.UserCollection{}, aphgrpc.HandleError(ctx, err)
		}
		if count > aphgrpc.DefaultPagesize {
			return s.dbToCollResourceWithPagination(lctx, count, dbUsers, aphgrpc.DefaultPagenum, aphgrpc.DefaultPagesize), nil
		}
		return s.dbToCollResource(lctx, dbUsers), nil
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
	grpc.SetTrailer(ctx, metadata.Pairs("method", "POST"))
	return s.buildResource(
		context.TODO(),
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
	return s.buildResource(
		context.TODO(),
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

func (s *UserService) emailToResourceId(email string) (int64, error) {
	var id int64
	err := s.Dbh.Select("auth_user_id").From(userDbTable).
		Where("email = $1", email).QueryScalar(&id)
	return id, err
}

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

func (s *UserService) getResourceWithSelectedAttr(ctx context.Context, id int64) (*user.User, error) {
	params, ok := ctx.Value(aphgrpc.ContextKeyParams).(*aphgrpc.JSONAPIParams)
	if !ok {
		return &user.User{}, fmt.Errorf("no params object found in context")
	}
	dusr := new(dbUser)
	err := s.Dbh.SQL(
		fmt.Sprintf(
			"SELECT %s FROM auth_user %s %s",
			strings.Join(s.mapFieldsToColumnsWithCast(params.Fields), ","),
			usrTablesJoin,
			"WHERE auth_user.auth_user_id = $1",
		), id).QueryStruct(dusr)
	if err != nil {
		return &user.User{}, err
	}
	return s.buildResource(ctx, id, s.dbToResourceAttributes(dusr)), nil
}

func (s *UserService) getResource(ctx context.Context, id int64) (*user.User, error) {
	dusr := new(dbUser)
	err := s.Dbh.SQL(
		fmt.Sprintf(
			"%s %s",
			usrTableStmt,
			"WHERE auth_user.auth_user_id = $1",
		), id).QueryStruct(dusr)
	if err != nil {
		return &user.User{}, err
	}
	return s.buildResource(ctx, id, s.dbToResourceAttributes(dusr)), nil
}

// -- Functions that queries the storage and generates a database user object

func (s *UserService) getAllRows(ctx context.Context) ([]*dbUser, error) {
	var dusrRows []*dbUser
	err := s.Dbh.SQL(usrTableStmt).QueryStructs(&dusrRows)
	return dusrRows, err
}

func (s *UserService) getAllRowsWithPaging(ctx context.Context, pagenum int64, pagesize int64) ([]*dbUser, error) {
	var dusrRows []*dbUser
	err := s.Dbh.SQL(
		fmt.Sprintf(
			"%s LIMIT %d OFFSET %d",
			usrTableStmt,
			pagesize,
			(pagenum-1)*pagesize,
		)).
		QueryStructs(&dusrRows)
	return dusrRows, err
}

func (s *UserService) getAllSelectedRowsWithPaging(ctx context.Context, pagenum, pagesize int64) ([]*dbUser, error) {
	var dusrRows []*dbUser
	params, ok := ctx.Value(aphgrpc.ContextKeyParams).(*aphgrpc.JSONAPIParams)
	if !ok {
		return dusrRows, fmt.Errorf("no params object found in context")
	}
	err := s.Dbh.SQL(
		fmt.Sprintf(
			"SELECT %s FROM auth_user %s LIMIT %d OFFSET %d",
			strings.Join(s.mapFieldsToColumnsWithCast(params.Fields), ","),
			usrTablesJoin,
			pagesize,
			(pagenum-1)*pagesize,
		)).
		QueryStructs(&dusrRows)
	return dusrRows, err
}

func (s *UserService) getAllFilteredRowsWithPaging(ctx context.Context, pagenum, pagesize int64) ([]*dbUser, error) {
	var dusrRows []*dbUser
	params, ok := ctx.Value(aphgrpc.ContextKeyParams).(*aphgrpc.JSONAPIParams)
	if !ok {
		return dusrRows, fmt.Errorf("no params object found in context")
	}
	var bindVals []string
	for _, v := range aphgrpc.FilterToBindValue(params.Filters) {
		bindVals = append(bindVals, v.(string))
	}
	err := s.Dbh.SQL(
		fmt.Sprintf(
			"%s %s LIMIT %d OFFSET %d",
			usrTableStmt,
			aphgrpc.FilterToWhereClause(s, params.Filters),
			pagesize,
			(pagenum-1)*pagesize,
		), strings.Join(bindVals, ","),
	).QueryStructs(&dusrRows)
	return dusrRows, err
}

func (s *UserService) getAllSelectedFilteredRowsWithPaging(ctx context.Context, pagenum, pagesize int64) ([]*dbUser, error) {
	var dusrRows []*dbUser
	params, ok := ctx.Value(aphgrpc.ContextKeyParams).(*aphgrpc.JSONAPIParams)
	if !ok {
		return dusrRows, fmt.Errorf("no params object found in context")
	}
	var bindVals []string
	for _, v := range aphgrpc.FilterToBindValue(params.Filters) {
		bindVals = append(bindVals, v.(string))
	}
	err := s.Dbh.SQL(
		fmt.Sprintf(
			"SELECT %s FROM auth_user %s %s LIMIT %d OFFSET %d",
			strings.Join(s.mapFieldsToColumnsWithCast(params.Fields), ","),
			usrTablesJoin,
			aphgrpc.FilterToWhereClause(s, params.Filters),
			pagesize,
			(pagenum-1)*pagesize,
		), strings.Join(bindVals, ","),
	).QueryStructs(dusrRows)
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
		`).Where("auth_user_role.auth_user_id = $1", id).QueryStructs(&drole)
	if err != nil {
		return rdata, err
	}
	return NewRoleService(s.Dbh).dbToCollResourceData(context.TODO(), drole), nil
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

func (s *UserService) buildResourceData(ctx context.Context, id int64, uattr *user.UserAttributes) *user.UserData {
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
			Self: s.GenResourceSelfLink(ctx, id),
		},
	}
}

func (s *UserService) buildResource(ctx context.Context, id int64, uattr *user.UserAttributes) *user.User {
	return &user.User{
		Data: s.buildResourceData(ctx, id, uattr),
		Links: &jsonapi.Links{
			Self: s.GenResourceSelfLink(ctx, id),
		},
	}
}

func (s *UserService) buildResourceRelationships(id int64, user *user.User) error {
	var allInc []*any.Any
	roles, err := s.getRoleResourceData(id)
	if err != nil {
		return err
	}
	incRoles, err := NewRoleService(s.Dbh).convertAllToAny(roles)
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

func (s *UserService) dbToCollResourceData(ctx context.Context, dbUsers []*dbUser) []*user.UserData {
	var udata []*user.UserData
	for _, dusr := range dbUsers {
		udata = append(udata, s.buildResourceData(ctx, dusr.AuthUserId, s.dbToResourceAttributes(dusr)))
	}
	return udata

}

func (s *UserService) dbToCollResource(ctx context.Context, dbUsers []*dbUser) *user.UserCollection {
	return &user.UserCollection{
		Data: s.dbToCollResourceData(ctx, dbUsers),
		Links: &jsonapi.PaginationLinks{
			Self: s.GenCollResourceSelfLink(ctx),
		},
	}
}

func (s *UserService) dbToCollResourceWithPagination(ctx context.Context, count int64, dbUsers []*dbUser, pagenum, pagesize int64) *user.UserCollection {
	udata := s.dbToCollResourceData(ctx, dbUsers)
	jsLinks, pages := s.GetPagination(ctx, count, pagenum, pagesize)
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

func (s *UserService) dbToCollResourceWithRelAndPagination(ctx context.Context, count int64, dbUsers []*dbUser, pagenum, pagesize int64) (*user.UserCollection, error) {
	udata := s.dbToCollResourceData(ctx, dbUsers)
	var allRoles []*user.RoleData
	for i, _ := range udata {
		roles, err := s.getRoleResourceData(dbUsers[i].AuthUserId)
		if err != nil {
			return &user.UserCollection{}, err
		}
		udata[i].Relationships.Roles.Data = s.buildRoleResourceIdentifiers(roles)
		allRoles = append(allRoles, roles...)
	}
	incRoles, err := NewRoleService(s.Dbh).convertAllToAny(allRoles)
	if err != nil {
		return &user.UserCollection{}, err
	}
	jsLinks, pages := s.GetPagination(ctx, count, pagenum, pagesize)
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

func (s *UserService) mapFieldsToColumnsWithCast(fields []string) []string {
	var columns []string
	for _, c := range s.MapFieldsToColumns(fields) {
		if c == "auth_user.email" {
			columns = append(columns, fmt.Sprintf("CAST(%s AS TEXT)", c))
		} else {
			columns = append(columns, c)
		}
	}
	return columns
}
