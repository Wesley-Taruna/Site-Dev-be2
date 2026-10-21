package dto

import "gitlab.com/fami-app-v2/familib/familabs/validation"

// TokenValidateResponse is POST /v1/validation (site-dev-api
// validateTokenResp): who the token belongs to.
type TokenValidateResponse struct {
	UserID   int64  `json:"user_id"`
	NIK      string `json:"nik"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// ApprovalCodeValidateRequest is the body of POST /v1/validation/approval.
type ApprovalCodeValidateRequest struct {
	ApprovalCode string `json:"approval_code" valid:"required"`
}

// Validate implements validation.Request: the code is exactly 32 characters.
func (r ApprovalCodeValidateRequest) Validate() *validation.Output {
	o := &validation.Output{Valid: true}
	if r.ApprovalCode != "" && len(r.ApprovalCode) != 32 {
		o.Failure("approval_code.len", "The approval code must be 32 characters.")
	}
	return o
}

// Messages implements validation.Request.
func (r ApprovalCodeValidateRequest) Messages() map[string]string {
	return map[string]string{}
}

// ApprovalUserProfile is the approver in the approval-link answer.
type ApprovalUserProfile struct {
	UserID   int64  `json:"user_id"`
	NIK      string `json:"nik"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// ApprovalCodeValidateResponse is POST /v1/validation/approval
// (site-dev-api ApprovalCodeValidateResp): token, site to open, approver.
type ApprovalCodeValidateResponse struct {
	JWT         string              `json:"jwt"`
	SiteID      string              `json:"site_id"`
	UserProfile ApprovalUserProfile `json:"user_profile"`
}
