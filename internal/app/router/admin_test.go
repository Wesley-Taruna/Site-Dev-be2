package router

// Side-by-side check, module 4 (master data / admin): wilayah, roles,
// users, approval matrix and settings against site-dev-api 20eeab4.

import (
	"database/sql/driver"
	"fmt"
	"net/http"
	"testing"
	"time"

	"site-dev-backend/internal/pkg/sqlfake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- copies of site-dev-api 20eeab4 internal/app/dto + model (unchanged) ---

type oldRole struct {
	RoleID    string `json:"role_id" gorm:"column:RoleID;primaryKey"`
	RoleName  string `json:"role_name" gorm:"column:RoleName"`
	RoleLevel int    `json:"role_level" gorm:"column:RoleLevel"`
}

type oldUserResp struct {
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

type oldApprovalMatrixResp struct {
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

type oldLimitsResp struct {
	AIInsightDailyQuota    int64      `json:"ai_insight_daily_quota"`
	AIInsightUpdatedBy     int64      `json:"ai_insight_updated_by,omitempty"`
	AIInsightUpdatedByName string     `json:"ai_insight_updated_by_name,omitempty"`
	AIInsightUpdatedAt     *time.Time `json:"ai_insight_updated_at,omitempty"`
	GeocodingDailyQuota    int64      `json:"geocoding_daily_quota"`
	GeocodingUpdatedBy     int64      `json:"geocoding_updated_by,omitempty"`
	GeocodingUpdatedByName string     `json:"geocoding_updated_by_name,omitempty"`
	GeocodingUpdatedAt     *time.Time `json:"geocoding_updated_at,omitempty"`
}

type oldQuotaFlushResp struct {
	UserID    int64  `json:"user_id"`
	QuotaType string `json:"quota_type"`
	Remaining int64  `json:"remaining"`
	Limit     int64  `json:"limit"`
}

type oldWilayahSearchResp struct {
	VillageCode  string `json:"village_code"`
	VillageName  string `json:"village_name"`
	VillageType  string `json:"village_type"`
	PostalCode   string `json:"postal_code"`
	DistrictCode string `json:"district_code"`
	DistrictName string `json:"district_name"`
	RegencyCode  string `json:"regency_code"`
	RegencyName  string `json:"regency_name"`
	ProvinceCode string `json:"province_code"`
	ProvinceName string `json:"province_name"`
}

type oldProvinceResp struct {
	ProvinceCode string `json:"province_code"`
	ProvinceName string `json:"province_name"`
}

type oldProvinceTreeResp struct {
	ProvinceCode string               `json:"province_code"`
	ProvinceName string               `json:"province_name"`
	Regencies    []oldRegencyTreeResp `json:"regencies"`
}

type oldRegencyTreeResp struct {
	RegencyCode string                `json:"regency_code"`
	RegencyName string                `json:"regency_name"`
	Districts   []oldDistrictTreeResp `json:"districts"`
}

type oldDistrictTreeResp struct {
	DistrictCode string           `json:"district_code"`
	DistrictName string           `json:"district_name"`
	Villages     []oldVillageResp `json:"villages"`
}

type oldVillageResp struct {
	VillageCode string `json:"village_code"`
	VillageName string `json:"village_name"`
	VillageType string `json:"village_type"`
	PostalCode  string `json:"postal_code"`
}

// ---------------------------------------------------------------------------

var roleColumns = []string{"RoleID", "RoleName", "RoleLevel"}

func roleRow(id, name string, level int64) sqlfake.Result {
	return sqlfake.Result{Match: fmt.Sprintf(`FROM "Roles" WHERE RoleID = @p1 ORDER BY "Roles"."RoleID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY |%s`, id),
		Columns: roleColumns, Rows: [][]driver.Value{{id, name, level}}}
}

var adminResults = []sqlfake.Result{
	userRow(9001, "admin", "admin@example.com", "ADMIN"),
	userRow(1001, "andi", "andi@example.com", "USER"),
	userRow(3001, "citra", "citra@example.com", "DEPT_HEAD"),
	roleRow("ADMIN", "Admin", 0),
	roleRow("USER", "Staff Site", 0),
	roleRow("DEPT_HEAD", "Dept Head", 1),
	roleRow("DIV_HEAD", "Div Head", 2),
}

func admin(extra ...sqlfake.Result) []sqlfake.Result {
	return append(extra, adminResults...)
}

func quotaKey(kind string, userID int64) string {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	return fmt.Sprintf("site-dev-api:test:counter:%s_quota:%d:%s:", kind, userID, time.Now().In(loc).Format("20060102"))
}

func TestSideBySide_AdminOnly(t *testing.T) {
	h, _, _ := stack(t, admin()...)

	for _, r := range []struct{ method, path string }{
		{http.MethodGet, "/v1/roles"}, {http.MethodGet, "/v1/users"}, {http.MethodPost, "/v1/users"},
		{http.MethodPut, "/v1/users/1001"}, {http.MethodDelete, "/v1/users/1001"},
		{http.MethodGet, "/v1/settings/limits"}, {http.MethodPut, "/v1/settings/limits"}, {http.MethodPost, "/v1/settings/quota-flush"},
	} {
		// the admin check comes before reading the body, like requireAdmin
		code, body := call(t, h, r.method, r.path, map[string]any{"bad": true}, 3001)
		assert.Equal(t, http.StatusForbidden, code, r.path)
		assertSameJSON(t, []byte(`{"code":403,"status":"failure","message":"Forbidden","errors":{"error":"hanya admin yang bisa mengakses fitur ini"}}`), body)
	}
}

func TestSideBySide_Roles(t *testing.T) {
	h, rec, _ := stack(t, admin(sqlfake.Result{Match: `SELECT * FROM "Roles" ORDER BY RoleLevel ASC`, Columns: roleColumns,
		Rows: [][]driver.Value{{"USER", "Staff Site", int64(0)}, {"DEPT_HEAD", "Dept Head", int64(1)}}})...)

	code, body := call(t, h, http.MethodGet, "/v1/roles", nil, 9001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, oldDataEnvelope(t, []*oldRole{{"USER", "Staff Site", 0}, {"DEPT_HEAD", "Dept Head", 1}}), body)
	assert.Equal(t, `SELECT * FROM "Roles" ORDER BY RoleLevel ASC`, rec.sqls[1])

	h, _, _ = stack(t, admin()...)
	_, body = call(t, h, http.MethodGet, "/v1/roles", nil, 9001)
	assert.Contains(t, string(body), `"data":[]`, "gorm Find gives [] for no rows")
}

func TestSideBySide_UserList(t *testing.T) {
	srv := withRedis(t)
	srv.Set(quotaKey("ai_insight", 1001), "7")
	srv.Set(quotaKey("geocoding", 1001), "-2") // over the limit: shown as 0
	h, rec, _ := stack(t, admin(
		sqlfake.Result{Match: `SELECT * FROM "Users" ORDER BY Username ASC`, Columns: userColumns, Rows: [][]driver.Value{
			{int64(1001), "andi", "andi@example.com", "USER", created, updated},
			{int64(4004), "dewi", "dewi@example.com", "GONE", created, created},
		}},
		sqlfake.Result{Match: `|ai_insight.daily_quota`, Columns: []string{"ID", "SettingKey", "SettingValue", "UpdatedBy", "CreatedAt", "UpdatedAt"},
			Rows: [][]driver.Value{{int64(1), "ai_insight.daily_quota", "30", int64(9001), created, updated}}},
	)...)

	code, body := call(t, h, http.MethodGet, "/v1/users", nil, 9001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, oldDataEnvelope(t, []oldUserResp{
		{UserID: 1001, Username: "andi", Email: "andi@example.com", RoleID: "USER", RoleName: "Staff Site",
			AIInsightRemaining: 7, AIInsightLimit: 30, GeocodingRemaining: 0, GeocodingLimit: 100, CreatedAt: created, UpdatedAt: updated},
		{UserID: 4004, Username: "dewi", Email: "dewi@example.com", RoleID: "GONE",
			AIInsightRemaining: 30, AIInsightLimit: 30, GeocodingRemaining: 100, GeocodingLimit: 100, CreatedAt: created, UpdatedAt: created},
	}), body)
	assert.Equal(t, `SELECT * FROM "SystemSettings" WHERE SettingKey = 'ai_insight.daily_quota' ORDER BY "SystemSettings"."ID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`, rec.sqls[2])
	// the limit read from the table is now cached for a week, like site-dev-api
	v, ok := srv.Get("site-dev-api:test:cache:system_setting:ai_insight.daily_quota:")
	assert.True(t, ok)
	assert.Equal(t, "30", v)
	assert.Equal(t, 7*24*time.Hour, srv.TTL("site-dev-api:test:cache:system_setting:ai_insight.daily_quota:"))
}

func TestSideBySide_UserCreate(t *testing.T) {
	h, rec, _ := stack(t, admin()...)

	code, body := call(t, h, http.MethodPost, "/v1/users", map[string]any{"user_id": 5005, "username": "eko", "email": "eko@example.com", "role_id": "DEPT_HEAD"}, 9001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, oldDataEnvelope(t, oldUserResp{UserID: 5005, Username: "eko", Email: "eko@example.com", RoleID: "DEPT_HEAD", RoleName: "Dept Head",
		CreatedAt: gormNow, UpdatedAt: gormNow}), body)
	assert.Equal(t, []string{
		`SELECT * FROM "Users" WHERE UserID = 9001 ORDER BY "Users"."UserID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT * FROM "Roles" WHERE RoleID = 'DEPT_HEAD' ORDER BY "Roles"."RoleID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SET IDENTITY_INSERT "Users" ON;INSERT INTO "Users" ("Username","Email","RoleID","CreatedAt","UpdatedAt","UserID") OUTPUT INSERTED."UserID" VALUES ('eko','eko@example.com','DEPT_HEAD','2026-10-21 08:00:00','2026-10-21 08:00:00',5005);SET IDENTITY_INSERT "Users" OFF;`,
	}, rec.sqls)
}

func TestSideBySide_UserCreate_UnknownRole(t *testing.T) {
	h, _, _ := stack(t, admin()...)

	code, body := call(t, h, http.MethodPost, "/v1/users", map[string]any{"user_id": 5005, "username": "eko", "email": "eko@example.com", "role_id": "CEO"}, 9001)

	assert.Equal(t, http.StatusInternalServerError, code)
	assertSameJSON(t, []byte(`{"code":500,"status":"failure","message":"Internal Server Error","errors":{"error":"user_service: role CEO not found: record not found"}}`), body)
}

func TestUserCreate_ValidationIs422(t *testing.T) {
	// DIFFERENCE (boilerplate binder): site-dev-api answered 500 with the
	// go-playground text ("Key: 'UserCreateReq.Email' Error:Field
	// validation for 'Email' failed on the 'email' tag"); the boilerplate's
	// familabs binder answers 422 with one message per field.
	h, rec, _ := stack(t, admin()...)

	code, body := call(t, h, http.MethodPost, "/v1/users", map[string]any{"user_id": 5005, "email": "not-an-email"}, 9001)

	assert.Equal(t, http.StatusUnprocessableEntity, code)
	assert.Contains(t, string(body), `"username":"The username field is required."`)
	assert.Contains(t, string(body), `"role_id":"The role id field is required."`)
	assert.Contains(t, string(body), `"email"`)
	assert.Len(t, rec.sqls, 1, "only the admin check ran")
}

func TestSideBySide_UserUpdate(t *testing.T) {
	h, rec, _ := stack(t, admin()...)

	code, body := call(t, h, http.MethodPut, "/v1/users/1001", map[string]any{"role_id": "DEPT_HEAD"}, 9001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, oldDataEnvelope(t, oldUserResp{UserID: 1001, Username: "andi", Email: "andi@example.com", RoleID: "USER", RoleName: "Staff Site",
		CreatedAt: created, UpdatedAt: created}), body)
	assert.Equal(t, []string{
		`SELECT * FROM "Users" WHERE UserID = 9001 ORDER BY "Users"."UserID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT * FROM "Roles" WHERE RoleID = 'DEPT_HEAD' ORDER BY "Roles"."RoleID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`UPDATE "Users" SET "RoleID"='DEPT_HEAD',"UpdatedAt"='2026-10-21 08:00:00' WHERE UserID = 1001`,
		`SELECT * FROM "Users" WHERE UserID = 1001 ORDER BY "Users"."UserID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT * FROM "Roles" WHERE RoleID = 'USER' ORDER BY "Roles"."RoleID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
	}, rec.sqls)

	code, body = call(t, h, http.MethodPut, "/v1/users/abc", map[string]any{}, 9001)
	assert.Equal(t, http.StatusUnprocessableEntity, code)
	assertSameJSON(t, []byte(`{"code":422,"status":"failure","message":"Unprocessable Entity","errors":{"id":"The id is invalid value."}}`), body)
}

func TestSideBySide_UserDelete(t *testing.T) {
	h, rec, _ := stack(t, admin()...)

	code, body := call(t, h, http.MethodDelete, "/v1/users/1001", nil, 9001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, []byte(`{"code":200,"status":"success"}`), body)
	assert.Equal(t, `DELETE FROM "Users" WHERE UserID = 1001`, rec.sqls[1])
}

var matrixColumns = []string{"ID", "RoleID", "UserID", "IsActive", "CreatedAt", "UpdatedAt"}

func TestSideBySide_ApprovalMatrix(t *testing.T) {
	h, rec, _ := stack(t, admin(
		sqlfake.Result{Match: `SELECT * FROM "ApprovalMatrix" ORDER BY RoleID ASC`, Columns: matrixColumns, Rows: [][]driver.Value{
			{int64(1), "DEPT_HEAD", int64(3001), true, created, updated},
			{int64(2), "DIV_HEAD", int64(7007), false, created, updated},
		}},
		sqlfake.Result{Match: `INSERT INTO "ApprovalMatrix"`, Columns: []string{"ID"}, Rows: [][]driver.Value{{int64(3)}}},
	)...)

	// any logged-in user may use the matrix (site-dev-api has no admin check here)
	code, body := call(t, h, http.MethodGet, "/v1/approval-matrix", nil, 1001)
	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, oldDataEnvelope(t, []oldApprovalMatrixResp{
		{ID: 1, RoleID: "DEPT_HEAD", RoleName: "Dept Head", RoleLevel: 1, UserID: 3001, Username: "citra", IsActive: true, CreatedAt: created, UpdatedAt: updated},
		{ID: 2, RoleID: "DIV_HEAD", RoleName: "Div Head", RoleLevel: 2, UserID: 7007, CreatedAt: created, UpdatedAt: updated},
	}), body)
	// no user lookup of the caller (no role needed; plan §4 diff 2)
	assert.Equal(t, `SELECT * FROM "ApprovalMatrix" ORDER BY RoleID ASC`, rec.sqls[0])

	code, body = call(t, h, http.MethodPost, "/v1/approval-matrix", map[string]any{"role_id": "DEPT_HEAD", "user_id": 3001}, 1001)
	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, oldDataEnvelope(t, oldApprovalMatrixResp{ID: 3, RoleID: "DEPT_HEAD", RoleName: "Dept Head", RoleLevel: 1, UserID: 3001, Username: "citra",
		IsActive: true, CreatedAt: gormNow, UpdatedAt: gormNow}), body)
	assert.Contains(t, rec.sqls, `INSERT INTO "ApprovalMatrix" ("RoleID","UserID","IsActive","CreatedAt","UpdatedAt") OUTPUT INSERTED."ID" VALUES ('DEPT_HEAD',3001,1,'2026-10-21 08:00:00','2026-10-21 08:00:00');`)

	code, body = call(t, h, http.MethodPut, "/v1/approval-matrix/2", map[string]any{"user_id": 3001, "is_active": true}, 1001)
	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, []byte(`{"code":200,"status":"success"}`), body)
	assert.Equal(t, `UPDATE "ApprovalMatrix" SET "IsActive"=1,"UserID"=3001,"UpdatedAt"='2026-10-21 08:00:00' WHERE ID = 2`, rec.sqls[len(rec.sqls)-1])
}

func TestSideBySide_SettingsLimits(t *testing.T) {
	srv := withRedis(t)
	settingRow := []string{"ID", "SettingKey", "SettingValue", "UpdatedBy", "CreatedAt", "UpdatedAt"}
	h, rec, _ := stack(t, admin(
		sqlfake.Result{Match: `|geocoding.daily_quota`, Columns: settingRow, Rows: [][]driver.Value{{int64(2), "geocoding.daily_quota", "250", int64(9001), created, updated}}},
	)...)
	testEnv.Set("ai_insight.daily_quota", 40)
	t.Cleanup(func() { testEnv.Set("ai_insight.daily_quota", 0) })

	code, body := call(t, h, http.MethodGet, "/v1/settings/limits", nil, 9001)

	require.Equal(t, http.StatusOK, code, string(body))
	updatedAt := updated
	assertSameJSON(t, oldDataEnvelope(t, oldLimitsResp{
		AIInsightDailyQuota: 40, // never saved: env.toml value
		GeocodingDailyQuota: 250, GeocodingUpdatedBy: 9001, GeocodingUpdatedByName: "admin", GeocodingUpdatedAt: &updatedAt,
	}), body)

	rec.sqls = nil
	code, body = call(t, h, http.MethodPut, "/v1/settings/limits", map[string]any{"ai_insight_daily_quota": 60}, 9001)
	require.Equal(t, http.StatusOK, code, string(body))
	assert.Contains(t, rec.sqls, `INSERT INTO "SystemSettings" ("SettingKey","SettingValue","UpdatedBy","CreatedAt","UpdatedAt") OUTPUT INSERTED."ID" VALUES ('ai_insight.daily_quota','60',9001,'2026-10-21 08:00:00','2026-10-21 08:00:00');`)
	v, _ := srv.Get("site-dev-api:test:cache:system_setting:ai_insight.daily_quota:")
	assert.Equal(t, "60", v, "saving refreshes the cache at once")
	assert.Contains(t, string(body), `"ai_insight_daily_quota":60`)

	code, body = call(t, h, http.MethodPut, "/v1/settings/limits", map[string]any{"geocoding_daily_quota": 0}, 9001)
	// DIFFERENCE: site-dev-api 500 (go-playground "min" text), new 422
	assert.Equal(t, http.StatusUnprocessableEntity, code)
	assert.Contains(t, string(body), `"geocoding_daily_quota"`)
}

func TestSideBySide_SettingsQuotaFlush(t *testing.T) {
	srv := withRedis(t)
	srv.Set(quotaKey("geocoding", 1001), "0")
	srv.Set(quotaKey("ai_insight", 1001), "0")
	h, _, _ := stack(t, admin()...)

	code, body := call(t, h, http.MethodPost, "/v1/settings/quota-flush", map[string]any{"user_id": 1001, "quota_type": "geocoding"}, 9001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, oldDataEnvelope(t, oldQuotaFlushResp{UserID: 1001, QuotaType: "geocoding", Remaining: 100, Limit: 100}), body)
	_, stillThere := srv.Get(quotaKey("geocoding", 1001))
	assert.False(t, stillThere)
	_, other := srv.Get(quotaKey("ai_insight", 1001))
	assert.True(t, other, "only the chosen quota is reset")

	code, body = call(t, h, http.MethodPost, "/v1/settings/quota-flush", map[string]any{"user_id": 4040, "quota_type": "ai_insight"}, 9001)
	assert.Equal(t, http.StatusInternalServerError, code)
	assertSameJSON(t, []byte(`{"code":500,"status":"failure","message":"Internal Server Error","errors":{"error":"settings_service: user 4040 not found: record not found"}}`), body)

	code, _ = call(t, h, http.MethodPost, "/v1/settings/quota-flush", map[string]any{"user_id": 1001, "quota_type": "gemini"}, 9001)
	assert.Equal(t, http.StatusUnprocessableEntity, code) // DIFFERENCE: site-dev-api 500 (validation)
}

var wilayahColumns = []string{"VillageCode", "VillageName", "VillageType", "PostalCode", "DistrictCode", "DistrictName", "RegencyCode", "RegencyName", "ProvinceCode", "ProvinceName"}

func TestSideBySide_WilayahSearch(t *testing.T) {
	h, rec, _ := stack(t,
		sqlfake.Result{Match: `SELECT count(*) FROM "MVillages"`, Columns: []string{"count"}, Rows: [][]driver.Value{{int64(1)}}},
		sqlfake.Result{Match: `FROM MVillages v`, Columns: wilayahColumns, Rows: [][]driver.Value{
			{"3201010001", "Tajurhalang", "DESA", "16320", "320101", "Tajurhalang", "3201", "Kab. Bogor", "32", "Jawa Barat"}}},
	)

	code, body := call(t, h, http.MethodGet, "/v1/wilayah/search?q=tajur", nil, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, oldListEnvelope(t, []oldWilayahSearchResp{{"3201010001", "Tajurhalang", "DESA", "16320", "320101", "Tajurhalang", "3201", "Kab. Bogor", "32", "Jawa Barat"}}, 1, 1, 0), body)
	assert.Equal(t, `SELECT count(*) FROM "MVillages" WHERE VillageName LIKE '%tajur%'`, rec.sqls[0])
	assert.Equal(t, squash(`SELECT v.VillageCode, v.VillageName, v.VillageType, v.PostalCode, d.DistrictCode, d.DistrictName, r.RegencyCode, r.RegencyName, p.ProvinceCode, p.ProvinceName
		FROM MVillages v INNER JOIN MDistricts d ON d.DistrictCode = v.DistrictCode INNER JOIN MRegencies r ON r.RegencyCode = d.RegencyCode
		INNER JOIN MProvinces p ON p.ProvinceCode = r.ProvinceCode WHERE v.VillageName LIKE '%tajur%' ORDER BY v.VillageName OFFSET 0 ROWS FETCH NEXT 20 ROWS ONLY`), squash(rec.sqls[1]))

	rec.sqls = nil
	code, _ = call(t, h, http.MethodGet, "/v1/wilayah/search?postal_code=16320&q=x", nil, 1001)
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, `SELECT count(*) FROM "MVillages" WHERE PostalCode = '16320'`, rec.sqls[0], "postal_code wins over q")

	code, body = call(t, h, http.MethodGet, "/v1/wilayah/search", nil, 1001)
	require.Equal(t, http.StatusOK, code)
	assertSameJSON(t, []byte(`{"code":200,"status":"success","data":[]}`), body)

	code, body = call(t, h, http.MethodGet, "/v1/wilayah/search?q=ta", nil, 1001)
	assert.Equal(t, http.StatusUnprocessableEntity, code)
	assertSameJSON(t, []byte(`{"code":422,"status":"failure","message":"Unprocessable Entity","errors":{"q (minimal 3 karakter)":"The q (minimal 3 karakter) is invalid value."}}`), body)
}

func TestSideBySide_WilayahProvinces(t *testing.T) {
	h, rec, _ := stack(t,
		sqlfake.Result{Match: `FROM "MProvinces" WHERE ProvinceCode`, Columns: []string{"ProvinceCode", "ProvinceName", "CreatedAt"}, Rows: [][]driver.Value{{"32", "Jawa Barat", created}}},
		sqlfake.Result{Match: `FROM "MProvinces"`, Columns: []string{"ProvinceCode", "ProvinceName", "CreatedAt"}, Rows: [][]driver.Value{{"11", "Aceh", created}, {"32", "Jawa Barat", created}}},
		sqlfake.Result{Match: `FROM "MRegencies"`, Columns: []string{"RegencyCode", "RegencyName", "ProvinceCode"}, Rows: [][]driver.Value{{"3201", "Kab. Bogor", "32"}, {"3271", "Kota Bogor", "32"}}},
		sqlfake.Result{Match: `FROM "MDistricts"`, Columns: []string{"DistrictCode", "DistrictName", "RegencyCode"}, Rows: [][]driver.Value{{"320101", "Cibinong", "3201"}, {"320102", "Tajurhalang", "3201"}}},
		sqlfake.Result{Match: `FROM "MVillages"`, Columns: []string{"VillageCode", "VillageName", "VillageType", "PostalCode", "DistrictCode"}, Rows: [][]driver.Value{
			{"3201010001", "Pakansari", "KEL", "16915", "320101"}, {"3201020001", "Tajurhalang", "DESA", "16320", "320102"}}},
	)

	code, body := call(t, h, http.MethodGet, "/v1/wilayah/provinces", nil, 1001)
	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, oldDataEnvelope(t, []oldProvinceResp{{"11", "Aceh"}, {"32", "Jawa Barat"}}), body)
	assert.Equal(t, []string{`SELECT * FROM "MProvinces" ORDER BY ProvinceName`}, rec.sqls)

	rec.sqls = nil
	code, body = call(t, h, http.MethodGet, "/v1/wilayah/provinces/32", nil, 1001)
	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, oldDataEnvelope(t, oldProvinceTreeResp{ProvinceCode: "32", ProvinceName: "Jawa Barat", Regencies: []oldRegencyTreeResp{
		{RegencyCode: "3201", RegencyName: "Kab. Bogor", Districts: []oldDistrictTreeResp{
			{DistrictCode: "320101", DistrictName: "Cibinong", Villages: []oldVillageResp{{"3201010001", "Pakansari", "KEL", "16915"}}},
			{DistrictCode: "320102", DistrictName: "Tajurhalang", Villages: []oldVillageResp{{"3201020001", "Tajurhalang", "DESA", "16320"}}},
		}},
		{RegencyCode: "3271", RegencyName: "Kota Bogor"}, // no districts: null, as in site-dev-api
	}}), body)
	assert.Equal(t, []string{
		`SELECT * FROM "MProvinces" WHERE ProvinceCode = '32' ORDER BY "MProvinces"."ProvinceCode" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT * FROM "MRegencies" WHERE ProvinceCode = '32' ORDER BY RegencyName`,
		`SELECT * FROM "MDistricts" WHERE RegencyCode IN ('3201','3271') ORDER BY DistrictName`,
		`SELECT * FROM "MVillages" WHERE DistrictCode IN ('320101','320102') ORDER BY VillageName`,
	}, rec.sqls)
}
