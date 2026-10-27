package router

// New site ID "SD27/10/2026001" (Wes, 27 Oct) in URL paths: the twin
// "SD27-10-2026001" and the "%2F"-encoded form both reach the service as
// the stored ID; a raw "/" is not one path segment (404, route not found).

import (
	"net/http"
	"strings"
	"testing"

	"site-dev-backend/internal/app/constants"

	"github.com/stretchr/testify/assert"
)

const newSiteID = "SD27/10/2026001"

func TestSiteIDInPath_TwinAndEncoded(t *testing.T) {
	routes := []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, "/v1/task/ai-insight/%s", nil},
		{http.MethodGet, "/v1/task/%s", nil},
		{http.MethodGet, "/v1/task/%s/history", nil},
		{http.MethodGet, "/v1/task/%s/attachment", nil},
		{http.MethodPut, "/v1/task/context/%s", map[string]any{"peak_hours": "17-20"}},
		{http.MethodPost, "/v1/task/%s/cancel", map[string]any{}},
		{http.MethodPost, "/v1/task/%s/approve", map[string]any{"decision": "APPROVE"}},
		{http.MethodGet, "/v1/site/%s", nil},
		{http.MethodGet, "/v1/site/%s/context", nil},
		{http.MethodDelete, "/v1/site/%s", nil},
		{http.MethodGet, "/v1/portal/site/%s/report", nil},
	}
	forms := map[string]string{
		"twin":          "SD27-10-2026001",
		"encoded":       "SD27%2F10%2F2026001",
		"encoded lower": "SD27%2f10%2f2026001",
	}
	for _, r := range routes {
		for name, form := range forms {
			t.Run(r.method+" "+r.path+" "+name, func(t *testing.T) {
				h, rec, _ := stack(t, userRow(1001, "andi", "andi@example.com", "ADMIN"))
				path := strings.Replace(r.path, "%s", form, 1)

				code, body := call(t, h, r.method, path, r.body, 1001)

				assert.True(t, sqlMentions(rec.sqls, "'"+newSiteID+"'"), "%s %s → %d %s; SQL: %v", r.method, path, code, body, rec.sqls)
				assert.NotContains(t, string(body), "SD27-10", "the twin never reaches the answer")
			})
		}
	}
}

func TestSiteIDInPath_RawSlashIsNoRoute(t *testing.T) {
	h, rec, _ := stack(t, userRow(1001, "andi", "andi@example.com", "ADMIN"))

	code, _ := call(t, h, http.MethodGet, "/v1/site/"+newSiteID, nil, 1001)

	assert.Equal(t, http.StatusNotFound, code)
	assert.False(t, sqlMentions(rec.sqls, "SD27"))
}

func TestSiteIDInQueryAndBody(t *testing.T) {
	withRedis(t)
	h, rec, _ := stack(t, userRow(1001, "andi", "andi@example.com", "USER"))

	call(t, h, http.MethodGet, "/v1/task?site_id=SD27-10-2026001", nil, 1001)
	call(t, h, http.MethodGet, "/v1/task?site_id=SD27%2F10%2F2026001", nil, 1001)
	call(t, h, http.MethodPost, "/v1/task/ai-insight", map[string]any{"site_id": "SD27-10-2026001"}, 1001)

	n := 0
	for _, s := range rec.sqls {
		if strings.Contains(s, "'"+newSiteID+"'") {
			n++
		}
	}
	assert.GreaterOrEqual(t, n, 3, "%v", rec.sqls)
}

func TestSiteIDPathHelpers(t *testing.T) {
	assert.Equal(t, "SD27-10-2026001", constants.SiteIDForPath(newSiteID))
	assert.Equal(t, newSiteID, constants.SiteIDFromPath("SD27-10-2026001"))
	assert.Equal(t, "SD27/10/20261000", constants.SiteIDFromPath("SD27-10-20261000"))
	assert.Equal(t, newSiteID, constants.SiteIDFromPath("SD27%2F10%2F2026001"))
	assert.Equal(t, newSiteID, constants.SiteIDFromPath(newSiteID))
	for _, old := range []string{"ST20261027000001", "TS20261027000001", "ST1", ""} {
		assert.Equal(t, old, constants.SiteIDFromPath(old))
		assert.Equal(t, old, constants.SiteIDForPath(old))
	}
	assert.Equal(t, "SD-1-2", constants.SiteIDFromPath("SD-1-2"), "not a twin: unchanged")
	assert.True(t, constants.IsSiteID(newSiteID))
	assert.True(t, constants.IsSiteID("ST20261027000001"))
	assert.False(t, constants.IsSiteID("TS20261027000001"))
}

func sqlMentions(sqls []string, s string) bool {
	for _, q := range sqls {
		if strings.Contains(q, s) {
			return true
		}
	}
	return false
}
