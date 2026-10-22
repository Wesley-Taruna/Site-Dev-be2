package router

// November scope, API-CHANGES §12: GET /v1/competitor-brands. New endpoint;
// shape from site-dev-mobile src/lib/brands.ts.

import (
	"database/sql/driver"
	"net/http"
	"testing"

	"site-dev-backend/internal/pkg/sqlfake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompetitorBrands(t *testing.T) {
	h, rec, _ := stack(t,
		sqlfake.Result{Match: `FROM "CompetitorBrands"`, Columns: []string{"Slug", "Name", "Patterns", "LogoPath", "Color", "Type", "LegacyType", "Active", "IsPlaceholder", "Sort"},
			Rows: [][]driver.Value{
				{"alfamidi", "Alfamidi", `["alfa\\s*midi"]`, "brands/alfamidi.png", "#e30613", "minimarket", "Mini Market", true, false, int64(2)},
				{"yomart", "Yomart", `not json`, "", "#00a650", "minimarket", "Mini Market", false, true, int64(8)},
			}},
	)

	code, body := call(t, h, http.MethodGet, "/v1/competitor-brands", nil, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, []byte(`{"code":200,"status":"success","data":[
		{"slug":"alfamidi","name":"Alfamidi","patterns":["alfa\\s*midi"],"logo_url":"brands/alfamidi.png","color":"#e30613","type":"minimarket","legacy_type":"Mini Market","active":true,"placeholder":false,"sort":2},
		{"slug":"yomart","name":"Yomart","patterns":[],"logo_url":"","color":"#00a650","type":"minimarket","legacy_type":"Mini Market","active":false,"placeholder":true,"sort":8}]}`), body)
	assert.Equal(t, []string{`SELECT * FROM "CompetitorBrands" ORDER BY Sort ASC, Slug ASC`}, rec.sqls)

	code, _ = call(t, h, http.MethodGet, "/v1/competitor-brands", nil, 0)
	assert.Equal(t, http.StatusForbidden, code, "login needed")
}
