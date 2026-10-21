package dto

import "time"

// UserRequestCreate is the body of POST /v1/users. UserID is the NIK; login
// itself stays with the identity provider that issues the token.
type UserRequestCreate struct {
	UserID   int64  `json:"user_id" valid:"required"`
	Username string `json:"username" valid:"required"`
	Email    string `json:"email" valid:"required|email"`
	RoleID   string `json:"role_id" valid:"required"`
}

// UserRequestUpdate is the body of PUT /v1/users/:id; empty fields are
// left unchanged.
type UserRequestUpdate struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	RoleID   string `json:"role_id"`
}

// UserResponseGet is one user (site-dev-api UserResp), with today's AI
// insight and geocoding quota.
type UserResponseGet struct {
	UserID             int64     `json:"user_id"`
	Username           string    `json:"username"`
	Email              string    `json:"email"`
	RoleID             string    `json:"role_id"`
	RoleName           string    `json:"role_name,omitempty"`
	AIInsightRemaining int64     `json:"ai_insight_remaining"`
	AIInsightLimit     int64     `json:"ai_insight_limit"`
	GeocodingRemaining int64     `json:"geocoding_remaining"`
	GeocodingLimit     int64     `json:"geocoding_limit"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
