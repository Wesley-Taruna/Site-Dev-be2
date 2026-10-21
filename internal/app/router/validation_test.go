package router

// Side-by-side check, module 5 (login + e-mail link): POST /v1/validation
// and POST /v1/validation/approval against site-dev-api 20eeab4.

import (
	"database/sql/driver"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"site-dev-backend/internal/pkg/sqlfake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- copies of site-dev-api 20eeab4 (unchanged) ----------------------------

type oldValidateTokenResp struct {
	UserID   int64  `json:"user_id"`
	NIK      string `json:"nik"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type oldApprovalUserProfile struct {
	UserID   int64  `json:"user_id"`
	NIK      string `json:"nik"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type oldApprovalCodeValidateResp struct {
	JWT         string                 `json:"jwt"`
	SiteID      string                 `json:"site_id"`
	UserProfile oldApprovalUserProfile `json:"user_profile"`
}

// ---------------------------------------------------------------------------

const testCode = "AbCdEfGhIjKlMnOpQrStUvWxYz012345" // 32 characters, test only

func TestSideBySide_ValidateToken(t *testing.T) {
	h, rec, _ := stack(t, userRow(1001, "andi", "andi@example.com", "USER"))

	code, body := call(t, h, http.MethodPost, "/v1/validation", nil, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	// site-dev-api: nik and username came from its token (nik "1001",
	// username "andi"), role from Users; same values here from Users
	assertSameJSON(t, oldDataEnvelope(t, oldValidateTokenResp{UserID: 1001, NIK: "1001", Username: "andi", Role: "USER"}), body)
	assert.Equal(t, []string{oldDetailSQL[0]}, rec.sqls)
}

func jwtServer(t *testing.T, status int, answer string) (*httptest.Server, *[]map[string]string) {
	var seen []map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		user, pass, _ := r.BasicAuth()
		seen = append(seen, map[string]string{"path": r.URL.Path, "body": string(b), "basic": user + ":" + pass, "type": r.Header.Get("Content-Type")})
		w.WriteHeader(status)
		_, _ = io.WriteString(w, answer)
	}))
	t.Cleanup(srv.Close)
	testEnv.Set("approval_jwt.base_url", srv.URL)
	testEnv.Set("public_api.username", "test-user")
	testEnv.Set("public_api.password", "test-pass")
	t.Cleanup(func() {
		testEnv.Set("approval_jwt.base_url", "")
		testEnv.Set("public_api.username", "")
		testEnv.Set("public_api.password", "")
	})
	return srv, &seen
}

func approvalLinkResults(status string) []sqlfake.Result {
	return []sqlfake.Result{
		{Match: `FROM "ApprovalDET" WHERE ID = @p1`, Columns: []string{"ID", "ApprovalHeaderID", "Level", "ApproverRoleID", "ApproverID", "Status", "SecureApprovalCode"},
			Rows: [][]driver.Value{{int64(32), int64(7), int64(2), "DIV_HEAD", int64(3002), status, testCode}}},
		{Match: `FROM "ApprovalHDR" WHERE ID = @p1`, Columns: []string{"ID", "TaskID"}, Rows: [][]driver.Value{{int64(7), "TS1"}}},
		{Match: `FROM "Tasks" WHERE TaskID = @p1`, Columns: taskColumns, Rows: [][]driver.Value{{"TS1", "ST1", "WAITING_DIV_HEAD", int64(1001), int64(3001), created, updated}}},
		userRow(3002, "dodi", "dodi@example.com", "DIV_HEAD"),
	}
}

func TestSideBySide_ValidateApprovalCode(t *testing.T) {
	_, seen := jwtServer(t, http.StatusOK, `{"code":200,"status":"success","data":{"token":"issued-test-token","expires_at":"x"}}`)
	srv := withRedis(t)
	srv.Set("site-dev-api:test:approval_code:"+testCode+":", "32")
	h, rec, _ := stack(t, approvalLinkResults("PENDING")...)

	code, body := call(t, h, http.MethodPost, "/v1/validation/approval", map[string]any{"approval_code": testCode}, 0)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, oldDataEnvelope(t, oldApprovalCodeValidateResp{JWT: "issued-test-token", SiteID: "ST1",
		UserProfile: oldApprovalUserProfile{UserID: 3002, NIK: "3002", Username: "dodi", Role: "DIV_HEAD"}}), body)
	assert.Equal(t, []string{
		`SELECT * FROM "ApprovalDET" WHERE ID = 32 ORDER BY "ApprovalDET"."ID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT * FROM "ApprovalHDR" WHERE ID = 7 ORDER BY "ApprovalHDR"."ID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT * FROM "Tasks" WHERE TaskID = 'TS1' ORDER BY "Tasks"."TaskID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT * FROM "Users" WHERE UserID = 3002 ORDER BY "Users"."UserID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
	}, rec.sqls)
	require.Len(t, *seen, 1)
	call := (*seen)[0]
	assert.Equal(t, "/fmi-web/v1/auth/jwt", call["path"])
	assert.Equal(t, "test-user:test-pass", call["basic"])
	assert.Equal(t, "application/json", call["type"])
	var sent map[string]string
	require.NoError(t, json.Unmarshal([]byte(call["body"]), &sent))
	assert.Equal(t, map[string]string{"username": "dodi", "role": "Site Dev", "nik": "3002"}, sent)
}

func TestSideBySide_ValidateApprovalCode_Refused(t *testing.T) {
	_, seen := jwtServer(t, http.StatusOK, `{"data":{"token":"x"}}`)
	srv := withRedis(t)
	srv.Set("site-dev-api:test:approval_code:"+testCode+":", "32")

	for _, tc := range []struct {
		name, code, status, want string
	}{
		{"not in redis", "ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ", "PENDING", "approval_validation: code is invalid or expired"},
		{"already decided", testCode, "APPROVED", "approval_validation: approval is no longer pending"},
	} {
		h, _, _ := stack(t, approvalLinkResults(tc.status)...)
		code, body := call(t, h, http.MethodPost, "/v1/validation/approval", map[string]any{"approval_code": tc.code}, 0)
		assert.Equal(t, http.StatusInternalServerError, code, tc.name)
		assertSameJSON(t, []byte(`{"code":500,"status":"failure","message":"Internal Server Error","errors":{"error":"`+tc.want+`"}}`), body)
	}
	assert.Empty(t, *seen, "no token is asked for")

	h, _, _ := stack(t, approvalLinkResults("PENDING")...)
	code, _ := call(t, h, http.MethodPost, "/v1/validation/approval", map[string]any{"approval_code": "short"}, 0)
	assert.Equal(t, http.StatusUnprocessableEntity, code) // DIFFERENCE: site-dev-api 500 (go-playground len=32)
}

func TestSideBySide_ValidateApprovalCode_TokenServiceDown(t *testing.T) {
	jwtServer(t, http.StatusBadGateway, `upstream down`)
	srv := withRedis(t)
	srv.Set("site-dev-api:test:approval_code:"+testCode+":", "32")
	h, _, _ := stack(t, approvalLinkResults("PENDING")...)

	code, body := call(t, h, http.MethodPost, "/v1/validation/approval", map[string]any{"approval_code": testCode}, 0)

	assert.Equal(t, http.StatusInternalServerError, code)
	assertSameJSON(t, []byte(`{"code":500,"status":"failure","message":"Internal Server Error","errors":{"error":"approval_jwt: unexpected status 502: upstream down"}}`), body)
}
