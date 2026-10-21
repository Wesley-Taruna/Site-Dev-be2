package router

// Side-by-side check, module 6b: PUT /v1/task/context/:site_id against
// site-dev-api 20eeab4.

import (
	"database/sql/driver"
	"net/http"
	"testing"

	"site-dev-backend/internal/pkg/sqlfake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func contextUpdateResults(status string, contextRows int64) []sqlfake.Result {
	return []sqlfake.Result{
		userRow(1001, "andi", "andi@example.com", "USER"),
		userRow(5005, "eko", "", "USER"),
		{Match: `FROM "Sites" WHERE SiteID = @p1 ORDER BY "Sites"."SiteID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY |ST1`, Columns: siteColumns, Rows: [][]driver.Value{siteST1Row()}},
		{Match: `FROM "Tasks" WHERE SiteID = @p1`, Columns: taskColumns, Rows: [][]driver.Value{{"TS1", "ST1", status, int64(1001), int64(1001), created, updated}}},
		{Match: `SELECT count(*) FROM "SiteContexts"`, Columns: []string{"count"}, Rows: [][]driver.Value{{contextRows}}},
		{Match: `INSERT INTO "SiteCompetitors"`, Columns: []string{"ID"}, Rows: [][]driver.Value{{int64(51)}, {int64(52)}}},
		{Match: `INSERT INTO "SiteCustomNotes"`, Columns: []string{"ID"}, Rows: [][]driver.Value{{int64(61)}}},
	}
}

var contextBody = map[string]any{
	"foot_traffic_observation": "ramai sore", "peak_hours": "17-20", "parking_availability": "", "public_transport": "KRL",
	"anchor_landmark": "Pasar", "observed_demographics": "keluarga",
	"competitors":  []map[string]any{{"competitor_name": "Indomaret", "competitor_type": "minimarket", "distance": "50m"}, {"competitor_name": "Alfamart"}},
	"custom_notes": []map[string]any{{"title": "Banjir", "content": "tidak"}},
}

func TestSideBySide_UpdateContext(t *testing.T) {
	h, rec, fdb := stack(t, contextUpdateResults("REVISION", 1)...)

	code, body := call(t, h, http.MethodPut, "/v1/task/context/ST1", contextBody, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	// created_at/updated_at are time.Now(): compare everything else
	assert.Contains(t, string(body), `"data":{"site_id":"ST1","foot_traffic_observation":"ramai sore","peak_hours":"17-20","parking_availability":"","public_transport":"KRL","anchor_landmark":"Pasar","observed_demographics":"keluarga","competitors":[{"id":51,"competitor_name":"Indomaret","competitor_type":"minimarket","distance":"50m"},{"id":52,"competitor_name":"Alfamart","competitor_type":"","distance":""}],"custom_notes":[{"id":61,"title":"Banjir","content":"tidak"}],"created_at":`)
	assert.Equal(t, []string{
		oldDetailSQL[0],
		oldDetailSQL[2],
		`SELECT * FROM "Tasks" WHERE SiteID = 'ST1' ORDER BY "Tasks"."TaskID" OFFSET 0 ROW FETCH NEXT 1 ROWS ONLY`,
		`SELECT count(*) FROM "SiteContexts" WHERE SiteID = 'ST1'`,
		`UPDATE "SiteContexts" SET "AnchorLandmark"='Pasar',"FootTrafficObservation"='ramai sore',"ObservedDemographics"='keluarga',"ParkingAvailability"=NULL,"PeakHours"='17-20',"PublicTransport"='KRL',"UpdatedAt"='<time>',"UpdatedBy"=1001 WHERE SiteID = 'ST1'`,
		`DELETE FROM "SiteCompetitors" WHERE SiteID = 'ST1'`,
		`INSERT INTO "SiteCompetitors" ("SiteID","CompetitorName","CompetitorType","Distance","CreatedAt") OUTPUT INSERTED."ID" VALUES ('ST1','Indomaret','minimarket','50m','<time>'),('ST1','Alfamart','','','<time>');`,
		`DELETE FROM "SiteCustomNotes" WHERE SiteID = 'ST1'`,
		`INSERT INTO "SiteCustomNotes" ("SiteID","Title","Content","CreatedAt") OUTPUT INSERTED."ID" VALUES ('ST1','Banjir','tidak','<time>');`,
		`INSERT INTO "TaskHistories" ("TaskID","Action","ActorID","Notes","CreatedAt","ActorRole","Round") OUTPUT INSERTED."ID" VALUES ('TS1','UPDATE',1001,'Konteks on-site diperbarui oleh andi','<time>','USER',1);`,
	}, stableSQL(rec.sqls))
	q := fdb.Queries()
	assert.Equal(t, "BEGIN", q[3], "the four writes are one transaction")
	assert.Equal(t, "COMMIT", q[len(q)-1])
}

func TestSideBySide_UpdateContext_FirstTime(t *testing.T) {
	h, rec, _ := stack(t, contextUpdateResults("DRAFT", 0)...)

	code, body := call(t, h, http.MethodPut, "/v1/task/context/ST1", map[string]any{"parking_availability": "NONE"}, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	assert.Contains(t, string(body), `"parking_availability":"NONE","public_transport":"","anchor_landmark":"","observed_demographics":"","competitors":null,"custom_notes":null`)
	assert.Contains(t, stableSQL(rec.sqls), `INSERT INTO "SiteContexts" ("SiteID","FootTrafficObservation","PeakHours","ParkingAvailability","PublicTransport","AnchorLandmark","ObservedDemographics","CreatedBy","UpdatedBy","CreatedAt","UpdatedAt") VALUES ('ST1','','','NONE','','','',1001,1001,'<time>','<time>');`)
	assert.Contains(t, rec.sqls, `DELETE FROM "SiteCompetitors" WHERE SiteID = 'ST1'`, "lists are cleared even when none are sent")
}

func TestSideBySide_UpdateContext_Refused(t *testing.T) {
	for _, tc := range []struct {
		name, status string
		user         int64
		code         int
		body         string
	}{
		{"not the creator", "DRAFT", 5005, http.StatusForbidden, `{"code":403,"status":"failure","message":"Forbidden","errors":{"error":"Hanya pembuat site Toko Kemang yang bisa edit konteks"}}`},
		{"already submitted", "WAITING_DEPT_HEAD", 1001, http.StatusBadRequest, `{"code":400,"status":"failure","message":"Bad Request","errors":{"error":"Task sudah disubmit, konteks tidak bisa diedit lagi"}}`},
	} {
		h, rec, fdb := stack(t, contextUpdateResults(tc.status, 1)...)
		code, body := call(t, h, http.MethodPut, "/v1/task/context/ST1", contextBody, tc.user)
		assert.Equal(t, tc.code, code, tc.name)
		assertSameJSON(t, []byte(tc.body), body)
		assert.NotContains(t, fdb.Queries(), "BEGIN", tc.name)
		_ = rec
	}

	h, _, _ := stack(t, contextUpdateResults("DRAFT", 1)...)
	code, body := call(t, h, http.MethodPut, "/v1/task/context/ST1", map[string]any{"parking_availability": "MAYBE", "competitors": []map[string]any{{"distance": "1km"}}}, 1001)
	// DIFFERENCE (boilerplate binder): site-dev-api 500 with go-playground text
	assert.Equal(t, http.StatusUnprocessableEntity, code)
	assert.Contains(t, string(body), `"parking_availability":"The selected parking availability is invalid."`)
	assert.Contains(t, string(body), `"competitors.0.competitor_name":"The competitor name field is required."`)
}
