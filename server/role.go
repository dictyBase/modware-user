package server

import (
	"github.com/dictyBase/go-genproto/dictybaseapis/user"
	dat "gopkg.in/mgutz/dat.v1"
)

type dbRole struct {
	AuthRoleId  string       `db:"auth_role_id"`
	Role        string       `db:"role"`
	Description string       `db:"description"`
	CreatedAt   dat.NullTime `db:"created_at"`
	UpdatedAt   dat.NullTime `db:"updated_at"`
}

type RoleService struct {
	Dbh        *runner.DB
	pathPrefix string
	include    []string
	fields     []string
	resource   string
	baseURL    string
}

func NewRoleService(dbh *runner.DB, pathPrefix string, baseURL string) *RoleService {
	return &RoleService{
		baseURL:    baseURL,
		resource:   "roles",
		Dbh:        dbh,
		pathPrefix: pathPrefix,
		include:    []string{"users", "permissions"},
		fields:     []string{"role", "description"},
	}
}

func (s *RoleService) AllowedInclude() []string {
	return s.include
}

func (s *RoleService) AllowedFields() []string {
	return s.fields
}

func (s *RoleService) GetResourceName() string {
	return s.resource
}

func (s *RoleService) GetBaseURL() string {
	return s.baseURL
}

func (s *RoleService) GetPathPrefix() string {
	return s.pathPrefix
}

func mapRoleAttributes(drole *dbRole) *user.RoleAttributes {
	return &user.RoleAttributes{
		Role:        drole.Role,
		Description: drole.Description,
		CreatedAt:   drole.CreatedAt,
		UpdatedAt:   drole.UpdatedAt,
	}
}
