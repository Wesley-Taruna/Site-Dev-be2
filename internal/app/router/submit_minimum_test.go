package router

// POST /v1/task/SUBMITED needs only name, map point and sales area (Wes,
// 26 Oct); POST /v1/task/DRAFT stays as loose as before.

import (
	"net/http"
	"testing"

	"site-dev-backend/internal/app/client"
	dto "site-dev-backend/internal/app/dto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// sendableSite: the three fields a send needs, nothing else.
func sendableSite() map[string]any {
	return map[string]any{"site_id": "ST1", "site_name_1": "Toko Kemang", "site_lat": -6.26, "site_lon": 106.81, "site_sales_area": 120.5}
}

func TestSubmit_OnlyThreeFieldsNeeded(t *testing.T) {
	withFlow(t)
	for name, tc := range map[string]struct {
		site map[string]any
		want string
		data string
	}{
		"nothing": {map[string]any{"site_name_1": ""},
			"Belum bisa dikirim. Lengkapi dulu: Nama site, Lokasi (titik di peta), Sales Area.", `"data":["site_name_1","site_location","site_sales_area"]`},
		"name only spaces": {map[string]any{"site_name_1": "   ", "site_lat": -6.26, "site_lon": 106.81, "site_sales_area": 80},
			"Belum bisa dikirim. Lengkapi dulu: Nama site.", `"data":["site_name_1"]`},
		"no pin (0,0)": {map[string]any{"site_name_1": "Toko Kemang", "site_lat": 0, "site_lon": 0, "site_sales_area": 80},
			"Belum bisa dikirim. Lengkapi dulu: Lokasi (titik di peta).", `"data":["site_location"]`},
		"pin out of range": {map[string]any{"site_name_1": "Toko Kemang", "site_lat": -96.2, "site_lon": 106.81, "site_sales_area": 80},
			"Belum bisa dikirim. Lengkapi dulu: Lokasi (titik di peta).", `"data":["site_location"]`},
		"sales area 0": {map[string]any{"site_name_1": "Toko Kemang", "site_lat": -6.26, "site_lon": 106.81, "site_sales_area": 0},
			"Belum bisa dikirim. Lengkapi dulu: Sales Area.", `"data":["site_sales_area"]`},
		"sales area negative": {map[string]any{"site_name_1": "Toko Kemang", "site_lat": -6.26, "site_lon": 106.81, "site_sales_area": -5},
			"Belum bisa dikirim. Lengkapi dulu: Sales Area.", `"data":["site_sales_area"]`},
	} {
		h, rec, _ := stack(t)
		code, body := call(t, h, http.MethodPost, "/v1/task/SUBMITED", map[string]any{"site": tc.site}, 1001)
		assert.Equal(t, http.StatusBadRequest, code, name)
		assert.Contains(t, string(body), `"errors":{"error":"`+tc.want+`"}`, name)
		assert.Contains(t, string(body), tc.data, name)
		assert.Empty(t, rec.sqls, name+": nothing read or written")
	}

	// the three fields alone pass the check (no address, contacts, opening
	// date, survey, traffic, competitors or photos); the call goes on and
	// fails later on the fixture, not here
	h, _, _ := stack(t, saveResults("DRAFT")...)
	_, body := call(t, h, http.MethodPost, "/v1/task/SUBMITED", map[string]any{"site": sendableSite()}, 1001)
	assert.NotContains(t, string(body), "Belum bisa dikirim", string(body))

	// a draft may still be saved without pin or sales area
	h, _, _ = stack(t, saveResults("DRAFT")...)
	_, body = call(t, h, http.MethodPost, "/v1/task/DRAFT", map[string]any{"site": map[string]any{"site_id": "ST1", "site_name_1": "Toko Kemang"}, "generate_ai": false}, 1001)
	assert.NotContains(t, string(body), "Belum bisa dikirim")
}

// AI on save only when asked (Wes, 26 Oct). The Gemini mock is strict: a
// call nobody set up fails the test, so "no AI" is checked by not setting
// it up.
func TestSave_GenerateAIOnlyWhenAsked(t *testing.T) {
	draft := map[string]any{"site": map[string]any{"site_id": "ST1", "site_name_1": "Toko Kemang"}}

	// generate_ai false: no AI, even with the old default (auto on)
	withFlow(t)
	h, _, _ := stack(t, saveResults("DRAFT")...)
	code, body := call(t, h, http.MethodPost, "/v1/task/DRAFT", map[string]any{"site": draft["site"], "generate_ai": false}, 1001)
	assert.Equal(t, http.StatusOK, code, string(body))

	// [ai_insight] auto_generate_on_save=false, no flag: no AI
	testEnv.Set("ai_insight.auto_generate_on_save", false)
	t.Cleanup(func() { testEnv.Set("ai_insight.auto_generate_on_save", true) })
	withFlow(t)
	h, _, _ = stack(t, saveResults("DRAFT")...)
	code, body = call(t, h, http.MethodPost, "/v1/task/DRAFT", draft, 1001)
	assert.Equal(t, http.StatusOK, code, string(body))

	// switch off but generate_ai true: AI runs (an updated draft without an
	// insight; Gemini answers an error here, logged only, the save is OK)
	m := withFlow(t)
	m.gemini.On("GenerateInsight", mock.Anything, mock.Anything).Return(dto.AIInsightResponseGet{}, "", client.ErrGeminiNotSetUp).Once()
	h, _, _ = stack(t, saveResults("DRAFT")...)
	code, body = call(t, h, http.MethodPost, "/v1/task/DRAFT", map[string]any{"site": draft["site"], "generate_ai": true}, 1001)
	assert.Equal(t, http.StatusOK, code, string(body))
	m.gemini.AssertNumberOfCalls(t, "GenerateInsight", 1)
}
