package router

// Side-by-side check, module 2 (task reads B + dashboard): same JSON bytes
// and same SQL as site-dev-api 20eeab4 for GET /v1/task/:id,
// /:id/attachment, /status/count, /site-list, /pending-approvals,
// /approval-history and GET /v1/dashboard/task/history, /status/count.

import (
	"database/sql/driver"
	"net/http"
	"strings"
	"testing"
	"time"

	"site-dev-backend/internal/pkg/sqlfake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- copies of site-dev-api 20eeab4 internal/app/dto (unchanged) ---------

const oldDateOnlyLayout = "2006-01-02"

type oldDateOnly struct {
	time.Time
}

func (d oldDateOnly) MarshalJSON() ([]byte, error) {
	if d.Time.IsZero() {
		return []byte(`""`), nil
	}
	return []byte(`"` + d.Time.Format(oldDateOnlyLayout) + `"`), nil
}

type oldSiteResp struct {
	SiteID                 string      `json:"site_id"`
	SiteName1              string      `json:"site_name_1"`
	SiteName2              string      `json:"site_name_2"`
	SiteAddress1           string      `json:"site_address_1"`
	SiteAddress2           string      `json:"site_address_2"`
	SiteAddress3           string      `json:"site_address_3"`
	SitePostalCode         string      `json:"site_postal_code"`
	SiteDistrict           string      `json:"site_district"`
	SiteCity               string      `json:"site_city"`
	SiteRegion             string      `json:"site_region"`
	SiteLat                float64     `json:"site_lat"`
	SiteLon                float64     `json:"site_lon"`
	SiteTelepon            string      `json:"site_telepon"`
	SiteTeleponExt         string      `json:"site_telepon_ext"`
	SiteMobilePhone        string      `json:"site_mobile_phone"`
	SitePic                string      `json:"site_pic"`
	SiteFax                string      `json:"site_fax"`
	SitePlannedOpeningDate oldDateOnly `json:"site_planned_opening_date"`
	SiteSalesArea          float64     `json:"site_sales_area"`
	SiteSellingAlcohol     bool        `json:"site_selling_alcohol"`
	SiteOpen24Hours        bool        `json:"site_open_24_hours"`
	CreatedBy              int64       `json:"created_by"`
	CreatedByName          string      `json:"created_by_name,omitempty"`
	UpdatedBy              *int64      `json:"updated_by"`
	UpdatedByName          string      `json:"updated_by_name,omitempty"`
	CreatedAt              time.Time   `json:"created_at"`
	UpdatedAt              *time.Time  `json:"updated_at"`
}

type oldCompetitorResp struct {
	ID             uint   `json:"id"`
	CompetitorName string `json:"competitor_name"`
	CompetitorType string `json:"competitor_type"`
	Distance       string `json:"distance"`
}

type oldCustomNoteResp struct {
	ID      uint   `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type oldSiteContextResp struct {
	SiteID                 string              `json:"site_id"`
	FootTrafficObservation string              `json:"foot_traffic_observation"`
	PeakHours              string              `json:"peak_hours"`
	ParkingAvailability    string              `json:"parking_availability"`
	PublicTransport        string              `json:"public_transport"`
	AnchorLandmark         string              `json:"anchor_landmark"`
	ObservedDemographics   string              `json:"observed_demographics"`
	Competitors            []oldCompetitorResp `json:"competitors"`
	CustomNotes            []oldCustomNoteResp `json:"custom_notes"`
	CreatedAt              time.Time           `json:"created_at"`
	UpdatedAt              time.Time           `json:"updated_at"`
}

type oldTaskFullDetailItemApproval struct {
	ID             uint       `json:"id"`
	Level          int        `json:"level"`
	ApproverRoleID string     `json:"approver_role_id"`
	ApproverID     int64      `json:"approver_id"`
	ApproverName   string     `json:"approver_name"`
	Status         string     `json:"status"`
	Notes          string     `json:"notes"`
	CreatedAt      time.Time  `json:"created_at"`
	ActedAt        *time.Time `json:"acted_at"`
}

type oldTaskFullDetailAttachment struct {
	ID         uint      `json:"id"`
	FileName   string    `json:"file_name"`
	FileURL    string    `json:"file_url"`
	UploadedBy int64     `json:"uploaded_by"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type oldTaskFullDetailTaskSection struct {
	TaskID     string                          `json:"task_id"`
	TaskStatus string                          `json:"task_status"`
	Approval   []oldTaskFullDetailItemApproval `json:"approval"`
	History    []oldTaskHistoryResp            `json:"history"`
}

type oldTaskFullDetailResp struct {
	Site       oldSiteResp                   `json:"site"`
	Context    *oldSiteContextResp           `json:"context"`
	Task       oldTaskFullDetailTaskSection  `json:"task"`
	Attachment []oldTaskFullDetailAttachment `json:"attachment"`
}

type oldTaskStatusCountResp struct {
	TaskStatus string `json:"task_status"`
	Count      int64  `json:"count"`
}

type oldPendingApprovalResp struct {
	ApprovalDetailID uint      `json:"approval_detail_id"`
	TaskID           string    `json:"task_id"`
	SiteID           string    `json:"site_id"`
	SiteName1        string    `json:"site_name_1"`
	SiteAddress1     string    `json:"site_address_1"`
	SiteAddress2     string    `json:"site_address_2"`
	SiteAddress3     string    `json:"site_address_3"`
	TaskStatus       string    `json:"task_status"`
	CreatedBy        int64     `json:"created_by"`
	CreatedByName    string    `json:"created_by_name"`
	Level            int       `json:"level"`
	SubmittedAt      time.Time `json:"submitted_at"`
	CreatedAt        time.Time `json:"created_at"`
}

type oldApprovalHistoryResp struct {
	ApprovalDetailID uint       `json:"approval_detail_id"`
	TaskID           string     `json:"task_id"`
	SiteID           string     `json:"site_id"`
	SiteName1        string     `json:"site_name_1"`
	SiteAddress1     string     `json:"site_address_1"`
	SiteAddress2     string     `json:"site_address_2"`
	SiteAddress3     string     `json:"site_address_3"`
	TaskStatus       string     `json:"task_status"`
	CreatedBy        int64      `json:"created_by"`
	CreatedByName    string     `json:"created_by_name"`
	Level            int        `json:"level"`
	Status           string     `json:"status"`
	Notes            string     `json:"notes"`
	ActedAt          *time.Time `json:"acted_at"`
	CreatedAt        time.Time  `json:"created_at"`
}

type oldDashboardTaskHistoryResp struct {
	ID         uint      `json:"id"`
	TaskID     string    `json:"task_id"`
	SiteID     string    `json:"site_id"`
	SiteName1  string    `json:"site_name_1"`
	TaskStatus string    `json:"task_status"`
	Action     string    `json:"action"`
	ActorID    int64     `json:"actor_id"`
	ActorName  string    `json:"actor_name"`
	Notes      string    `json:"notes"`
	CreatedAt  time.Time `json:"created_at"`
}

// ---------------------------------------------------------------------------

var (
	planned = time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC)
	acted   = time.Date(2026, 10, 4, 10, 30, 0, 0, time.UTC)
)

var siteColumns = []string{"SiteID", "SiteName1", "SiteName2", "SiteAddress1", "SiteAddress2", "SiteAddress3", "SitePostalCode",
	"SiteDistrict", "SiteCity", "SiteRegion", "SiteLat", "SiteLon", "SiteTelepon", "SiteTeleponExt", "SiteMobilePhone", "SitePic",
	"SiteFax", "SitePlannedOpeningDate", "SiteSalesArea", "SiteSellingAlcohol", "SiteOpen24Hours", "CreatedBy", "UpdatedBy", "CreatedAt", "UpdatedAt"}

func siteST1Row() []driver.Value {
	return []driver.Value{"ST1", "Toko Kemang", "Kemang 2", "Jl. Kemang Raya 1", "RT 01", "Jakarta Selatan", "12730",
		"Mampang", "Jakarta", "DKI", -6.26, 106.81, "021-1", "12", "0812", "Pak Joko",
		"", planned, 120.5, false, true, int64(1001), int64(2002), created, updated}
}

func oldSiteST1() oldSiteResp {
	updatedBy := int64(2002)
	updatedAt := updated
	return oldSiteResp{SiteID: "ST1", SiteName1: "Toko Kemang", SiteName2: "Kemang 2", SiteAddress1: "Jl. Kemang Raya 1",
		SiteAddress2: "RT 01", SiteAddress3: "Jakarta Selatan", SitePostalCode: "12730", SiteDistrict: "Mampang", SiteCity: "Jakarta",
		SiteRegion: "DKI", SiteLat: -6.26, SiteLon: 106.81, SiteTelepon: "021-1", SiteTeleponExt: "12", SiteMobilePhone: "0812",
		SitePic: "Pak Joko", SitePlannedOpeningDate: oldDateOnly{planned}, SiteSalesArea: 120.5, SiteOpen24Hours: true,
		CreatedBy: 1001, UpdatedBy: &updatedBy, CreatedAt: created, UpdatedAt: &updatedAt}
}

var taskColumns = []string{"TaskID", "SiteID", "TaskStatus", "CreatedBy", "UpdatedBy", "CreatedAt", "UpdatedAt"}

func detailResults(role string) []sqlfake.Result {
	return []sqlfake.Result{
		userRow(1001, "andi", "andi@example.com", role),
		userRow(2002, "budi", "", "USER"),
		userRow(3001, "citra", "", "DEPT_HEAD"),
		noRows(`|3002`), // approver no longer in Users: name ""
		{Match: `FROM "Tasks" WHERE SiteID = @p1 ORDER BY "Tasks"."TaskID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY |ST1`, Columns: taskColumns,
			Rows: [][]driver.Value{{"TS1", "ST1", "WAITING_DIV_HEAD", int64(1001), int64(3001), created, updated}}},
		{Match: `FROM "Tasks" WHERE TaskID = @p1 ORDER BY "Tasks"."TaskID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY |TS1`, Columns: taskColumns,
			Rows: [][]driver.Value{{"TS1", "ST1", "WAITING_DIV_HEAD", int64(1001), int64(3001), created, updated}}},
		{Match: `FROM "Sites" WHERE SiteID = @p1 ORDER BY "Sites"."SiteID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY |ST1`, Columns: siteColumns,
			Rows: [][]driver.Value{siteST1Row()}},
		{Match: `FROM "SiteContexts"`, Columns: []string{"SiteID", "FootTrafficObservation", "PeakHours", "ParkingAvailability", "PublicTransport",
			"AnchorLandmark", "ObservedDemographics", "CreatedBy", "UpdatedBy", "CreatedAt", "UpdatedAt"},
			Rows: [][]driver.Value{{"ST1", "ramai", "17-20", "LIMITED", "TransJakarta", "Masjid", "mahasiswa", int64(1001), int64(1001), created, updated}}},
		{Match: `FROM "SiteCompetitors"`, Columns: []string{"ID", "SiteID", "CompetitorName", "CompetitorType", "Distance", "CreatedAt"},
			Rows: [][]driver.Value{{int64(11), "ST1", "Indomaret", "minimarket", "50m", created}, {int64(12), "ST1", "Alfamart", "minimarket", "120m", created}}},
		{Match: `FROM "SiteCustomNotes"`, Columns: []string{"ID", "SiteID", "Title", "Content", "CreatedAt"},
			Rows: [][]driver.Value{{int64(21), "ST1", "Banjir", "tidak pernah", created}}},
		{Match: `FROM "TaskHistories"`, Columns: []string{"ID", "TaskID", "Action", "ActorID", "Notes", "CreatedAt"},
			Rows: [][]driver.Value{{int64(1), "TS1", "CREATE", int64(1001), "", created}, {int64(2), "TS1", "SUBMIT", int64(1001), "dikirim", updated}}},
		{Match: `FROM "ApprovalHDR"`, Columns: []string{"ID", "TaskID", "TotalLevel", "CurrentLevel", "OverallStatus", "SubmittedBy", "SubmittedAt", "CreatedAt", "UpdatedAt"},
			Rows: [][]driver.Value{{int64(7), "TS1", int64(2), int64(2), "PENDING", int64(1001), updated, updated, updated}}},
		{Match: `FROM "ApprovalDET"`, Columns: []string{"ID", "ApprovalHeaderID", "Level", "ApproverRoleID", "ApproverID", "Status", "Notes", "ActedAt", "SecureApprovalCode", "CreatedAt", "UpdatedAt"},
			Rows: [][]driver.Value{
				{int64(31), int64(7), int64(1), "DEPT_HEAD", int64(3001), "APPROVED", "ok", acted, "SECRETCODE1", updated, acted},
				{int64(32), int64(7), int64(2), "DIV_HEAD", int64(3002), "PENDING", "", nil, "SECRETCODE2", updated, acted},
			}},
		{Match: `FROM "SiteAttachments"`, Columns: []string{"SiteAttachmentID", "SiteID", "FileName", "Path", "CreatedAt"},
			Rows: [][]driver.Value{{int64(41), "ST1", "depan.jpg", "bucket/sites/ST1/depan.jpg", created}}},
	}
}

func oldDetailST1() oldTaskFullDetailResp {
	site := oldSiteST1()
	site.CreatedByName = "andi"
	site.UpdatedByName = "budi"
	actedAt := acted
	return oldTaskFullDetailResp{
		Site: site,
		Context: &oldSiteContextResp{SiteID: "ST1", FootTrafficObservation: "ramai", PeakHours: "17-20", ParkingAvailability: "LIMITED",
			PublicTransport: "TransJakarta", AnchorLandmark: "Masjid", ObservedDemographics: "mahasiswa",
			Competitors: []oldCompetitorResp{{ID: 11, CompetitorName: "Indomaret", CompetitorType: "minimarket", Distance: "50m"},
				{ID: 12, CompetitorName: "Alfamart", CompetitorType: "minimarket", Distance: "120m"}},
			CustomNotes: []oldCustomNoteResp{{ID: 21, Title: "Banjir", Content: "tidak pernah"}},
			CreatedAt:   created, UpdatedAt: updated},
		Task: oldTaskFullDetailTaskSection{
			TaskID: "TS1", TaskStatus: "WAITING_DIV_HEAD",
			Approval: []oldTaskFullDetailItemApproval{
				{ID: 31, Level: 1, ApproverRoleID: "DEPT_HEAD", ApproverID: 3001, ApproverName: "citra", Status: "APPROVED", Notes: "ok", CreatedAt: updated, ActedAt: &actedAt},
				{ID: 32, Level: 2, ApproverRoleID: "DIV_HEAD", ApproverID: 3002, Status: "PENDING", CreatedAt: updated},
			},
			History: []oldTaskHistoryResp{{ID: 1, Action: "CREATE", ActorID: 1001, CreatedAt: created}, {ID: 2, Action: "SUBMIT", ActorID: 1001, Notes: "dikirim", CreatedAt: updated}},
		},
		Attachment: []oldTaskFullDetailAttachment{{ID: 41, FileName: "depan.jpg", FileURL: "bucket/sites/ST1/depan.jpg", UploadedBy: 1001, UploadedAt: created}},
	}
}

// SQL site-dev-api runs for GET /v1/task/ST1 (middleware user lookup, then
// TaskService.GetTaskDetail).
var oldDetailSQL = []string{
	`SELECT * FROM "Users" WHERE UserID = 1001 ORDER BY "Users"."UserID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
	`SELECT * FROM "Tasks" WHERE SiteID = 'ST1' ORDER BY "Tasks"."TaskID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
	`SELECT * FROM "Sites" WHERE SiteID = 'ST1' ORDER BY "Sites"."SiteID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
	`SELECT * FROM "Users" WHERE UserID = 1001 ORDER BY "Users"."UserID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
	`SELECT * FROM "Users" WHERE UserID = 2002 ORDER BY "Users"."UserID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
	`SELECT * FROM "SiteContexts" WHERE SiteID = 'ST1' ORDER BY "SiteContexts"."SiteID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
	`SELECT * FROM "SiteCompetitors" WHERE SiteID = 'ST1' ORDER BY ID ASC`,
	`SELECT * FROM "SiteCustomNotes" WHERE SiteID = 'ST1' ORDER BY ID ASC`,
	`SELECT * FROM "TaskHistories" WHERE TaskID = 'TS1' ORDER BY CreatedAt ASC`,
	`SELECT * FROM "ApprovalHDR" WHERE TaskID = 'TS1' ORDER BY ID DESC OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
	`SELECT * FROM "ApprovalDET" WHERE ApprovalHeaderID = 7 ORDER BY Level ASC`,
	`SELECT * FROM "Users" WHERE UserID = 3001 ORDER BY "Users"."UserID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
	`SELECT * FROM "Users" WHERE UserID = 3002 ORDER BY "Users"."UserID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
	`SELECT * FROM "SiteAttachments" WHERE SiteID = 'ST1' ORDER BY SiteAttachmentID ASC`,
}

func TestSideBySide_TaskDetail_BySiteID(t *testing.T) {
	h, rec, _ := stack(t, detailResults("USER")...)

	code, body := call(t, h, http.MethodGet, "/v1/task/ST1", nil, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, oldDataEnvelope(t, oldDetailST1()), body)
	assert.Equal(t, oldDetailSQL, rec.sqls)
	assert.NotContains(t, string(body), "SECRETCODE", "approval codes never leave the server")
}

func TestSideBySide_TaskDetail_ByTaskID(t *testing.T) {
	h, rec, _ := stack(t, detailResults("DIV_HEAD")...)

	code, body := call(t, h, http.MethodGet, "/v1/task/TS1", nil, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, oldDataEnvelope(t, oldDetailST1()), body)
	assert.Equal(t, `SELECT * FROM "Tasks" WHERE TaskID = 'TS1' ORDER BY "Tasks"."TaskID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`, rec.sqls[1])
}

func TestSideBySide_TaskDetail_NoContextNoApproval(t *testing.T) {
	results := []sqlfake.Result{noRows(`FROM "SiteContexts"`), noRows(`FROM "ApprovalHDR"`), noRows(`FROM "SiteAttachments"`), noRows(`FROM "TaskHistories"`)}
	h, _, _ := stack(t, append(results, detailResults("ADMIN")...)...)

	code, body := call(t, h, http.MethodGet, "/v1/task/ST1", nil, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	want := oldDetailST1()
	want.Context = nil
	want.Task.Approval = nil
	want.Task.History = nil
	want.Attachment = nil
	// site-dev-api sends null (not []) for the empty lists here
	assertSameJSON(t, oldDataEnvelope(t, want), body)
	assert.Contains(t, string(body), `"context":null`)
	assert.Contains(t, string(body), `"approval":null,"history":null},"attachment":null`)
}

func TestSideBySide_TaskDetail_StaffOtherSite(t *testing.T) {
	results := []sqlfake.Result{userRow(1001, "andi", "", "USER")}
	results = append(results, detailResults("USER")...)
	h, _, _ := stack(t, results...)

	code, body := call(t, h, http.MethodGet, "/v1/task/ST1", nil, 5005)

	// 5005 is not in Users: site-dev-api's middleware answers 401 first
	assert.Equal(t, http.StatusUnauthorized, code)
	assertSameJSON(t, []byte(`{"code":401,"status":"failure","message":"Unauthorized","errors":{"error":"user not found"}}`), body)

	h, _, _ = stack(t, append([]sqlfake.Result{userRow(5005, "eko", "", "USER")}, detailResults("USER")...)...)
	code, body = call(t, h, http.MethodGet, "/v1/task/ST1", nil, 5005)

	assert.Equal(t, http.StatusForbidden, code)
	assertSameJSON(t, []byte(`{"code":403,"status":"failure","message":"Forbidden","errors":{"error":"Anda tidak punya akses ke task TS1"}}`), body)
}

func TestSideBySide_TaskDetail_UnknownIDKind(t *testing.T) {
	h, _, _ := stack(t, detailResults("ADMIN")...)

	code, body := call(t, h, http.MethodGet, "/v1/task/XX1", nil, 1001)

	assert.Equal(t, http.StatusInternalServerError, code)
	assertSameJSON(t, []byte(`{"code":500,"status":"failure","message":"Internal Server Error","errors":{"error":"id \"XX1\" tidak dikenali (harus task_id atau site_id)"}}`), body)
}

func TestSideBySide_TaskAttachment(t *testing.T) {
	h, rec, _ := stack(t, detailResults("USER")...)

	code, body := call(t, h, http.MethodGet, "/v1/task/ST1/attachment", nil, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, oldDataEnvelope(t, []oldTaskFullDetailAttachment{
		{ID: 41, FileName: "depan.jpg", FileURL: "bucket/sites/ST1/depan.jpg", UploadedBy: 1001, UploadedAt: created},
	}), body)
	// site-dev-api: middleware user lookup + site + attachments; the new
	// service skips the user lookup here (no role needed, plan §4 diff 2)
	assert.Equal(t, oldDetailSQL[2], rec.sqls[0])
	assert.Equal(t, oldDetailSQL[13], rec.sqls[1])
	assert.Len(t, rec.sqls, 2)

	h, _, _ = stack(t, append([]sqlfake.Result{noRows(`FROM "SiteAttachments"`)}, detailResults("USER")...)...)
	_, body = call(t, h, http.MethodGet, "/v1/task/ST1/attachment", nil, 1001)
	assert.Contains(t, string(body), `"data":[]`, "this list is [] when empty")
}

var countResult = sqlfake.Result{Match: `COUNT(1) as Count`, Columns: []string{"TaskStatus", "Count"},
	Rows: [][]driver.Value{{"DRAFT", int64(3)}, {"WAITING_DEPT_HEAD", int64(2)}}}

func TestSideBySide_TaskStatusCount(t *testing.T) {
	for _, tc := range []struct {
		role string
		sql  string
	}{
		{"USER", `SELECT TaskStatus, COUNT(1) as Count FROM "Tasks" WHERE CreatedBy = 1001 GROUP BY "TaskStatus"`},
		{"DIV_HEAD", `SELECT TaskStatus, COUNT(1) as Count FROM "Tasks" GROUP BY "TaskStatus"`},
	} {
		h, rec, _ := stack(t, countResult, userRow(1001, "andi", "", tc.role))

		code, body := call(t, h, http.MethodGet, "/v1/task/status/count", nil, 1001)

		require.Equal(t, http.StatusOK, code, string(body))
		assertSameJSON(t, oldDataEnvelope(t, []oldTaskStatusCountResp{{"DRAFT", 3}, {"WAITING_DEPT_HEAD", 2}}), body)
		assert.Equal(t, []string{oldDetailSQL[0], tc.sql}, rec.sqls, tc.role)
	}
}

func TestSideBySide_DashboardStatusCount(t *testing.T) {
	h, rec, _ := stack(t, countResult)

	code, body := call(t, h, http.MethodGet, "/v1/dashboard/task/status/count", nil, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, oldDataEnvelope(t, []oldTaskStatusCountResp{{"DRAFT", 3}, {"WAITING_DEPT_HEAD", 2}}), body)
	assert.Equal(t, []string{`SELECT TaskStatus, COUNT(1) as Count FROM "Tasks" GROUP BY "TaskStatus"`}, rec.sqls)
}

func TestSideBySide_SiteList(t *testing.T) {
	h, rec, _ := stack(t,
		sqlfake.Result{Match: `count(*)`, Columns: []string{"count"}, Rows: [][]driver.Value{{int64(41)}}},
		sqlfake.Result{Match: `FROM "Sites" WHERE SiteID LIKE`, Columns: []string{"SiteID"}, Rows: [][]driver.Value{{"ST1"}, {"ST2"}}},
		sqlfake.Result{Match: `FROM "Tasks"`, Columns: taskColumns, Rows: [][]driver.Value{{"TS1", "ST1", "COMPLETED", int64(1001), int64(3002), created, updated}}},
		sqlfake.Result{Match: `FROM "Sites"`, Columns: siteColumns, Rows: [][]driver.Value{siteST1Row()}},
	)

	code, body := call(t, h, http.MethodGet, "/v1/task/site-list?q=kem&page=3&sort_by=TaskStatus&sort_dir=-1", nil, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, oldListEnvelope(t, []oldTaskResp{
		{TaskID: "TS1", SiteID: "ST1", SiteName1: "Toko Kemang", SiteAddress1: "Jl. Kemang Raya 1", SiteAddress2: "RT 01", SiteAddress3: "Jakarta Selatan",
			TaskStatus: "COMPLETED", CreatedBy: 1001, UpdatedBy: 3002, CreatedAt: created, UpdatedAt: updated},
	}, 41, 3, 0), body)
	statuses := `('DRAFT','SUBMITED','WAITING_DEPT_HEAD','WAITING_DIV_HEAD','REJECTED','REVISION','COMPLETED')`
	assert.Equal(t, []string{
		`SELECT "SiteID" FROM "Sites" WHERE SiteID LIKE '%kem%' OR SiteName1 LIKE '%kem%' OR SiteName2 LIKE '%kem%' OR SiteAddress1 LIKE '%kem%' OR SiteAddress2 LIKE '%kem%' OR SiteAddress3 LIKE '%kem%'`,
		`SELECT count(*) FROM "Tasks" WHERE TaskStatus IN ` + statuses + ` AND SiteID IN ('ST1','ST2')`,
		`SELECT * FROM "Tasks" WHERE TaskStatus IN ` + statuses + ` AND SiteID IN ('ST1','ST2') ORDER BY TaskStatus DESC OFFSET 40 ROWS FETCH NEXT 20 ROWS ONLY`,
		`SELECT * FROM "Sites" WHERE SiteID IN ('ST1')`,
	}, rec.sqls)
}

func approverResults() []sqlfake.Result {
	return []sqlfake.Result{
		{Match: `SELECT count(*) FROM "ApprovalDET"`, Columns: []string{"count"}, Rows: [][]driver.Value{{int64(1)}}},
		{Match: `SELECT ApprovalDET.* FROM "ApprovalDET"`, Columns: []string{"ID", "ApprovalHeaderID", "Level", "ApproverRoleID", "ApproverID", "Status", "Notes", "ActedAt", "SecureApprovalCode", "CreatedAt", "UpdatedAt"},
			Rows: [][]driver.Value{{int64(31), int64(7), int64(1), "DEPT_HEAD", int64(3001), "REVISION", "foto kurang", acted, "SECRETCODE1", updated, acted}}},
		{Match: `FROM "ApprovalHDR" WHERE ID = @p1`, Columns: []string{"ID", "TaskID", "TotalLevel", "CurrentLevel", "OverallStatus", "SubmittedBy", "SubmittedAt", "CreatedAt", "UpdatedAt"},
			Rows: [][]driver.Value{{int64(7), "TS1", int64(2), int64(1), "PENDING", int64(1001), updated, updated, updated}}},
	}
}

func TestSideBySide_PendingApprovals(t *testing.T) {
	h, rec, _ := stack(t, append(approverResults(), detailResults("DEPT_HEAD")...)...)

	code, body := call(t, h, http.MethodGet, "/v1/task/pending-approvals?q=Kemang&sort_by=Level&sort_dir=desc", nil, 3001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, oldListEnvelope(t, []oldPendingApprovalResp{{
		ApprovalDetailID: 31, TaskID: "TS1", SiteID: "ST1", SiteName1: "Toko Kemang", SiteAddress1: "Jl. Kemang Raya 1", SiteAddress2: "RT 01",
		SiteAddress3: "Jakarta Selatan", TaskStatus: "WAITING_DIV_HEAD", CreatedBy: 1001, CreatedByName: "andi", Level: 1, SubmittedAt: updated, CreatedAt: created,
	}}, 1, 1, 0), body)
	joins := `FROM "ApprovalDET" JOIN ApprovalHDR ON ApprovalHDR.ID = ApprovalDET.ApprovalHeaderID JOIN Tasks ON Tasks.TaskID = ApprovalHDR.TaskID JOIN Sites ON Sites.SiteID = Tasks.SiteID WHERE (ApprovalDET.ApproverID = 3001 AND ApprovalDET.Status = 'PENDING') AND Sites.SiteName1 LIKE '%Kemang%'`
	assert.Equal(t, []string{
		`SELECT count(*) ` + joins,
		`SELECT ApprovalDET.* ` + joins + ` ORDER BY ApprovalDET.Level DESC OFFSET 0 ROW FETCH NEXT 20 ROWS ONLY`,
		`SELECT * FROM "ApprovalHDR" WHERE ID = 7 ORDER BY "ApprovalHDR"."ID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT * FROM "Tasks" WHERE TaskID = 'TS1' ORDER BY "Tasks"."TaskID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		oldDetailSQL[2],
		oldDetailSQL[0],
	}, rec.sqls)
}

func TestSideBySide_ApprovalHistory(t *testing.T) {
	h, rec, _ := stack(t, append(approverResults(), detailResults("DEPT_HEAD")...)...)

	code, body := call(t, h, http.MethodGet, "/v1/task/approval-history?page=2", nil, 3001)

	require.Equal(t, http.StatusOK, code, string(body))
	actedAt := acted
	assertSameJSON(t, oldListEnvelope(t, []oldApprovalHistoryResp{{
		ApprovalDetailID: 31, TaskID: "TS1", SiteID: "ST1", SiteName1: "Toko Kemang", SiteAddress1: "Jl. Kemang Raya 1", SiteAddress2: "RT 01",
		SiteAddress3: "Jakarta Selatan", TaskStatus: "WAITING_DIV_HEAD", CreatedBy: 1001, CreatedByName: "andi", Level: 1, Status: "REVISION",
		Notes: "foto kurang", ActedAt: &actedAt, CreatedAt: created,
	}}, 1, 2, 0), body)
	joins := `FROM "ApprovalDET" JOIN ApprovalHDR ON ApprovalHDR.ID = ApprovalDET.ApprovalHeaderID JOIN Tasks ON Tasks.TaskID = ApprovalHDR.TaskID JOIN Sites ON Sites.SiteID = Tasks.SiteID WHERE ApprovalDET.ApproverID = 3001 AND ApprovalDET.Status IN ('APPROVED','REJECTED','REVISION')`
	assert.Equal(t, `SELECT count(*) `+joins, rec.sqls[0])
	assert.Equal(t, `SELECT ApprovalDET.* `+joins+` ORDER BY ApprovalDET.ActedAt ASC OFFSET 20 ROWS FETCH NEXT 20 ROWS ONLY`, rec.sqls[1])
}

func TestSideBySide_DashboardHistory(t *testing.T) {
	h, rec, _ := stack(t,
		sqlfake.Result{Match: `SELECT count(*) FROM "TaskHistories"`, Columns: []string{"count"}, Rows: [][]driver.Value{{int64(3)}}},
		sqlfake.Result{Match: `FROM "TaskHistories"`, Columns: []string{"ID", "TaskID", "Action", "ActorID", "Notes", "CreatedAt"},
			Rows: [][]driver.Value{
				{int64(9), "TS1", "APPROVE", int64(3001), "ok", acted},
				{int64(8), "TS1", "SUBMIT", int64(1001), "", updated},
				{int64(5), "TS9", "CREATE", int64(4004), "", created},
			}},
		sqlfake.Result{Match: `FROM "Tasks" WHERE TaskID IN`, Columns: taskColumns, Rows: [][]driver.Value{{"TS1", "ST1", "WAITING_DIV_HEAD", int64(1001), int64(3001), created, updated}}},
		sqlfake.Result{Match: `FROM "Sites"`, Columns: siteColumns, Rows: [][]driver.Value{siteST1Row()}},
		userRow(3001, "citra", "", "DEPT_HEAD"),
		userRow(1001, "andi", "", "USER"),
	)

	code, body := call(t, h, http.MethodGet, "/v1/dashboard/task/history?sort_dir=desc", nil, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, oldListEnvelope(t, []oldDashboardTaskHistoryResp{
		{ID: 9, TaskID: "TS1", SiteID: "ST1", SiteName1: "Toko Kemang", TaskStatus: "WAITING_DIV_HEAD", Action: "APPROVE", ActorID: 3001, ActorName: "citra", Notes: "ok", CreatedAt: acted},
		{ID: 8, TaskID: "TS1", SiteID: "ST1", SiteName1: "Toko Kemang", TaskStatus: "WAITING_DIV_HEAD", Action: "SUBMIT", ActorID: 1001, ActorName: "andi", CreatedAt: updated},
		{ID: 5, TaskID: "TS9", Action: "CREATE", ActorID: 4004, CreatedAt: created},
	}, 3, 1, 0), body)
	// site-dev-api also ran `SELECT count(*) FROM "Tasks" WHERE TaskID IN
	// (...)` (its generic FindAll) and threw the number away; not repeated.
	assert.Equal(t, []string{
		`SELECT count(*) FROM "TaskHistories"`,
		`SELECT * FROM "TaskHistories" ORDER BY CreatedAt DESC OFFSET 0 ROW FETCH NEXT 20 ROWS ONLY`,
		`SELECT * FROM "Tasks" WHERE TaskID IN ('TS1','TS9')`,
		`SELECT * FROM "Sites" WHERE SiteID IN ('ST1')`,
		`SELECT * FROM "Users" WHERE UserID = 3001 ORDER BY "Users"."UserID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT * FROM "Users" WHERE UserID = 1001 ORDER BY "Users"."UserID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT * FROM "Users" WHERE UserID = 4004 ORDER BY "Users"."UserID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
	}, rec.sqls)
	assert.True(t, strings.Contains(string(body), `"actor_name":""`))
}
