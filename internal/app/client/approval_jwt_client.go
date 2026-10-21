package client

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"gitlab.com/fami-app-v2/familib/familabs/env"
)

// approvalJWTRole is fixed by product decision: every approval-link login
// gets a token with role "Site Dev", whatever the approver's own role is.
const approvalJWTRole = "Site Dev"

// IApprovalJWTClient asks the fami-module auth service for a token for a
// user (POST <approval_jwt.base_url>/fmi-web/v1/auth/jwt). Site Dev never
// signs tokens itself.
//
// NOTE (plan Q1b): that service issues today's token shape (nik, role,
// username). The boilerplate Authorized() used by this service reads
// familabs claims (user_id, expire_at), so such a token is refused until
// the token question is answered.
type IApprovalJWTClient interface {
	IssueJWT(ctx context.Context, username string, nik string) (token string, err error)
}

type ApprovalJWTClient struct {
	baseURL string
	// authHeader is Basic auth with the [public_api] username/password
	authHeader string
	http       *http.Client
}

// NewApprovalJWTClient reads approval_jwt.base_url and
// public_api.username/password from the config (values in devops' secret
// store, never in the repo).
func NewApprovalJWTClient(env env.Provider) IApprovalJWTClient {
	creds := base64.StdEncoding.EncodeToString([]byte(env.GetString("public_api.username") + ":" + env.GetString("public_api.password")))
	return &ApprovalJWTClient{
		baseURL:    env.GetString("approval_jwt.base_url"),
		authHeader: "Basic " + creds,
		http:       &http.Client{Timeout: 15 * time.Second},
	}
}

type approvalJWTRequest struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	NIK      string `json:"nik"`
}

// approvalJWTResponse: {"code":200,"status":"success","data":{"token":"...","expires_at":"..."}}
type approvalJWTResponse struct {
	Data struct {
		Token string `json:"token"`
	} `json:"data"`
}

func (c *ApprovalJWTClient) IssueJWT(ctx context.Context, username string, nik string) (token string, err error) {
	payload, err := json.Marshal(approvalJWTRequest{Username: username, Role: approvalJWTRole, NIK: nik})
	if err != nil {
		return "", fmt.Errorf("approval_jwt: failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/fmi-web/v1/auth/jwt", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("approval_jwt: failed to build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.authHeader)

	res, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("approval_jwt: request failed: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("approval_jwt: failed to read response: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("approval_jwt: unexpected status %d: %s", res.StatusCode, string(body))
	}

	var parsed approvalJWTResponse
	if err = json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("approval_jwt: failed to parse response: %w", err)
	}
	if parsed.Data.Token == "" {
		return "", fmt.Errorf("approval_jwt: response had no token: %s", string(body))
	}
	return parsed.Data.Token, nil
}
