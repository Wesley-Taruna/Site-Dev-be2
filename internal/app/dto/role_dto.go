package dto

// RoleResponseGet is one row of GET /v1/roles (site-dev-api sent the
// Roles model itself: role_id, role_name, role_level).
type RoleResponseGet struct {
	RoleID    string `json:"role_id"`
	RoleName  string `json:"role_name"`
	RoleLevel int    `json:"role_level"`
}
