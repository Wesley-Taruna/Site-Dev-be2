package dto

import "time"

// ApprovalMatrixRequestCreate is the body of POST /v1/approval-matrix.
type ApprovalMatrixRequestCreate struct {
	RoleID string `json:"role_id" valid:"required"`
	UserID int64  `json:"user_id" valid:"required"`
}

// ApprovalMatrixRequestUpdate is the body of PUT /v1/approval-matrix/:id;
// user_id 0 and a missing is_active are left unchanged.
type ApprovalMatrixRequestUpdate struct {
	UserID   int64 `json:"user_id"`
	IsActive *bool `json:"is_active"`
}

// ApprovalMatrixResponseGet is one approval matrix row (site-dev-api
// ApprovalMatrixResp).
type ApprovalMatrixResponseGet struct {
	ID        uint      `json:"id"`
	RoleID    string    `json:"role_id"`
	RoleName  string    `json:"role_name,omitempty"`
	RoleLevel int       `json:"role_level,omitempty"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username,omitempty"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
