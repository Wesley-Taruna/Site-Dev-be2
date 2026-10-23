package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

const testGeoKey = "TEST-KEY-not-a-real-key"

func geoClient(fn roundTripFunc) *GeocodingClient {
	return &GeocodingClient{apiKey: testGeoKey, baseURL: "https://maps.example.test/geocode/json", http: &http.Client{Transport: fn}}
}

func answer(status int, body string) (*http.Response, error) {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}, nil
}

const kemang = `{"status":"OK","results":[{"formatted_address":"Jl. Kemang Raya No.1, Jakarta","geometry":{"location":{"lat":-6.26,"lng":106.81}},
"address_components":[{"long_name":"1","types":["street_number"]},{"long_name":"Jalan Kemang Raya","types":["route"]},
{"long_name":"Bangka","types":["administrative_area_level_4"]},{"long_name":"Mampang Prapatan","types":["administrative_area_level_3"]},
{"long_name":"Jakarta Selatan","types":["administrative_area_level_2"]},{"long_name":"DKI Jakarta","types":["administrative_area_level_1"]},
{"long_name":"12730","types":["postal_code"]}]}]}`

func TestGeocodingReverse(t *testing.T) {
	var got *http.Request
	c := geoClient(func(r *http.Request) (*http.Response, error) { got = r; return answer(200, kemang) })

	res, err := c.Reverse(context.Background(), -6.26, 106.81)

	require.NoError(t, err)
	assert.Equal(t, "Jalan Kemang Raya No. 1, Bangka", res.Address1)
	assert.Equal(t, "Mampang Prapatan, Jakarta Selatan", res.Address2)
	assert.Equal(t, "DKI Jakarta", res.Address3)
	assert.Equal(t, "12730", res.PostalCode)
	assert.Equal(t, "-6.260000,106.810000", got.URL.Query().Get("latlng"))
	assert.Equal(t, "id", got.URL.Query().Get("language"))
}

func TestGeocodingSearch(t *testing.T) {
	var got *http.Request
	c := geoClient(func(r *http.Request) (*http.Response, error) { got = r; return answer(200, kemang) })

	res, err := c.Search(context.Background(), "kemang raya")

	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, "Jl. Kemang Raya No.1, Jakarta", res[0].DisplayName)
	assert.Equal(t, -6.26, res[0].Lat)
	assert.Equal(t, "country:ID", got.URL.Query().Get("components"))
	assert.Equal(t, "id", got.URL.Query().Get("region"))

	c = geoClient(func(*http.Request) (*http.Response, error) {
		return answer(200, `{"status":"ZERO_RESULTS","results":[]}`)
	})
	res, err = c.Search(context.Background(), "nowhere")
	require.NoError(t, err)
	assert.NotNil(t, res)
	assert.Empty(t, res)
}

func TestGeocodingErrors(t *testing.T) {
	c := geoClient(func(*http.Request) (*http.Response, error) {
		return answer(200, `{"status":"REQUEST_DENIED","error_message":"bad key"}`)
	})
	_, err := c.Reverse(context.Background(), 1, 2)
	assert.EqualError(t, err, "geocoding API status REQUEST_DENIED: bad key")
	var denied *GeocodingDeniedError
	require.True(t, errors.As(err, &denied), "refused key = typed error (service: 503)")
	assert.Equal(t, "REQUEST_DENIED", denied.Status)

	c = geoClient(func(*http.Request) (*http.Response, error) { return answer(403, `{"error":"forbidden"}`) })
	_, err = c.Search(context.Background(), "x")
	require.True(t, errors.As(err, &denied))
	assert.Equal(t, "HTTP 403", denied.Status)

	c = geoClient(func(*http.Request) (*http.Response, error) {
		return answer(200, `{"status":"OVER_QUERY_LIMIT","error_message":"slow down"}`)
	})
	_, err = c.Reverse(context.Background(), 1, 2)
	assert.False(t, errors.As(err, &denied), "other statuses stay plain errors")

	c = geoClient(func(*http.Request) (*http.Response, error) { return nil, errors.New("dial tcp: i/o timeout") })
	_, err = c.Reverse(context.Background(), 1, 2)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), testGeoKey, "the key never reaches the error text")
	assert.Equal(t, "geocoding: request failed: dial tcp: i/o timeout", err.Error())

	_, err = (&GeocodingClient{http: http.DefaultClient}).Reverse(context.Background(), 1, 2)
	assert.EqualError(t, err, "geocoding: apikey not configured (check [geocoding] section in config)")
}

// 23 Oct: "No. No.5" when Google's street_number carries its own "No.".
func TestStreetWithNumber(t *testing.T) {
	for _, tc := range [][3]string{
		{"Jalan Kemang Raya", "1", "Jalan Kemang Raya No. 1"},
		{"Jalan Kemang Raya", "No.5", "Jalan Kemang Raya No. 5"},
		{"Jalan Kemang Raya", "No. 5A", "Jalan Kemang Raya No. 5A"},
		{"Jalan Kemang Raya", "nomor 12", "Jalan Kemang Raya No. 12"},
		{"Jl. H. R. Rasuna Said No.5", "No.5", "Jl. H. R. Rasuna Said No.5"},
		{"Jalan Kemang Raya", "", "Jalan Kemang Raya"},
		{"", "No.5", "5"},
		{"Jalan Nomaden", "7", "Jalan Nomaden No. 7"},
	} {
		assert.Equal(t, tc[2], streetWithNumber(tc[0], tc[1]), tc[0]+" | "+tc[1])
	}
}
