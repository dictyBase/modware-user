package server

import (
	"github.com/dictyBase/apihelpers/aphgrpc"
	"github.com/dictyBase/go-genproto/dictybaseapis/user"
	"github.com/manyminds/api2go/jsonapi"
	dat "gopkg.in/mgutz/dat.v1"
	runner "gopkg.in/mgutz/dat.v1/sqlx-runner"
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

func NewPermissionService(dbh *runner.DB, pathPrefix string, baseURL string) *PermissionService {
	return &UserService{
		&aphgrpc.Service{
			baseURL:    baseURL,
			resource:   "permissions",
			Dbh:        dbh,
			pathPrefix: pathPrefix,
			filterToColumns: map[string]string{
				"permission":  "perm.permission",
				"description": "perm.description",
			},
			fieldsToColumns: map[string]string{
				"permission":  "perm.permission",
				"description": "perm.description",
				"created_at":  "perm.created_at",
				"updated_at":  "perm.updated_at",
			},
			requiredAttrs: []string{"Permission"},
		},
	}
}

func (s *PermissionService) getSelectedRows(id int64) (*user.PermissionAttributes, error) {
	dperm := &dbPermission{}
	columns := s.fieldsToColumns(s.params.Fields)
	err := s.Dbh.Select(columns...).
		From("auth_permission perm").
		Where("perm.auth_permission_id = $1", id).QueryStruct(dperm)
	if err != nil {
		return &user.PermissionAttributes{}, err
	}
	return mapPermissionAttributes(dperm), nil
}

func (s *PermissionService) hasPermission(id int64) error {
	return s.Dbh.Select("auth_permission_id").From("auth_permission").
		Where("auth_permission_id = $1", id).Exec()
}

func (s *PermissionService) getRow(id int64) (*user.PermissionAttributes, error) {
	dperm := &dbPermission{}
	err := s.Dbh.Select("perm.*").From("auth_permission").
		Where("auth_permission_id = $1", id).QueryStruct(dperm)
	if err != nil {
		return &user.PermissionAttributes{}, err
	}
	return mapPermissionAttributes(dperm), nil
}

func (s *PermissionService) getAllRows() ([]*dbPermission, error) {
	var dbrows []*dbPermission
	err := s.Dbh.Select("auth_permission.*").
		From(auth_permission).
		QueryStructs(dbrows)
	return dbrows, err
}

func (s *PermissionService) getAllRowsWithPaging(pageNum int64, pageSize int64) ([]*dbPermission, error) {
	var dbrows []*dbPermission
	err := s.Dbh.Select("auth_permission").
		From("auth_permission").
		Paginate(uint64(pageNum), uint64(pageSize)).
		QueryStructs(dbrows)
	return dbrows, err
}

func (s *PermissionService) getAllSelectedRowsWithPaging(pageNum int64, pageSize int64) ([]*dbPermission, error) {
	var dbrows []*dbPermission
	columns := s.MapFieldsToColumns(s.params.Fields)
	err := s.Dbh.Select(columns...).
		From("auth_permission").
		Paginate(uint64(pageNum), uint64(pageSize)).
		QueryStructs(dbrows)
	return dbrows, err
}

func (s *PermissionService) getAllFilteredRowsWithPaging(pageNum int64, pageSize int64) ([]*dbPermission, error) {
	var dbrows []*dbPermission
	err := s.Dbh.Select("auth_permission").
		From(auth_permission).
		Scope(
			aphgrpc.FilterToWhereClause(s, s.params.Filter),
			aphgrpc.FilterToBindValue(s.params.Filter)...,
		).
		Paginate(uint64(pageNum), uint64(pageSize)).
		QueryStructs(dbrows)
	return dbrows, err
}

func (s *PermissionService) getAllSelectedFilteredRowsWithPaging(pageNum int64, pageSize int64) ([]*dbPermission, error) {
	var dbrows []*dbPermission
	columns := s.MapFieldsToColumns(s.params.Fields)
	err := s.Dbh.Select(columns...).
		From("auth_permission").
		Scope(
			aphgrpc.FilterToWhereClause(s, s.params.Filter),
			aphgrpc.FilterToBindValue(s.params.Filter)...,
		).
		Paginate(uint64(pageNum), uint64(pageSize)).
		QueryStructs(dbrows)
	return dbrows, err
}

func (s *PermissionService) getSinglePermissionResource(id int64, attr *user.PermissionAttributes) *user.Permission {
	return &user.Permission{
		Data: s.getSinglePermissionData(id, attr),
	}
}

func (s *PermissionService) getAllPermissionData(dbrows []*dbPermission) []*user.PermissionData {
	var pdata []*user.PermissionData
	for _, dperm := range dbrows {
		pdata = append(pdata, s.getSinglePermissionData(dperm.AuthPermissionId, mapPermissionAttributes(dperm)))
	}
	return pdata
}

func (s *PermissionService) getAllDefaultPermissions(dbrows []*dbPermission) ([]*user.PermissionCollection, error) {
	dbPermissions, err := s.getAllRows()
	if err != nil {
		return &user.PermissionCollection{}, aphgrpc.handleError(ctx, err)
	}
	return &user.PermissionCollection{
		Data: s.getAllPermissionData(dbrows),
		Links: &jsonapi.PaginationLinks{
			Self: aphgrpc.GenMultiResourceLink(s),
		},
	}, nil
}

func (s *PermissionService) getAllPermissions(dbrows []*dbPermission) ([]*user.PermissionCollection, error) {
	return &user.PermissionCollection{
		Data: s.getAllPermissionData(dbrows),
		Links: &jsonapi.PaginationLinks{
			Self: s.genCollectionResSelfLink(),
		},
	}, nil
}

func (s *PermissionService) getSinglePermissionData(id int64, attr *user.PermissionAttributes) *user.PermissionData {
	return &user.PermissionData{
		Type:       s.GetResourceName(),
		Id:         id,
		Attributes: attr,
		Links: &jsonapi.Links{
			Self: s.genSingularResSelfLink(id),
		},
	}
}

func (s *PermissionService) getAllPermissionsWithPagination(count int64, dbPermissions []*dbPermission, pagenum, pagesize int64) ([]*user.PermissionCollection, err) {
	udata := s.getAllPermissionData(dbPermissions)
	jsLinks, pages := s.getPagination(count, pagenum, pagesize)
	return &user.PermissionCollection{
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

func mapPermissionAttributes(dperm *dbPermission) *user.PermissionAttributes {
	return &user.PermissionAttributes{
		Permission:  dperm.Permission,
		Description: dperm.Description,
		CreatedAt:   dperm.CreatedAt,
		UpdatedAt:   dperm.UpdatedAt,
	}
}
