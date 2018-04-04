package server

import (
	"context"
	"fmt"

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
	permDbAlias = "auth_permission perm"
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

func permissionServiceOptions() *aphgrpc.ServiceOptions {
	return &aphgrpc.ServiceOptions{
		Resource:   "permissions",
		PathPrefix: "permissions",
		FilToColumns: map[string]string{
			"permission":  fmt.Sprintf("%s.permission", permDbTable),
			"description": fmt.Sprintf("%s.description", permDbTable),
		},
		FieldsToColumns: map[string]string{
			"permission":  fmt.Sprintf("%s.permission", permDbTable),
			"description": fmt.Sprintf("%s.description", permDbTable),
			"created_at":  fmt.Sprintf("%s.created_at", permDbTable),
			"updated_at":  fmt.Sprintf("%s.updated_at", permDbTable),
		},
		ReqAttrs: []string{"Permission"},
	}
}

func NewPermissionService(dbh *runner.DB, opt ...aphgrpc.Option) *PermissionService {
	so := permissionServiceOptions()
	for _, optfn := range opt {
		optfn(so)
	}
	srv := &aphgrpc.Service{Dbh: dbh}
	aphgrpc.AssignFieldsToStructs(so, srv)
	return &PermissionService{srv}
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
	gctx := aphgrpc.GetReqCtx(params, getReq)
	switch {
	case params.HasFields:
		perm, err := s.getResourceWithSelectedAttr(gctx, r.Id)
		if err != nil {
			return &user.Permission{}, aphgrpc.HandleError(ctx, err)
		}
		return perm, nil
	default:
		perm, err := s.getResource(gctx, r.Id)
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
	lctx := aphgrpc.ListReqCtx(
		params,
		&jsonapi.ListRequest{
			Fields:  r.Fields,
			Filter:  r.Filter,
			Include: r.Include,
		})
	// request without any pagination query parameters
	switch {
	case params.HasFields && params.HasFilter:
		dbrows, err := s.getAllSelectedFilteredRows(lctx)
		if err != nil {
			return &user.PermissionCollection{}, aphgrpc.HandleError(ctx, err)
		}
		return s.dbToCollResource(lctx, dbrows), nil
	case params.HasFields:
		dbrows, err := s.getAllSelectedRows(lctx)
		if err != nil {
			return &user.PermissionCollection{}, aphgrpc.HandleError(ctx, err)
		}
		return s.dbToCollResource(lctx, dbrows), nil
	case params.HasFilter:
		dbrows, err := s.getAllFilteredRows(lctx)
		if err != nil {
			return &user.PermissionCollection{}, aphgrpc.HandleError(ctx, err)
		}
		return s.dbToCollResource(lctx, dbrows), nil
	default:
		dbrows, err := s.getAllRows(lctx)
		if err != nil {
			return &user.PermissionCollection{}, aphgrpc.HandleError(ctx, err)
		}
		return s.dbToCollResource(lctx, dbrows), nil
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
	grpc.SetTrailer(ctx, metadata.Pairs("method", "POST"))
	return s.buildResource(
		context.TODO(),
		aphgrpc.NullToInt64(newdbPerm.AuthPermissionId),
		s.dbToResourceAttributes(newdbPerm),
	), nil
}

func (s *PermissionService) UpdatePermission(ctx context.Context, r *user.UpdatePermissionRequest) (*user.Permission, error) {
	result, err := s.existsResource(r.Id)
	if err != nil {
		return &user.Permission{}, aphgrpc.HandleError(ctx, err)
	}
	if !result {
		grpc.SetTrailer(ctx, aphgrpc.ErrNotFound)
		return &user.Permission{}, status.Error(codes.NotFound, fmt.Sprintf("id %d not found", r.Id))
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
	return s.buildResource(context.TODO(), r.Data.Id, r.Data.Attributes), nil
}

func (s *PermissionService) DeletePermission(ctx context.Context, r *jsonapi.DeleteRequest) (*empty.Empty, error) {
	result, err := s.existsResource(r.Id)
	if err != nil {
		return &empty.Empty{}, aphgrpc.HandleError(ctx, err)
	}
	if !result {
		grpc.SetTrailer(ctx, aphgrpc.ErrNotFound)
		return &empty.Empty{}, status.Error(codes.NotFound, fmt.Sprintf("id %d not found", r.Id))
	}
	_, err = s.Dbh.DeleteFrom("auth_permission").Where("auth_permission_id = $1", r.Id).Exec()
	if err != nil {
		grpc.SetTrailer(ctx, aphgrpc.ErrDatabaseDelete)
		return &empty.Empty{}, status.Error(codes.Internal, err.Error())
	}
	return &empty.Empty{}, nil
}

// All helper functions

func (s *PermissionService) existsResource(id int64) (bool, error) {
	r, err := s.Dbh.Select("auth_permission_id").From("auth_permission").
		Where("auth_permission_id = $1", id).Exec()
	if err != nil {
		return false, err
	}
	if r.RowsAffected != 1 {
		return false, nil
	}
	return true, nil
}

func (s *PermissionService) getResourceWithSelectedAttr(ctx context.Context, id int64) (*user.Permission, error) {
	dperm := &dbPermission{}
	params, ok := ctx.Value(aphgrpc.ContextKeyParams).(*aphgrpc.JSONAPIParams)
	if !ok {
		return &user.Permission{}, fmt.Errorf("no params object found in context")
	}
	columns := s.MapFieldsToColumns(params.Fields)
	err := s.Dbh.Select(columns...).
		From(permDbTable).
		Where("auth_permission_id = $1", id).QueryStruct(dperm)
	if err != nil {
		return &user.Permission{}, err
	}
	return s.buildResource(ctx, id, s.dbToResourceAttributes(dperm)), nil
}

func (s *PermissionService) getResource(ctx context.Context, id int64) (*user.Permission, error) {
	dperm := &dbPermission{}
	err := s.Dbh.Select(fmt.Sprintf("%s.*", permDbTable)).From(permDbTable).
		Where("auth_permission_id = $1", id).QueryStruct(dperm)
	if err != nil {
		return &user.Permission{}, err
	}
	return s.buildResource(ctx, id, s.dbToResourceAttributes(dperm)), nil
}

func (s *PermissionService) getAllRows(ctx context.Context) ([]*dbPermission, error) {
	var dbrows []*dbPermission
	err := s.Dbh.Select(fmt.Sprintf("%s.*", permDbTable)).
		From(permDbTable).
		QueryStructs(&dbrows)
	return dbrows, err
}

func (s *PermissionService) getAllSelectedRows(ctx context.Context) ([]*dbPermission, error) {
	var dbrows []*dbPermission
	params, ok := ctx.Value(aphgrpc.ContextKeyParams).(*aphgrpc.JSONAPIParams)
	if !ok {
		return dbrows, fmt.Errorf("no params object found in context")
	}
	columns := s.MapFieldsToColumns(params.Fields)
	err := s.Dbh.Select(columns...).
		From(permDbTable).
		QueryStructs(&dbrows)
	return dbrows, err
}

func (s *PermissionService) getAllFilteredRows(ctx context.Context) ([]*dbPermission, error) {
	var dbrows []*dbPermission
	params, ok := ctx.Value(aphgrpc.ContextKeyParams).(*aphgrpc.JSONAPIParams)
	if !ok {
		return dbrows, fmt.Errorf("no params object found in context")
	}
	err := s.Dbh.Select(fmt.Sprintf("%s.*", permDbTable)).
		From(permDbTable).
		Scope(
			aphgrpc.FilterToWhereClause(s, params.Filters),
			aphgrpc.FilterToBindValue(params.Filters)...,
		).
		QueryStructs(&dbrows)
	return dbrows, err
}

func (s *PermissionService) getAllSelectedFilteredRows(ctx context.Context) ([]*dbPermission, error) {
	var dbrows []*dbPermission
	params, ok := ctx.Value(aphgrpc.ContextKeyParams).(*aphgrpc.JSONAPIParams)
	if !ok {
		return dbrows, fmt.Errorf("no params object found in context")
	}
	columns := s.MapFieldsToColumns(params.Fields)
	err := s.Dbh.Select(columns...).
		From(permDbTable).
		Scope(
			aphgrpc.FilterToWhereClause(s, params.Filters),
			aphgrpc.FilterToBindValue(params.Filters)...,
		).
		QueryStructs(&dbrows)
	return dbrows, err
}

func (s *PermissionService) buildResourceData(ctx context.Context, id int64, attr *user.PermissionAttributes) *user.PermissionData {
	return &user.PermissionData{
		Type:       s.GetResourceName(),
		Id:         id,
		Attributes: attr,
		Links: &jsonapi.Links{
			Self: s.GenResourceSelfLink(ctx, id),
		},
	}
}

func (s *PermissionService) buildResource(ctx context.Context, id int64, attr *user.PermissionAttributes) *user.Permission {
	return &user.Permission{
		Data: s.buildResourceData(ctx, id, attr),
		Links: &jsonapi.Links{
			Self: s.GenResourceSelfLink(ctx, id),
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

func (s *PermissionService) dbToCollResourceData(ctx context.Context, dbrows []*dbPermission) []*user.PermissionData {
	var pdata []*user.PermissionData
	for _, dperm := range dbrows {
		pdata = append(
			pdata,
			s.buildResourceData(
				ctx,
				aphgrpc.NullToInt64(dperm.AuthPermissionId),
				s.dbToResourceAttributes(dperm),
			))
	}
	return pdata
}

func (s *PermissionService) dbToCollResource(ctx context.Context, dbrows []*dbPermission) *user.PermissionCollection {
	return &user.PermissionCollection{
		Data: s.dbToCollResourceData(ctx, dbrows),
		Links: &jsonapi.Links{
			Self: s.GenCollResourceSelfLink(ctx),
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
