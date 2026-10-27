package router

// Side-by-side check, modules 6c + 7 (save/send, decide, queue callbacks):
// POST /v1/task/:status, POST /v1/task/:id/approve and POST /v1/queue/*
// against site-dev-api 20eeab4. E-mail, background jobs and Gemini are
// mocks (their real clients are TODOs, plan Q4); the job queue is off, so
// e-mail and AI insight run at once, as site-dev-api does with
// cloud_task.enabled = false.

import (
	"context"
	"database/sql/driver"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"site-dev-backend/internal/app/client"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/mocks"
	"site-dev-backend/internal/pkg/redisfake"
	"site-dev-backend/internal/pkg/sqlfake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- copies of site-dev-api 20eeab4 internal/app/dto/site.go (unchanged) ---

type oldAttachmentResp struct {
	SiteAttachmentID uint      `json:"site_attachment_id"`
	FileName         string    `json:"file_name"`
	Path             string    `json:"path"`
	CreatedAt        time.Time `json:"created_at"`
}

type oldSiteUpsertResp struct {
	Site        oldSiteResp           `json:"site"`
	Context     *oldSiteContextResp   `json:"context,omitempty"`
	Attachments []oldAttachmentResp   `json:"attachments,omitempty"`
	Task        oldTaskFullDetailResp `json:"task"`
}

// ---------------------------------------------------------------------------

var anyTime = regexp.MustCompile(`"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})"`)

// assertSameJSONButTimes is assertSameJSON with every timestamp replaced
// (the save flow stamps time.Now()).
func assertSameJSONButTimes(t *testing.T, want, got []byte) {
	assertSameJSON(t, anyTime.ReplaceAll(want, []byte(`"<t>"`)), anyTime.ReplaceAll(got, []byte(`"<t>"`)))
}

func jakartaToday() string {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	return time.Now().In(loc).Format("20060102")
}

type flowMocks struct {
	mail   *mocks.IMailClient
	gemini *mocks.IGeminiClient
	redis  *redisfake.Server
}

func withFlow(t *testing.T) flowMocks {
	m := flowMocks{mail: mocks.NewIMailClient(t), gemini: mocks.NewIGeminiClient(t)}
	withClients(t, func(c *client.Clients) {
		c.Mail = m.mail
		c.Gemini = m.gemini
	})
	m.redis = withRedis(t)
	return m
}

var detailCols = []string{"ID", "ApprovalHeaderID", "Level", "ApproverRoleID", "ApproverID", "Status", "Notes", "ActedAt", "SecureApprovalCode", "CreatedAt", "UpdatedAt"}

func chainResult() sqlfake.Result {
	return sqlfake.Result{Match: `FROM "ApprovalMatrix" JOIN Roles`, Columns: []string{"RoleID", "RoleLevel", "UserID"},
		Rows: [][]driver.Value{{"DEPT_HEAD", int64(1), int64(3001)}, {"DIV_HEAD", int64(2), int64(3002)}}}
}

func TestSideBySide_SaveNewSiteAndSubmit(t *testing.T) {
	m := withFlow(t)
	// DIFFERENCE (Wes, 27 Oct): new site ID format SDdd/mm/yyyy + 3 digits
	// (was "ST" + yyyymmdd + 6 digits); same Redis counter key
	loc, _ := time.LoadLocation("Asia/Jakarta")
	siteID := "SD" + time.Now().In(loc).Format("02/01/2006") + "001"
	taskID := "TS" + jakartaToday() + "000001"
	newSite := append([]driver.Value{siteID, "Toko Baru"}, siteST1Row()[2:]...)
	newSite[22] = nil // never updated
	h, rec, fdb := stack(t,
		noRows(`WHERE SiteName1 = @p1`),
		noRows(`SELECT count(*) FROM "Sites"`),
		noRows(`SELECT count(*) FROM "SiteContexts"`),
		noRows(`FROM "Tasks" WHERE SiteID = @p1`),
		sqlfake.Result{Match: `INSERT INTO "SiteAttachments"`, Columns: []string{"SiteAttachmentID"}, Rows: [][]driver.Value{{int64(41)}}},
		sqlfake.Result{Match: `INSERT INTO "SiteCompetitors"`, Columns: []string{"ID"}, Rows: [][]driver.Value{{int64(51)}}},
		sqlfake.Result{Match: `INSERT INTO "ApprovalHDR"`, Columns: []string{"ID"}, Rows: [][]driver.Value{{int64(7)}}},
		sqlfake.Result{Match: `INSERT INTO "ApprovalDET"`, Columns: []string{"ID"}, Rows: [][]driver.Value{{int64(31)}, {int64(32)}}},
		sqlfake.Result{Match: `INSERT INTO "SiteAIInsights"`, Columns: []string{"SiteAIID"}, Rows: [][]driver.Value{{int64(77)}}},
		chainResult(),
		sqlfake.Result{Match: `FROM "Tasks" WHERE TaskID = @p1`, Columns: taskColumns, Rows: [][]driver.Value{{taskID, siteID, "DRAFT", int64(1001), int64(0), created, created}}},
		sqlfake.Result{Match: `FROM "Sites" WHERE SiteID = @p1`, Columns: siteColumns, Rows: [][]driver.Value{newSite}},
		noRows(`FROM "SiteContexts" WHERE SiteID`),
		noRows(`FROM "ApprovalHDR" WHERE TaskID`),
		noRows(`FROM "SiteAttachments" WHERE SiteID`),
		noRows(`FROM "TaskHistories" WHERE TaskID`),
		userRow(1001, "andi", "andi@example.com", "USER"),
		userRow(3001, "citra", "citra@example.com", "DEPT_HEAD"),
		roleRow("DEPT_HEAD", "Dept Head", 1),
	)
	var sentCode string
	m.mail.On("SendApprovalPending", mock.Anything, "citra@example.com", "citra", mock.MatchedBy(func(e client.ApprovalPendingEmail) bool {
		sentCode = e.Code
		return e.TaskID == taskID && e.SiteName == "Toko Baru" && e.SubmittedBy == "andi" && e.ApproverRole == "Dept Head" &&
			e.Notes == "Site diajukan oleh andi untuk proses approval"
	})).Return(nil)
	m.gemini.On("GenerateInsight", mock.Anything, mock.Anything).Return(geminiFor)

	code, body := call(t, h, http.MethodPost, "/v1/task/SUBMITED", map[string]any{
		"site": map[string]any{"site_name_1": "Toko Baru", "site_name_2": "Kemang 2", "site_address_1": "Jl. Kemang Raya 1", "site_address_2": "RT 01",
			"site_address_3": "Jakarta Selatan", "site_postal_code": "12730", "site_district": "Mampang", "site_city": "Jakarta", "site_region": "DKI",
			"site_lat": -6.26, "site_lon": 106.81, "site_telepon": "021-1", "site_telepon_ext": "12", "site_mobile_phone": "0812", "site_pic": "Pak Joko",
			"site_planned_opening_date": "2027-01-15", "site_sales_area": 120.5, "site_open_24_hours": true},
		"context":    map[string]any{"peak_hours": "17-20", "competitors": []map[string]any{{"competitor_name": "Indomaret", "distance": "50m"}}},
		"attachment": []map[string]any{{"file_name": "depan.jpg", "path": "bucket/temp/depan.jpg"}},
	}, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	site := oldSiteST1()
	// UpdatedAt is set on create too: gorm fills a nil UpdatedAt (same
	// model and gorm version as site-dev-api), UpdatedBy stays NULL
	site.SiteID, site.SiteName1, site.UpdatedBy = siteID, "Toko Baru", nil
	detailSite := site
	detailSite.CreatedByName = "andi"
	assertSameJSONButTimes(t, oldDataEnvelope(t, oldSiteUpsertResp{
		Site:        site,
		Context:     &oldSiteContextResp{SiteID: siteID, PeakHours: "17-20", Competitors: []oldCompetitorResp{{ID: 51, CompetitorName: "Indomaret", Distance: "50m"}}},
		Attachments: []oldAttachmentResp{{SiteAttachmentID: 41, FileName: "depan.jpg", Path: "bucket/temp/depan.jpg"}},
		Task:        oldTaskFullDetailResp{Site: detailSite, Task: oldTaskFullDetailTaskSection{TaskID: taskID, TaskStatus: "DRAFT"}},
	}), body)

	// SQL in site-dev-api's order (its middleware user lookup first)
	sqls := stableSQL(rec.sqls)
	assert.Equal(t, []string{
		`SELECT * FROM "Users" WHERE UserID = 1001 ORDER BY "Users"."UserID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT * FROM "Sites" WHERE SiteName1 = 'Toko Baru' ORDER BY "Sites"."SiteID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT count(*) FROM "Sites" WHERE SiteID = '` + siteID + `'`,
		`INSERT INTO "Sites" ("SiteID","SiteName1","SiteName2","SiteAddress1","SiteAddress2","SiteAddress3","SitePostalCode","SiteDistrict","SiteCity","SiteRegion","SiteLat","SiteLon","SiteTelepon","SiteTeleponExt","SiteMobilePhone","SitePic","SiteFax","SitePlannedOpeningDate","SiteSalesArea","SiteSellingAlcohol","SiteOpen24Hours","CreatedBy","UpdatedBy","CreatedAt","UpdatedAt") VALUES ('` + siteID + `','Toko Baru','Kemang 2','Jl. Kemang Raya 1','RT 01','Jakarta Selatan','12730','Mampang','Jakarta','DKI',-6.26,106.81,'021-1','12','0812','Pak Joko','','<time>',120.5,0,1,1001,NULL,'<time>','<time>');`,
		`DELETE FROM "SiteAttachments" WHERE SiteID = '` + siteID + `'`,
		`INSERT INTO "SiteAttachments" ("SiteID","FileName","Path","CreatedAt") OUTPUT INSERTED."SiteAttachmentID" VALUES ('` + siteID + `','depan.jpg','bucket/temp/depan.jpg','<time>');`,
		`SELECT count(*) FROM "SiteContexts" WHERE SiteID = '` + siteID + `'`,
		`INSERT INTO "SiteContexts" ("SiteID","FootTrafficObservation","PeakHours","ParkingAvailability","PublicTransport","AnchorLandmark","ObservedDemographics","VehicleTrafficLevel","VehicleTrafficCount","VehicleTrafficMinutes","VehicleTrafficNote","ParkingNote","CompetitorNote","CreatedBy","UpdatedBy","CreatedAt","UpdatedAt") VALUES ('` + siteID + `','','17-20',NULL,'','','',NULL,NULL,NULL,NULL,NULL,NULL,1001,1001,'<time>','<time>');`,
		`DELETE FROM "SiteCompetitors" WHERE SiteID = '` + siteID + `'`,
		`INSERT INTO "SiteCompetitors" ("SiteID","CompetitorName","CompetitorType","Distance","CreatedAt") OUTPUT INSERTED."ID" VALUES ('` + siteID + `','Indomaret','','50m','<time>');`,
		`DELETE FROM "SiteCustomNotes" WHERE SiteID = '` + siteID + `'`,
		`SELECT * FROM "Tasks" WHERE SiteID = '` + siteID + `' ORDER BY "Tasks"."TaskID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`INSERT INTO "Tasks" ("TaskID","SiteID","TaskStatus","CreatedBy","UpdatedBy","CreatedAt","UpdatedAt","SubmitCount") VALUES ('` + taskID + `','` + siteID + `','DRAFT',1001,0,'<time>','<time>',1);`,
		`INSERT INTO "TaskHistories" ("TaskID","Action","ActorID","Notes","CreatedAt","ActorRole","Round") OUTPUT INSERTED."ID" VALUES ('` + taskID + `','CREATE',1001,'Site baru dibuat oleh andi','<time>','USER',1);`,
		`SELECT * FROM "Tasks" WHERE TaskID = '` + taskID + `' ORDER BY "Tasks"."TaskID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT ApprovalMatrix.RoleID, Roles.RoleLevel, ApprovalMatrix.UserID FROM "ApprovalMatrix" JOIN Roles ON Roles.RoleID = ApprovalMatrix.RoleID WHERE ApprovalMatrix.IsActive = 1 ORDER BY Roles.RoleLevel ASC`,
		`INSERT INTO "ApprovalHDR" ("TaskID","TotalLevel","CurrentLevel","OverallStatus","SubmittedBy","SubmittedAt","CreatedAt","UpdatedAt") OUTPUT INSERTED."ID" VALUES ('` + taskID + `',2,1,'PENDING',1001,'<time>','<time>','<time>');`,
		`INSERT INTO "ApprovalDET" ("ApprovalHeaderID","Level","ApproverRoleID","ApproverID","Status","Notes","ActedAt","SecureApprovalCode","CreatedAt","UpdatedAt") OUTPUT INSERTED."ID" VALUES (7,1,'DEPT_HEAD',3001,'PENDING','',NULL,'<code1>','<time>','<time>'),(7,2,'DIV_HEAD',3002,'WAITING','',NULL,'<code2>','<time>','<time>');`,
		`SELECT * FROM "Users" WHERE UserID = 3001 ORDER BY "Users"."UserID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT * FROM "Sites" WHERE SiteID = '` + siteID + `' ORDER BY "Sites"."SiteID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT * FROM "Users" WHERE UserID = 1001 ORDER BY "Users"."UserID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT * FROM "Roles" WHERE RoleID = 'DEPT_HEAD' ORDER BY "Roles"."RoleID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`UPDATE "Tasks" SET "TaskStatus"='WAITING_DEPT_HEAD',"UpdatedAt"='<time>',"UpdatedBy"=1001 WHERE TaskID = '` + taskID + `'`,
		`SELECT * FROM "Roles" WHERE RoleID = 'DEPT_HEAD' ORDER BY "Roles"."RoleID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`INSERT INTO "TaskHistories" ("TaskID","Action","ActorID","Notes","CreatedAt","ActorRole","Round") OUTPUT INSERTED."ID" VALUES ('` + taskID + `','SUBMIT',1001,'Approval dikirim ke Dept Head, status PENDING (catatan: Site diajukan oleh andi untuk proses approval)','<time>','USER',1);`,
	}, maskCodes(sqls[:25]))
	q := fdb.Queries()
	iBegin, iCommit := indexOf(q, "BEGIN"), indexOf(q, "COMMIT")
	assert.Less(t, iBegin, indexOf(q, `INSERT INTO "Sites"`))
	assert.Less(t, indexOf(q, `UPDATE "Tasks"`), iCommit, "submit runs inside the save transaction")
	assert.Less(t, iCommit, indexOf(q, `SELECT * FROM "SystemSettings"`), "AI insight only after the commit")

	// then the AI insight (new site, queue off: at once) and the task detail
	assert.Equal(t, `SELECT * FROM "SystemSettings" WHERE SettingKey = 'ai_insight.daily_quota' ORDER BY "SystemSettings"."ID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`, sqls[25])
	assert.Contains(t, sqls, `DELETE FROM "SiteAIInsights" WHERE SiteID = '`+siteID+`'`)
	assert.Equal(t, `SELECT * FROM "SiteAttachments" WHERE SiteID = '`+siteID+`' ORDER BY SiteAttachmentID ASC`, sqls[len(sqls)-1])

	// Redis: running IDs and the e-mail link code (24 h), same keys
	day := jakartaToday()
	v, _ := m.redis.Get("site-dev-api:test:counter:site:" + day + ":")
	assert.Equal(t, "1", v)
	assert.Equal(t, 48*time.Hour, m.redis.TTL("site-dev-api:test:counter:site:"+day+":"))
	v, _ = m.redis.Get("site-dev-api:test:counter:task:" + day + ":")
	assert.Equal(t, "1", v)
	require.Len(t, sentCode, 32)
	v, _ = m.redis.Get("site-dev-api:test:approval_code:" + sentCode + ":")
	assert.Equal(t, "31", v)
	assert.Equal(t, 24*time.Hour, m.redis.TTL("site-dev-api:test:approval_code:"+sentCode+":"))
}

// geminiFor answers like the real client: the insight of the given site.
func geminiFor(_ context.Context, in client.GeminiInput) (dto.AIInsightResponseGet, string, error) {
	insight := geminiInsight()
	insight.SiteID = in.Site.SiteID
	return insight, "gemini-2.5-flash", nil
}

var codeLiteral = regexp.MustCompile(`'[A-Za-z0-9]{32}'`)

// maskCodes replaces the two random approval codes with <code1>, <code2>.
func maskCodes(sqls []string) []string {
	out := make([]string, len(sqls))
	for i, s := range sqls {
		n := 0
		out[i] = codeLiteral.ReplaceAllStringFunc(s, func(string) string { n++; return fmt.Sprintf("'<code%d>'", n) })
	}
	return out
}

func indexOf(list []string, prefix string) int {
	for i, s := range list {
		if strings.HasPrefix(s, prefix) {
			return i
		}
	}
	return -1
}

func saveResults(taskStatus string) []sqlfake.Result {
	return append([]sqlfake.Result{
		{Match: `SELECT count(*) FROM "Sites"`, Columns: []string{"count"}, Rows: [][]driver.Value{{int64(1)}}},
		noRows(`FROM "SiteAIInsights"`),
		{Match: `FROM "Tasks" WHERE SiteID = @p1`, Columns: taskColumns, Rows: [][]driver.Value{{"TS1", "ST1", taskStatus, int64(1001), int64(1001), created, updated}}},
	}, detailResults("USER")...)
}

func TestSideBySide_SaveDraftUpdate(t *testing.T) {
	m := withFlow(t)
	m.gemini.On("GenerateInsight", mock.Anything, mock.Anything).Return(geminiInsight(), "gemini-2.5-flash", nil)
	h, rec, _ := stack(t, saveResults("REVISION")...)

	code, body := call(t, h, http.MethodPost, "/v1/task/DRAFT", map[string]any{"site": map[string]any{"site_id": "ST1", "site_name_1": "Toko Kemang"}, "notes": "foto diganti"}, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	sqls := stableSQL(rec.sqls)
	assert.Equal(t, `SELECT * FROM "Sites" WHERE SiteID = 'ST1' ORDER BY "Sites"."SiteID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`, sqls[1])
	assert.Equal(t, `UPDATE "Sites" SET "SiteAddress1"='',"SiteAddress2"='',"SiteAddress3"='',"SiteCity"='',"SiteDistrict"='',"SiteFax"='',"SiteLat"=0,"SiteLon"=0,"SiteMobilePhone"='',"SiteName1"='Toko Kemang',"SiteName2"='',"SiteOpen24Hours"=0,"SitePic"='',"SitePlannedOpeningDate"='<time>',"SitePostalCode"='',"SiteRegion"='',"SiteSalesArea"=0,"SiteSellingAlcohol"=0,"SiteTelepon"='',"SiteTeleponExt"='',"UpdatedAt"='<time>',"UpdatedBy"=1001 WHERE SiteID = 'ST1'`, sqls[3])
	assert.Equal(t, `INSERT INTO "TaskHistories" ("TaskID","Action","ActorID","Notes","CreatedAt","ActorRole","Round") OUTPUT INSERTED."ID" VALUES ('TS1','UPDATE',1001,'foto diganti','<time>','USER',1);`, sqls[5])
	// no attachment/context sent: those tables are not touched
	for _, s := range sqls {
		assert.NotContains(t, s, "SiteAttachments\" WHERE SiteID = 'ST1'\x00")
		assert.False(t, strings.HasPrefix(s, `DELETE FROM "SiteAttachments"`) || strings.HasPrefix(s, `DELETE FROM "SiteCompetitors"`), s)
	}
	// updated DRAFT without an insight yet: generated
	m.gemini.AssertNumberOfCalls(t, "GenerateInsight", 1)
	assert.Contains(t, string(body), `"site_id":"ST1","site_name_1":"Toko Kemang"`)
}

func TestSideBySide_SaveRefused(t *testing.T) {
	for _, tc := range []struct {
		name, path   string
		body         map[string]any
		user         int64
		taskStatus   string
		code         int
		want         string
		extraResults []sqlfake.Result
	}{
		{"bad status", "/v1/task/APPROVED", map[string]any{}, 1001, "DRAFT", http.StatusUnprocessableEntity,
			`{"code":422,"status":"failure","message":"Unprocessable Entity","errors":{"status (must be DRAFT or SUBMITED)":"The status (must be DRAFT or SUBMITED) is invalid value."}}`, nil},
		{"duplicate name", "/v1/task/DRAFT", map[string]any{"site": map[string]any{"site_name_1": "Toko Kemang"}}, 1001, "DRAFT", http.StatusBadRequest,
			`{"code":400,"status":"failure","message":"Bad Request","errors":{"error":"Site with name Toko Kemang already exists"}}`,
			[]sqlfake.Result{{Match: `WHERE SiteName1 = @p1`, Columns: siteColumns, Rows: [][]driver.Value{siteST1Row()}}}},
		{"already sent", "/v1/task/DRAFT", map[string]any{"site": map[string]any{"site_id": "ST1", "site_name_1": "Toko Kemang"}}, 1001, "WAITING_DEPT_HEAD", http.StatusBadRequest,
			`{"code":400,"status":"failure","message":"Bad Request","errors":{"error":"Data site sudah disubmit tidak bisa diedit kembali"}}`, nil},
		{"not creator (draft)", "/v1/task/DRAFT", map[string]any{"site": map[string]any{"site_id": "ST1", "site_name_1": "Toko Kemang"}}, 5005, "DRAFT", http.StatusForbidden,
			`{"code":403,"status":"failure","message":"Forbidden","errors":{"error":"Hanya pembuat site Toko Kemang yang bisa update"}}`, nil},
		{"not creator (submit)", "/v1/task/SUBMITED", map[string]any{"site": sendableSite()}, 5005, "DRAFT", http.StatusForbidden,
			`{"code":403,"status":"failure","message":"Forbidden","errors":{"error":"Hanya pembuat site Toko Kemang yang bisa submit"}}`, nil},
		{"submit twice", "/v1/task/SUBMITED", map[string]any{"site": sendableSite()}, 1001, "WAITING_DEPT_HEAD", http.StatusInternalServerError,
			`{"code":500,"status":"failure","message":"Internal Server Error","errors":{"error":"task_service: task TS1 cannot be submitted from status WAITING_DIV_HEAD"}}`, nil},
		{"unknown site", "/v1/task/DRAFT", map[string]any{"site": map[string]any{"site_id": "ST9", "site_name_1": "x"}}, 1001, "DRAFT", http.StatusInternalServerError,
			`{"code":500,"status":"failure","message":"Internal Server Error","errors":{"error":"Site ST9 record not found"}}`, nil},
	} {
		withFlow(t)
		results := append(tc.extraResults, userRow(5005, "eko", "", "USER"))
		h, _, fdb := stack(t, append(results, saveResults(tc.taskStatus)...)...)
		code, body := call(t, h, http.MethodPost, tc.path, tc.body, tc.user)
		assert.Equal(t, tc.code, code, tc.name)
		assertSameJSON(t, []byte(tc.want), body)
		if indexOf(fdb.Queries(), "BEGIN") >= 0 {
			assert.Contains(t, fdb.Queries(), "ROLLBACK", tc.name)
			assert.NotContains(t, fdb.Queries(), "COMMIT", tc.name)
		}
	}
}

func approveResults(level int64, totalLevel int64) []sqlfake.Result {
	return append([]sqlfake.Result{
		{Match: `FROM "ApprovalHDR" WHERE TaskID`, Columns: []string{"ID", "TaskID", "TotalLevel", "CurrentLevel", "OverallStatus", "SubmittedBy", "SubmittedAt"},
			Rows: [][]driver.Value{{int64(7), "TS1", totalLevel, level, "PENDING", int64(1001), updated}}},
		{Match: `WHERE ApprovalHeaderID = @p1 AND ApproverID = @p2 AND Status = @p3 ORDER BY "ApprovalDET"."ID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY |7 |3001 |PENDING`,
			Columns: detailCols, Rows: [][]driver.Value{{int64(31), int64(7), level, "DEPT_HEAD", int64(3001), "PENDING", "", nil, "CODE31", updated, updated}}},
		{Match: `FROM "ApprovalDET" WHERE ApprovalHeaderID = @p1 ORDER BY Level ASC`, Columns: detailCols, Rows: [][]driver.Value{
			{int64(31), int64(7), int64(1), "DEPT_HEAD", int64(3001), "PENDING", "", nil, "CODE31CODE31CODE31CODE31CODE3131", updated, updated},
			{int64(32), int64(7), int64(2), "DIV_HEAD", int64(3002), "WAITING", "", nil, "CODE32CODE32CODE32CODE32CODE3232", updated, updated},
		}},
		roleRow("DEPT_HEAD", "Dept Head", 1),
		roleRow("DIV_HEAD", "Div Head", 2),
		userRow(3002, "dodi", "dodi@example.com", "DIV_HEAD"),
	}, detailResults("DEPT_HEAD")...)
}

func TestSideBySide_ApproveForwardsToNextLevel(t *testing.T) {
	m := withFlow(t)
	m.mail.On("SendApprovalPending", mock.Anything, "dodi@example.com", "dodi", mock.MatchedBy(func(e client.ApprovalPendingEmail) bool {
		return e.TaskID == "TS1" && e.SiteName == "Toko Kemang" && e.SubmittedBy == "andi" && e.ApproverRole == "Div Head" && e.Notes == "ok" &&
			e.Code == "CODE32CODE32CODE32CODE32CODE3232"
	})).Return(nil)
	h, rec, fdb := stack(t, approveResults(1, 2)...)

	code, body := call(t, h, http.MethodPost, "/v1/task/TS1/approve", map[string]any{"decision": "APPROVE", "notes": "ok"}, 3001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, []byte(`{"code":200,"status":"success"}`), body)
	assert.Equal(t, []string{
		`SELECT * FROM "Tasks" WHERE TaskID = 'TS1' ORDER BY "Tasks"."TaskID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT * FROM "ApprovalHDR" WHERE TaskID = 'TS1' ORDER BY ID DESC OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT * FROM "ApprovalDET" WHERE ApprovalHeaderID = 7 AND ApproverID = 3001 AND Status = 'PENDING' ORDER BY "ApprovalDET"."ID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT * FROM "Users" WHERE UserID = 3001 ORDER BY "Users"."UserID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`UPDATE "ApprovalDET" SET "ActedAt"='<time>',"Notes"='ok',"Status"='APPROVED',"UpdatedAt"='<time>' WHERE ID = 31 AND Status = 'PENDING'`,
		`SELECT * FROM "Roles" WHERE RoleID = 'DEPT_HEAD' ORDER BY "Roles"."RoleID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`INSERT INTO "TaskHistories" ("TaskID","Action","ActorID","Notes","CreatedAt","ActorRole","Round") OUTPUT INSERTED."ID" VALUES ('TS1','APPROVE',3001,'Site di approve oleh citra (Dept Head) dengan catatan (ok)','<time>','DEPT_HEAD',1);`,
		`SELECT * FROM "ApprovalDET" WHERE ApprovalHeaderID = 7 ORDER BY Level ASC`,
		`UPDATE "ApprovalDET" SET "Status"='PENDING',"UpdatedAt"='<time>' WHERE ID = 32`,
		`UPDATE "Tasks" SET "TaskStatus"='WAITING_DIV_HEAD',"UpdatedAt"='<time>',"UpdatedBy"=3001 WHERE TaskID = 'TS1'`,
		`SELECT * FROM "Roles" WHERE RoleID = 'DIV_HEAD' ORDER BY "Roles"."RoleID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`INSERT INTO "TaskHistories" ("TaskID","Action","ActorID","Notes","CreatedAt","ActorRole","Round") OUTPUT INSERTED."ID" VALUES ('TS1','FORWARD',3001,'Approval diteruskan ke Div Head, status PENDING','<time>','DEPT_HEAD',1);`,
		`UPDATE "ApprovalHDR" SET "CurrentLevel"=2,"UpdatedAt"='<time>' WHERE ID = 7`,
		// after the commit: e-mail to the next approver
		`SELECT * FROM "Users" WHERE UserID = 3002 ORDER BY "Users"."UserID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT * FROM "Sites" WHERE SiteID = 'ST1' ORDER BY "Sites"."SiteID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT * FROM "Users" WHERE UserID = 1001 ORDER BY "Users"."UserID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT * FROM "Roles" WHERE RoleID = 'DIV_HEAD' ORDER BY "Roles"."RoleID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
	}, stableSQL(rec.sqls))
	q := fdb.Queries()
	assert.Equal(t, "COMMIT", q[len(q)-5], "the e-mail goes out after the commit")
	v, _ := m.redis.Get("site-dev-api:test:approval_code:CODE32CODE32CODE32CODE32CODE3232:")
	assert.Equal(t, "32", v)
}

func TestSideBySide_ApproveLastLevelCompletes(t *testing.T) {
	withFlow(t)
	h, rec, _ := stack(t, approveResults(2, 2)...)

	code, _ := call(t, h, http.MethodPost, "/v1/task/TS1/approve", map[string]any{"decision": "APPROVE"}, 3001)

	require.Equal(t, http.StatusOK, code)
	sqls := stableSQL(rec.sqls)
	assert.Equal(t, []string{
		`INSERT INTO "TaskHistories" ("TaskID","Action","ActorID","Notes","CreatedAt","ActorRole","Round") OUTPUT INSERTED."ID" VALUES ('TS1','APPROVE',3001,'Site di approve oleh citra (Dept Head)','<time>','DEPT_HEAD',1);`,
		`UPDATE "ApprovalHDR" SET "OverallStatus"='APPROVED',"UpdatedAt"='<time>' WHERE ID = 7`,
		`UPDATE "Tasks" SET "TaskStatus"='COMPLETED',"UpdatedAt"='<time>',"UpdatedBy"=3001 WHERE TaskID = 'TS1'`,
		`INSERT INTO "TaskHistories" ("TaskID","Action","ActorID","Notes","CreatedAt","ActorRole","Round") OUTPUT INSERTED."ID" VALUES ('TS1','COMPLETE',3001,'Task Complete','<time>','DEPT_HEAD',1);`,
	}, sqls[6:])
}

func TestSideBySide_ApproveRejectAndRevision(t *testing.T) {
	withFlow(t)
	h, rec, _ := stack(t, approveResults(1, 2)...)

	code, _ := call(t, h, http.MethodPost, "/v1/task/TS1/approve", map[string]any{"decision": "REJECT", "notes": "lokasi banjir"}, 3001)
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, []string{
		`UPDATE "ApprovalDET" SET "ActedAt"='<time>',"Notes"='lokasi banjir',"Status"='REJECTED',"UpdatedAt"='<time>' WHERE ID = 31 AND Status = 'PENDING'`,
		`SELECT * FROM "Roles" WHERE RoleID = 'DEPT_HEAD' ORDER BY "Roles"."RoleID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`INSERT INTO "TaskHistories" ("TaskID","Action","ActorID","Notes","CreatedAt","ActorRole","Round") OUTPUT INSERTED."ID" VALUES ('TS1','REJECT',3001,'Site di reject oleh citra (Dept Head) dengan catatan (lokasi banjir)','<time>','DEPT_HEAD',1);`,
		`UPDATE "ApprovalHDR" SET "OverallStatus"='REJECTED',"UpdatedAt"='<time>' WHERE ID = 7`,
		`UPDATE "Tasks" SET "TaskStatus"='REJECTED',"UpdatedAt"='<time>',"UpdatedBy"=3001 WHERE TaskID = 'TS1'`,
	}, stableSQL(rec.sqls)[4:])

	rec.sqls = nil
	code, _ = call(t, h, http.MethodPost, "/v1/task/TS1/approve", map[string]any{"decision": "REVISION", "notes": "foto kurang"}, 3001)
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, []string{
		`INSERT INTO "TaskHistories" ("TaskID","Action","ActorID","Notes","CreatedAt","ActorRole","Round") OUTPUT INSERTED."ID" VALUES ('TS1','REVISION',3001,'Site diminta revisi oleh citra (Dept Head) dengan catatan (foto kurang)','<time>','DEPT_HEAD',1);`,
		`UPDATE "Tasks" SET "TaskStatus"='REVISION',"UpdatedAt"='<time>',"UpdatedBy"=3001 WHERE TaskID = 'TS1'`,
	}, stableSQL(rec.sqls)[6:], "REVISION leaves the round PENDING, as in site-dev-api")
}

func TestSideBySide_ApproveRefused(t *testing.T) {
	withFlow(t)
	conflict := sqlfake.Result{Match: `UPDATE "ApprovalDET"`, Affected: sqlfake.Int64(0)}
	h, _, fdb := stack(t, append([]sqlfake.Result{conflict, noRows(`|7 |1001 |PENDING`), noRows(`|7 |1001 |HOLD`)}, approveResults(1, 2)...)...)

	code, body := call(t, h, http.MethodPost, "/v1/task/TS1/approve", map[string]any{"decision": "APPROVE"}, 3001)
	assert.Equal(t, http.StatusInternalServerError, code)
	assertSameJSON(t, []byte(`{"code":500,"status":"failure","message":"Internal Server Error","errors":{"error":"task_service: approval already acted on for task TS1 (conflict)"}}`), body)
	assert.Contains(t, fdb.Queries(), "ROLLBACK")

	code, body = call(t, h, http.MethodPost, "/v1/task/TS1/approve", map[string]any{"decision": "APPROVE"}, 1001)
	assert.Equal(t, http.StatusInternalServerError, code)
	assertSameJSON(t, []byte(`{"code":500,"status":"failure","message":"Internal Server Error","errors":{"error":"task_service: no pending approval for this user on task TS1: record not found"}}`), body)

	// DIFFERENCE (Wes, 26 Oct): REJECT / REVISION without notes are fine
	// now (site-dev-api: 500 "notes required"); too long notes still 422
	code, body = call(t, h, http.MethodPost, "/v1/task/TS1/approve", map[string]any{"decision": "REJECT", "notes": strings.Repeat("x", 501)}, 3001)
	assert.Equal(t, http.StatusUnprocessableEntity, code)
	assert.Contains(t, string(body), `"notes":"The notes may not be greater than 500"`)
	code, _ = call(t, h, http.MethodPost, "/v1/task/TS1/approve", map[string]any{"decision": "MAYBE"}, 3001)
	assert.Equal(t, http.StatusUnprocessableEntity, code)
}

func basicCall(t *testing.T, h http.Handler, path string, user, pass string, body string) (int, []byte) {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if user != "" {
		req.SetBasicAuth(user, pass)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr.Code, rr.Body.Bytes()
}

func TestSideBySide_QueueCallbacks(t *testing.T) {
	m := withFlow(t)
	jwtServer(t, http.StatusOK, `{}`) // sets the test public_api username/password
	h, _, _ := stack(t, detailResults("USER")...)

	for _, tc := range []struct{ user, pass, want string }{
		{"", "", `{"error":"Basic auth required"}`},
		{"other", "test-pass", `{"error":"User does not exist"}`},
		{"test-user", "wrong", `{"error":"Incorrect password"}`},
	} {
		code, body := basicCall(t, h, "/v1/queue/approval-email", tc.user, tc.pass, `{}`)
		assert.Equal(t, http.StatusUnauthorized, code)
		assert.JSONEq(t, tc.want, string(body)) // plain JSON, not the envelope (as in site-dev-api)
	}

	m.mail.On("SendApprovalPending", mock.Anything, "dodi@example.com", "dodi", client.ApprovalPendingEmail{
		TaskID: "TS1", SiteName: "Toko Kemang", SubmittedBy: "andi", ApproverRole: "Div Head", Notes: "ok", Code: "C"}).Return(nil)
	code, body := basicCall(t, h, "/v1/queue/approval-email", "test-user", "test-pass",
		`{"to":"dodi@example.com","to_name":"dodi","task_id":"TS1","site_name":"Toko Kemang","submitted_by":"andi","approver_role":"Div Head","notes":"ok","approval_code":"C"}`)
	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, []byte(`{"code":200,"status":"success"}`), body)

	m.gemini.On("GenerateInsight", mock.Anything, mock.Anything).Return(geminiInsight(), "gemini-2.5-flash", nil)
	code, body = basicCall(t, h, "/v1/queue/ai-insight", "test-user", "test-pass", `{"site_id":"ST1","actor_id":1001}`)
	require.Equal(t, http.StatusOK, code, string(body))
	v, _ := m.redis.Get(quotaKey("ai_insight", 1001))
	assert.Equal(t, "49", v, "the job counts against the user who saved the site")

	m.redis.Set("site-dev-api:test:counter:ai_insight_quota:5:20261020:", "3")
	code, body = basicCall(t, h, "/v1/queue/ai-insight-quota-flush", "test-user", "test-pass", ``)
	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, []byte(`{"code":200,"status":"success","data":{"deleted":2}}`), body)
}
