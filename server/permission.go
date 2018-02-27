package server

import (
	"context"

	"github.com/dictyBase/apihelpers/aphgrpc"
	"github.com/dictyBase/go-genproto/dictybaseapis/api/jsonapi"
	"github.com/dictyBase/go-genproto/dictybaseapis/user"
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
	permDbTable = "auth_permission"
)

var permissionCols = []string{
	"permission",
	"description",
	"created_at",
	"updated_at",
}

type dbPermission struct {
	AuthPermissionId dat.NullInt64  `db:"auth_permission_id"`
	Permission       string         `db:"permission"`
	Description      dat.NullString `db:"description"`
	CreatedAt        dat.NullTime   `db:"created_at"`
	UpdatedAt        dat.NullTime   `db:"updated_at"`
}

type PermissionService struct {
	*aphgrpc.Service
}

func NewPermissionService(dbh *runner.DB, pathPrefix string) *PermissionService {
	return &PermissionService{
		&aphgrpc.Service{
			Resource:   "permissions",
			Dbh:        dbh,
			PathPrefix: pathPrefix,
			FilToColumns: map[string]string{
				"permission":  "perm.permission",
				"description": "perm.description",
			},
			FieldsToColumns: map[string]string{
				"permission":  "perm.permission",
				"description": "perm.description",
				"created_at":  "perm.created_at",
				"updated_at":  "perm.updated_at",
			},
			ReqAttrs: []string{"Permission"},
		},
	}
}

func (s *PermissionService) GetPermission(ctx context.Context, r *jsonapi.GetRequestWithFields) (*user.Permission, error) {
	getReq := &jsonapi.GetRequest{
		Id:     r.Id,
		Fields: r.Fields,
	}
	params, md, err := aphgrpc.ValidateAndParseGetParams(s, getReq)
	if err != nil {
		grpc.SetTrailer(ctx, md)
		return &user.Permission{}, status.Error(codes.InvalidArgument, err.Error())
	}
	s.Params = params
	s.ListMethod = false
	s.SetBaseURL(ctx)
	switch {
	case params.HasFields:
		s.FieldsStr = r.Fields
		perm, err := s.getResourceWithSelectedAttr(r.Id)
		if err != nil {
			return &user.Permission{}, aphgrpc.HandleError(ctx, err)
		}
		return perm, nil
	default:
		perm, err := s.getResource(r.Id)
		if err != nil {
			return &user.Permission{}, aphgrpc.HandleError(ctx, err)
		}
		return perm, nil
	}
}

func (s *PermissionService) ListPermissions(ctx context.Context, r *jsonapi.SimpleListRequest) (*user.PermissionCollection, error) {
	params, md, err := aphgrpc.ValidateAndParseSimpleListParams(s, r)
	if err != nil {
		grpc.SetTrailer(ctx, md)
		return &user.PermissionCollection{}, status.Error(codes.InvalidArgument, err.Error())
	}
	s.Params = params
	s.ListMethod = true
	s.SetBaseURL(ctx)
	// request without any pagination query parameters
	switch {
	case params.HasFields && params.HasFilter:
		s.FieldsStr = r.Fields
		s.FilterStr = r.Filter
		dbrows, err := s.getAllSelectedFilteredRows()
		if err != nil {
			return &user.PermissionCollection{}, aphgrpc.HandleError(ctx, err)
		}
		return s.dbToCollResource(dbrows), nil
	case params.HasFields:
		s.FieldsStr = r.Fields
		dbrows, err := s.getAllSelectedRows()
		if err != nil {
			return &user.PermissionCollection{}, aphgrpc.HandleError(ctx, err)
		}
		return s.dbToCollResource(dbrows), nil
	case params.HasFilter:
		s.FilterStr = r.Filter
		dbrows, err := s.getAllFilteredRows()
		if err != nil {
			return &user.PermissionCollection{}, aphgrpc.HandleError(ctx, err)
		}
		return s.dbToCollResource(dbrows), nil
	default:
		dbrows, err := s.getAllRows()
		if err != nil {
			return &user.PermissionCollection{}, aphgrpc.HandleError(ctx, err)
		}
		return s.dbToCollResource(dbrows), nil
	}
}

func (s *PermissionService) CreatePermission(ctx context.Context, r *user.CreatePermissionRequest) (*user.Permission, error) {
	dbperm := s.attrTodbPermission(r.Data.Attributes)
	pcolumns := aphgrpc.GetDefinedTags(dbperm, "db")
	allcolumns := append(permissionCols, "auth_permission_id")
	newdbPerm := &dbPermission{}
	if len(pcolumns) > 0 {
		err := s.Dbh.InsertInto(permDbTable).
			Columns(pcolumns...).
			Record(dbperm).
			Returning(allcolumns...).
			QueryStruct(newdbPerm)
		if err != nil {
			grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseInsert)
			return &user.Permission{}, status.Error(codes.Internal, err.Error())
		}
	}
	s.SetBaseURL(ctx)
	grpc.SetTrailer(ctx, metadata.Pairs("method", "POST"))
	return s.buildResource(
		aphgrpc.NullToInt64(newdbPerm.AuthPermissionId),
		s.dbToResourceAttributes(newdbPerm),
	), nil
}

func (s *PermissionService) UpdatePermission(ctx context.Context, r *user.UpdatePermissionRequest) (*user.Permission, error) {
	if err := s.existsResource(r.Data.Id); err != nil {
		return &user.Permission{}, aphgrpc.HandleError(ctx, err)
	}
	dbperm := s.attrTodbPermission(r.Data.Attributes)
	permMap := aphgrpc.GetDefinedTagsWithValue(dbperm, "db")
	if len(permMap) > 0 {
		_, err := s.Dbh.Update("auth_permission").SetMap(permMap).
			Where("auth_permission_id = $1", r.Data.Id).Exec()
		if err != nil {
			grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseUpdate)
			return &user.Permission{}, status.Error(codes.Internal, err.Error())
		}
	}
	s.SetBaseURL(ctx)
	return s.buildResource(r.Data.Id, r.Data.Attributes), nil
}

func (s *PermissionService) DeletePermission(ctx context.Context, r *jsonapi.DeleteRequest) (*empty.Empty, error) {
	if err := s.existsResource(r.Id); err != nil {
		return &empty.Empty{}, aphgrpc.HandleError(ctx, err)
	}
	_, err := s.Dbh.DeleteFrom("auth_permission").Where("auth_permission_id = $1", r.Id).Exec()
	if err != nil {
		grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseDelete)
		return &empty.Empty{}, status.Error(codes.Internal, err.Error())
	}
	return &empty.Empty{}, nil
}

// All helper functions

func (s *PermissionService) existsResource(id int64) error {
	_, err := s.Dbh.Select("auth_permission_id").From("auth_permission").
		Where("auth_permission_id = $1", id).Exec()
	return err
}

func (s *PermissionService) getResourceWithSelectedAttr(id int64) (*user.Permission, error) {
	dperm := &dbPermission{}
	columns := s.MapFieldsToColumns(s.Params.Fields)
	err := s.Dbh.Select(columns...).
		From("auth_permission perm").
		Where("perm.auth_permission_id = $1", id).QueryStruct(dperm)
	if err != nil {
		return &user.Permission{}, err
	}
	return s.buildResource(id, s.dbToResourceAttributes(dperm)), nil
}

func (s *PermissionService) getResource(id int64) (*user.Permission, error) {
	dperm := &dbPermission{}
	err := s.Dbh.Select("perm.*").From("auth_permission perm").
		Where("auth_permission_id = $1", id).QueryStruct(dperm)
	if err != nil {
		return &user.Permission{}, err
	}
	return s.buildResource(id, s.dbToResourceAttributes(dperm)), nil
}

func (s *PermissionService) getAllRows() ([]*dbPermission, error) {
	var dbrows []*dbPermission
	err := s.Dbh.Select("auth_permission.*").
		From(permDbTable).
		QueryStructs(&dbrows)
	return dbrows, err
}

func (s *PermissionService) getAllSelectedRows() ([]*dbPermission, error) {
	var dbrows []*dbPermission
	columns := s.MapFieldsToColumns(s.Params.Fields)
	err := s.Dbh.Select(columns...).
		From("auth_permission").
		QueryStructs(&dbrows)
	return dbrows, err
}

func (s *PermissionService) getAllFilteredRows() ([]*dbPermission, error) {
	var dbrows []*dbPermission
	err := s.Dbh.Select("auth_permission.*").
		From(permDbTable).
		Scope(
			aphgrpc.FilterToWhereClause(s, s.Params.Filters),
			aphgrpc.FilterToBindValue(s.Params.Filters)...,
		).
		QueryStructs(&dbrows)
	return dbrows, err
}

func (s *PermissionService) getAllSelectedFilteredRows() ([]*dbPermission, error) {
	var dbrows []*dbPermission
	columns := s.MapFieldsToColumns(s.Params.Fields)
	err := s.Dbh.Select(columns...).
		From("auth_permission").
		Scope(
			aphgrpc.FilterToWhereClause(s, s.Params.Filters),
			aphgrpc.FilterToBindValue(s.Params.Filters)...,
		).
		QueryStructs(&dbrows)
	return dbrows, err
}

func (s *PermissionService) buildResourceData(id int64, attr *user.PermissionAttributes) *user.PermissionData {
	return &user.PermissionData{
		Type:       s.GetResourceName(),
		Id:         id,
		Attributes: attr,
		Links: &jsonapi.Links{
			Self: s.GenResourceSelfLink(id),
		},
	}
}

func (s *PermissionService) buildResource(id int64, attr *user.PermissionAttributes) *user.Permission {
	return &user.Permission{
		Data: s.buildResourceData(id, attr),
		Links: &jsonapi.Links{
			Self: s.GenResourceSelfLink(id),
		},
	}
}

func (s *PermissionService) dbToResourceAttributes(dperm *dbPermission) *user.PermissionAttributes {
	return &user.PermissionAttributes{
		Permission:  dperm.Permission,
		Description: aphgrpc.NullToString(dperm.Description),
		CreatedAt:   aphgrpc.NullToTime(dperm.CreatedAt),
		UpdatedAt:   aphgrpc.NullToTime(dperm.UpdatedAt),
	}
}

func (s *PermissionService) attrTodbPermission(attr *user.PermissionAttributes) *dbPermission {
	return &dbPermission{
		Permission:  attr.Permission,
		Description: dat.NullStringFrom(attr.Description),
		CreatedAt:   dat.NullTimeFrom(aphgrpc.ProtoTimeStamp(attr.CreatedAt)),
		UpdatedAt:   dat.NullTimeFrom(aphgrpc.ProtoTimeStamp(attr.UpdatedAt)),
	}
}

func (s *PermissionService) dbToCollResourceData(dbrows []*dbPermission) []*user.PermissionData {
	var pdata []*user.PermissionData
	for _, dperm := range dbrows {
		pdata = append(pdata, s.buildResourceData(aphgrpc.NullToInt64(dperm.AuthPermissionId), s.dbToResourceAttributes(dperm)))
	}
	return pdata
}

func (s *PermissionService) dbToCollResource(dbrows []*dbPermission) *user.PermissionCollection {
	return &user.PermissionCollection{
		Data: s.dbToCollResourceData(dbrows),
		Links: &jsonapi.Links{
			Self: s.GenCollResourceSelfLink(),
		},
	}
}

func (s *PermissionService) convertAllToAny(perms []*user.PermissionData) ([]*any.Any, error) {
	aslice := make([]*any.Any, len(perms))
	for i, p := range perms {
		pkg, err := ptypes.MarshalAny(p)
		if err != nil {
			return aslice, err
		}
		aslice[i] = pkg
	}
	return aslice, nil
}
