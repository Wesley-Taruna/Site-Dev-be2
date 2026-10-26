package router

// November scope, API-CHANGES §13.5 / §6 note 19 Oct: vehicle traffic,
// parking note and staff competitor note on the survey (PUT
// /v1/task/context/:site_id, POST /v1/task/:status), and the
// "traffic required on SUBMITED" switch. New behaviour; limits as the
// prototype's form (site-dev-mobile ecdb9be).

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func surveyBody(extra map[string]any) map[string]any {
	b := map[string]any{"peak_hours": "17-20"}
	for k, v := range extra {
		b[k] = v
	}
	return b
}

func TestSurvey_NewFieldsSavedAndAnswered(t *testing.T) {
	h, rec, _ := stack(t, contextUpdateResults("DRAFT", 1)...)

	code, body := call(t, h, http.MethodPut, "/v1/task/context/ST1", surveyBody(map[string]any{
		"vehicle_traffic_level": "BUSY", "vehicle_traffic_count": 120, "vehicle_traffic_minutes": 10, "vehicle_traffic_note": "bus berhenti",
		"parking_note": "motor 6", "competitor_note": "Alfamart tutup",
	}), 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	assert.Contains(t, stableSQL(rec.sqls), `UPDATE "SiteContexts" SET "AnchorLandmark"='',"CompetitorNote"='Alfamart tutup',"FootTrafficObservation"='',"ObservedDemographics"='',"ParkingAvailability"=NULL,"ParkingNote"='motor 6',"PeakHours"='17-20',"PublicTransport"='',"UpdatedAt"='<time>',"UpdatedBy"=1001,"VehicleTrafficCount"=120,"VehicleTrafficLevel"='BUSY',"VehicleTrafficMinutes"=10,"VehicleTrafficNote"='bus berhenti' WHERE SiteID = 'ST1'`)
	var res map[string]any
	require.NoError(t, jsonUnmarshalData(body, &res))
	assert.Equal(t, "BUSY", res["vehicle_traffic_level"])
	assert.Equal(t, float64(120), res["vehicle_traffic_count"])
	assert.Equal(t, float64(10), res["vehicle_traffic_minutes"])
	assert.Equal(t, "bus berhenti", res["vehicle_traffic_note"])
	assert.Equal(t, "motor 6", res["parking_note"])
	assert.Equal(t, "Alfamart tutup", res["competitor_note"])
}

func TestSurvey_Validation(t *testing.T) {
	h, _, _ := stack(t, contextUpdateResults("DRAFT", 1)...)
	long := make([]rune, 301)
	for i := range long {
		long[i] = 'é' // characters, not bytes
	}

	for name, tc := range map[string]struct {
		extra map[string]any
		want  string
	}{
		"bad level":         {map[string]any{"vehicle_traffic_level": "LOUD"}, `"vehicle_traffic_level":"The selected vehicle traffic level is invalid."`},
		"count too big":     {map[string]any{"vehicle_traffic_count": 100000, "vehicle_traffic_minutes": 10}, `"vehicle_traffic_count":"The vehicle traffic count must be between 0 and 99999."`},
		"count no minutes":  {map[string]any{"vehicle_traffic_count": 5}, `"vehicle_traffic_minutes":"The vehicle traffic minutes field is required when vehicle traffic count is present."`},
		"minutes too long":  {map[string]any{"vehicle_traffic_count": 5, "vehicle_traffic_minutes": 121}, `"vehicle_traffic_minutes":"The vehicle traffic minutes must be between 1 and 120."`},
		"parking note long": {map[string]any{"parking_note": string(long)}, `"parking_note":"The parking_note may not be greater than 300 characters."`},
	} {
		code, body := call(t, h, http.MethodPut, "/v1/task/context/ST1", surveyBody(tc.extra), 1001)
		assert.Equal(t, http.StatusUnprocessableEntity, code, name)
		assert.Contains(t, string(body), tc.want, name)
	}

	code, body := call(t, h, http.MethodPut, "/v1/task/context/ST1", surveyBody(map[string]any{"parking_note": string(long[:300])}), 1001)
	assert.Equal(t, http.StatusOK, code, string(body))
}

// Traffic is optional on send too (Wes, 26 Oct): the old
// [survey] traffic_required_on_submit switch is gone, so even a config that
// still says true does not stop a send without vehicle_traffic_level.
func TestSurvey_TrafficNotRequiredOnSubmit(t *testing.T) {
	withFlow(t)
	testEnv.Set("survey.traffic_required_on_submit", true)
	t.Cleanup(func() { testEnv.Set("survey.traffic_required_on_submit", false) })
	h, _, _ := stack(t, saveResults("DRAFT")...)
	send := map[string]any{"site": sendableSite(), "context": surveyBody(nil)}

	// the call goes on past the checks (fails later on the fixture, not here)
	_, body := call(t, h, http.MethodPost, "/v1/task/SUBMITED", send, 1001)
	assert.NotContains(t, string(body), "Lalu lintas kendaraan wajib diisi")
	assert.NotContains(t, string(body), "Belum bisa dikirim")
}
