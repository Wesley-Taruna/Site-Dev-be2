package router

// Side-by-side check, module 9: GET /v1/geocoding/reverse and /search
// against site-dev-api 20eeab4 (Google is a mock here; the real client is
// ported and tested in internal/app/client).

import (
	"net/http"
	"testing"

	"site-dev-backend/internal/app/client"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// copies of site-dev-api 20eeab4 internal/app/dto/geocoding.go (unchanged)
type oldReverseGeocodeResp struct {
	Address1   string `json:"address1"`
	Address2   string `json:"address2"`
	Address3   string `json:"address3"`
	PostalCode string `json:"postal_code"`
}

type oldAddressSearchResult struct {
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	DisplayName string  `json:"display_name"`
	Address1    string  `json:"address1"`
	Address2    string  `json:"address2"`
	Address3    string  `json:"address3"`
	PostalCode  string  `json:"postal_code"`
}

func withGeocoding(t *testing.T) *mocks.IGeocodingClient {
	testEnv.Set("geocoding.enabled", true) // off by default since 22 Oct
	t.Cleanup(func() { testEnv.Set("geocoding.enabled", false) })
	geo := mocks.NewIGeocodingClient(t)
	geo.On("Configured").Return(true).Maybe()
	withClients(t, func(c *client.Clients) { c.Geocoding = geo })
	return geo
}

func TestSideBySide_GeocodingReverse(t *testing.T) {
	geo := withGeocoding(t)
	redis := withRedis(t)
	geo.On("Reverse", mock.Anything, -6.26, 106.81).Return(dto.ReverseGeocodeResponse{Address1: "Jl. Kemang", Address2: "Mampang, Jaksel", Address3: "DKI", PostalCode: "12730"}, nil)
	h, _, _ := stack(t)

	code, body := call(t, h, http.MethodGet, "/v1/geocoding/reverse?lat=-6.26&lng=106.81", nil, 1001)

	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, oldDataEnvelope(t, oldReverseGeocodeResp{"Jl. Kemang", "Mampang, Jaksel", "DKI", "12730"}), body)
	v, _ := redis.Get(quotaKey("geocoding", 1001))
	assert.Equal(t, "99", v)

	code, body = call(t, h, http.MethodGet, "/v1/geocoding/reverse?lat=x&lng=1", nil, 1001)
	assert.Equal(t, http.StatusUnprocessableEntity, code)
	assertSameJSON(t, []byte(`{"code":422,"status":"failure","message":"Unprocessable Entity","errors":{"lat":"The lat is invalid value."}}`), body)

	redis.Set(quotaKey("geocoding", 1001), "0")
	code, body = call(t, h, http.MethodGet, "/v1/geocoding/reverse?lat=1&lng=1", nil, 1001)
	assert.Equal(t, http.StatusTooManyRequests, code)
	assertSameJSON(t, []byte(`{"code":429,"status":"failure","message":"Too Many Requests","errors":{"error":"Kuota reverse-geocode harian (100) sudah habis, coba lagi besok."}}`), body)
}

func TestSideBySide_GeocodingSearch(t *testing.T) {
	geo := withGeocoding(t)
	withRedis(t)
	geo.On("Search", mock.Anything, "kemang raya").Return([]dto.AddressSearchResult{{Lat: -6.26, Lng: 106.81, DisplayName: "Kemang", Address1: "Jl. Kemang", PostalCode: "12730"}}, nil)
	h, _, _ := stack(t)

	code, body := call(t, h, http.MethodGet, "/v1/geocoding/search?q=+kemang+raya+", nil, 1001)
	require.Equal(t, http.StatusOK, code, string(body))
	assertSameJSON(t, oldDataEnvelope(t, []oldAddressSearchResult{{Lat: -6.26, Lng: 106.81, DisplayName: "Kemang", Address1: "Jl. Kemang", PostalCode: "12730"}}), body)

	code, body = call(t, h, http.MethodGet, "/v1/geocoding/search?q=", nil, 1001)
	require.Equal(t, http.StatusOK, code)
	assertSameJSON(t, []byte(`{"code":200,"status":"success","data":[]}`), body)

	code, _ = call(t, h, http.MethodGet, "/v1/geocoding/search?q=ke", nil, 1001)
	assert.Equal(t, http.StatusUnprocessableEntity, code)
}

func TestGeocoding_OffByDefault(t *testing.T) {
	h, _, _ := stack(t)

	code, body := call(t, h, http.MethodGet, "/v1/geocoding/reverse?lat=-6.26&lng=106.81", nil, 1001)

	assert.Equal(t, http.StatusServiceUnavailable, code)
	assertSameJSON(t, []byte(`{"code":503,"status":"failure","message":"Service Unavailable","errors":{"error":"Alamat otomatis (Google Geocoding) belum diaktifkan di server ini ([geocoding] enabled = false)."}}`), body)
}
