package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	dto "site-dev-backend/internal/app/dto"

	"gitlab.com/fami-app-v2/familib/familabs/env"
)

// IGeocodingClient is the Google Geocoding API (point -> address, typed
// address -> points), ported from site-dev-api (standard library only).
type IGeocodingClient interface {
	// Configured: a key is set (geocoding.apikey).
	Configured() bool
	Reverse(ctx context.Context, lat float64, lng float64) (res dto.ReverseGeocodeResponse, err error)
	Search(ctx context.Context, query string) (res []dto.AddressSearchResult, err error)
}

type GeocodingClient struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// NewGeocodingClient reads geocoding.apikey (devops' secret store).
func NewGeocodingClient(env env.Provider) IGeocodingClient {
	c := &GeocodingClient{
		baseURL: "https://maps.googleapis.com/maps/api/geocode/json",
		http:    &http.Client{Timeout: 15 * time.Second},
	}
	if env != nil {
		c.apiKey = env.GetString("geocoding.apikey")
	}
	return c
}

type googleGeocodeAddressComponent struct {
	LongName string   `json:"long_name"`
	Types    []string `json:"types"`
}

type googleGeocodeGeometry struct {
	Location struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	} `json:"location"`
}

type googleGeocodeResult struct {
	AddressComponents []googleGeocodeAddressComponent `json:"address_components"`
	FormattedAddress  string                          `json:"formatted_address"`
	Geometry          googleGeocodeGeometry           `json:"geometry"`
}

type googleGeocodeResponseBody struct {
	Results      []googleGeocodeResult `json:"results"`
	Status       string                `json:"status"`
	ErrorMessage string                `json:"error_message"`
}

func hasType(types []string, want ...string) bool {
	for _, t := range types {
		for _, w := range want {
			if t == w {
				return true
			}
		}
	}
	return false
}

func joinNonEmpty(sep string, parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, sep)
}

// mapAddressComponents: address1 = street + number + village/sublocality,
// address2 = district (kecamatan), city, address3 = province. Indonesia:
// level_1 province, level_2 city/regency, level_3 kecamatan, level_4
// kelurahan/desa.
func mapAddressComponents(components []googleGeocodeAddressComponent) dto.ReverseGeocodeResponse {
	var route, streetNumber, sublocality, village, subdistrict, city, province, postal string
	for _, c := range components {
		switch {
		case hasType(c.Types, "route"):
			route = c.LongName
		case hasType(c.Types, "street_number"):
			streetNumber = c.LongName
		case hasType(c.Types, "sublocality_level_1", "sublocality"):
			sublocality = c.LongName
		case hasType(c.Types, "administrative_area_level_4"):
			village = c.LongName
		case hasType(c.Types, "administrative_area_level_3"):
			subdistrict = c.LongName
		case hasType(c.Types, "administrative_area_level_2"):
			city = c.LongName
		case hasType(c.Types, "administrative_area_level_1"):
			province = c.LongName
		case hasType(c.Types, "postal_code"):
			postal = c.LongName
		}
	}

	street := joinNonEmpty(" No. ", route, streetNumber)
	return dto.ReverseGeocodeResponse{
		Address1:   joinNonEmpty(", ", street, joinNonEmpty(" ", village, sublocality)),
		Address2:   joinNonEmpty(", ", subdistrict, city),
		Address3:   province,
		PostalCode: postal,
	}
}

func (g *GeocodingClient) Configured() bool { return g.apiKey != "" }

// call sends one Geocoding request; ZERO_RESULTS is an empty answer, not an
// error.
func (g *GeocodingClient) call(ctx context.Context, params url.Values) (googleGeocodeResponseBody, error) {
	if g.apiKey == "" {
		return googleGeocodeResponseBody{}, fmt.Errorf("geocoding: apikey not configured (check [geocoding] section in config)")
	}
	params.Set("language", "id")
	params.Set("key", g.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return googleGeocodeResponseBody{}, fmt.Errorf("geocoding: build request: %w", err)
	}

	res, err := g.http.Do(req)
	if err != nil {
		// the URL carries ?key= and *url.Error quotes it: keep only the
		// cause so the key never reaches logs or the answer (our fix
		// e4d8c60, not in 20eeab4)
		var uErr *url.Error
		if errors.As(err, &uErr) {
			err = uErr.Err
		}
		return googleGeocodeResponseBody{}, fmt.Errorf("geocoding: request failed: %w", err)
	}
	defer res.Body.Close()

	respBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return googleGeocodeResponseBody{}, fmt.Errorf("geocoding: read response: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		snippet := string(respBytes)
		if len(snippet) > 300 {
			snippet = snippet[:300]
		}
		return googleGeocodeResponseBody{}, fmt.Errorf("geocoding API %d: %s", res.StatusCode, snippet)
	}

	var parsed googleGeocodeResponseBody
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return googleGeocodeResponseBody{}, fmt.Errorf("geocoding: unmarshal response: %w", err)
	}

	switch parsed.Status {
	case "OK", "ZERO_RESULTS":
		return parsed, nil
	default:
		return googleGeocodeResponseBody{}, fmt.Errorf("geocoding API status %s: %s", parsed.Status, parsed.ErrorMessage)
	}
}

func (g *GeocodingClient) Reverse(ctx context.Context, lat float64, lng float64) (res dto.ReverseGeocodeResponse, err error) {
	q := url.Values{}
	q.Set("latlng", fmt.Sprintf("%f,%f", lat, lng))

	parsed, err := g.call(ctx, q)
	if err != nil {
		return dto.ReverseGeocodeResponse{}, err
	}
	if len(parsed.Results) == 0 {
		// ZERO_RESULTS (e.g. open water): blank, not an error
		return dto.ReverseGeocodeResponse{}, nil
	}
	return mapAddressComponents(parsed.Results[0].AddressComponents), nil
}

func (g *GeocodingClient) Search(ctx context.Context, query string) (res []dto.AddressSearchResult, err error) {
	q := url.Values{}
	q.Set("address", query)
	q.Set("region", "id")
	q.Set("components", "country:ID")

	parsed, err := g.call(ctx, q)
	if err != nil {
		return nil, err
	}

	res = make([]dto.AddressSearchResult, 0, len(parsed.Results))
	for _, r := range parsed.Results {
		addr := mapAddressComponents(r.AddressComponents)
		res = append(res, dto.AddressSearchResult{
			Lat:         r.Geometry.Location.Lat,
			Lng:         r.Geometry.Location.Lng,
			DisplayName: r.FormattedAddress,
			Address1:    addr.Address1,
			Address2:    addr.Address2,
			Address3:    addr.Address3,
			PostalCode:  addr.PostalCode,
		})
	}
	return res, nil
}
