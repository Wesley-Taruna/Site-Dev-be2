package client

// Classification of nearby places (local §6, site-dev-api 2191e07
// nearby_area.go / places_overpass.go, from the prototype's areaMeta.ts and
// overpass.ts): which chain a name belongs to, and which OSM objects count.
// Pure functions, no network.

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"

	"site-dev-backend/internal/app/constants"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/pkg/geo"
)

type competitorBrand struct {
	name string
	re   *regexp.Regexp
}

// competitorBrands: order = match order (Alfamidi before Alfamart).
var competitorBrands = []competitorBrand{
	{"Indomaret", regexp.MustCompile(`(?i)indomaret`)},
	{"Alfamidi", regexp.MustCompile(`(?i)alfa\s*midi`)},
	{"Alfamart", regexp.MustCompile(`(?i)alfa\s*mart|alfa\s*express`)},
	{"Lawson", regexp.MustCompile(`(?i)lawson`)},
	{"Circle K", regexp.MustCompile(`(?i)circle\s*-?\s*k\b`)},
	{"7-Eleven", regexp.MustCompile(`(?i)7\s*-?\s*eleven|seven\s*eleven`)},
	{"K3 Mart", regexp.MustCompile(`(?i)k3\s*mart`)},
	{"Yomart", regexp.MustCompile(`(?i)yo\s*mart`)},
	{"Lulu Mart", regexp.MustCompile(`(?i)lulu\s*mart`)},
}

// BrandOther marks an unbranded minimarket; those are never counted.
const BrandOther = "Lainnya"

// BrandOf is the chain name found in text, or "".
func BrandOf(text string) string {
	for _, b := range competitorBrands {
		if b.re.MatchString(text) {
			return b.name
		}
	}
	return ""
}

var (
	reFamilyMart = regexp.MustCompile(`(?i)family\s*mart`)
	// reNotAShop: warehouses / offices that carry a chain name
	reNotAShop  = regexp.MustCompile(`(?i)\b(dc|gudang|warehouse|kantor|head office|distribution|kantor cabang)\b`)
	reVape      = regexp.MustCompile(`(?i)\bvape|vapor|vapour`)
	reNotOffice = regexp.MustCompile(`(?i)(pos satpam|pos ronda|pos kamling|poskamling|pos (rt|rw)|sekretariat (rt|rw)|rt\.? ?\d+ ?/? ?rw)`)
	reNormDrop  = regexp.MustCompile(`\b(the|apartemen|apartement|apartment|apartments|apt)\b`)
	reNonAlnum  = regexp.MustCompile(`[^a-z0-9]`)
)

// IsFamilyMart: our own stores come from MStore, never from the map data.
func IsFamilyMart(name string) bool { return reFamilyMart.MatchString(name) }

// IsNotAShop: a chain name on a warehouse or office.
func IsNotAShop(name string) bool { return reNotAShop.MatchString(name) }

// OverpassAreaQuery is the Overpass query for every counted category in
// the box around the circle (the circle is applied afterwards).
func OverpassAreaQuery(lat float64, lng float64, radiusM int) string {
	minLat, maxLat, minLng, maxLng := geo.BoundingBox(lat, lng, float64(radiusM))
	bb := fmt.Sprintf("%.5f,%.5f,%.5f,%.5f", minLat, minLng, maxLat, maxLng)
	return `[out:json][timeout:60][bbox:` + bb + `];
(
  nwr[shop~"^(convenience|supermarket|minimarket)$"];
  nwr[shop][name~"indomaret|alfa ?mart|alfa ?midi|alfa ?express|lawson|circle ?k|7.?eleven|k3 ?mart|yo ?mart|lulu ?mart",i];
  nwr[office][name];
  nwr[building=office][name];
  nwr[building=commercial][name~"menara|tower|wisma|graha|plaza|gedung|office",i];
  nwr[amenity~"^(school|kindergarten|university|college)$"][name];
  nwr[building~"^(school|kindergarten)$"][name];
  nwr[building=apartments][name];
  nwr[residential=apartments][name];
);
out center tags 5000;`
}

// osmCategoryOf classifies one OSM object.
func osmCategoryOf(t map[string]string) (category string, brand string, ok bool) {
	name := t["name"]
	if reFamilyMart.MatchString(t["brand"] + " " + name) {
		return "", "", false
	}
	if t["shop"] != "" {
		var parts []string
		for _, k := range []string{"brand", "name", "operator"} {
			if t[k] != "" {
				parts = append(parts, t[k])
			}
		}
		if b := BrandOf(strings.Join(parts, " ")); b != "" {
			return constants.NearbyCategoryCompetitor, b, true
		}
		if reVape.MatchString(name) {
			return "", "", false // vape shops are sometimes tagged convenience
		}
		if t["shop"] == "convenience" || t["shop"] == "minimarket" {
			return constants.NearbyCategoryCompetitor, BrandOther, true
		}
		return "", "", false // plain supermarkets are not minimarket competitors
	}
	// campus = the campus area only, not each building in it
	if t["amenity"] == "university" || t["amenity"] == "college" {
		return constants.NearbyCategoryUniversity, "", true
	}
	if t["amenity"] == "school" || t["amenity"] == "kindergarten" || t["building"] == "school" || t["building"] == "kindergarten" {
		return constants.NearbyCategorySchool, "", true
	}
	if t["building"] == "apartments" || t["residential"] == "apartments" {
		return constants.NearbyCategoryApartment, "", true
	}
	if reNotOffice.MatchString(name) {
		return "", "", false
	}
	if t["office"] != "" || t["building"] == "office" || t["building"] == "commercial" {
		return constants.NearbyCategoryOffice, "", true
	}
	return "", "", false
}

// normPlaceName merges duplicates like "The Capital Residence" and
// "Apartment Capital Residences".
func normPlaceName(s string) string {
	s = strings.ToLower(s)
	s = reNormDrop.ReplaceAllString(s, "")
	s = reNonAlnum.ReplaceAllString(s, "")
	return strings.TrimSuffix(s, "s")
}

type overpassElement struct {
	Type   string                      `json:"type"`
	ID     int64                       `json:"id"`
	Lat    *float64                    `json:"lat"`
	Lon    *float64                    `json:"lon"`
	Center *struct{ Lat, Lon float64 } `json:"center"`
	Tags   map[string]string           `json:"tags"`
}

// ParseOverpassArea turns an Overpass JSON answer into the places within
// radiusM of the point, nearest first; the same place mapped twice (same
// category and name within 150 m) counts once.
func ParseOverpassArea(raw []byte, lat float64, lng float64, radiusM int) ([]dto.NearbyPlaceItem, error) {
	var parsed struct {
		Elements []overpassElement `json:"elements"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("overpass: unreadable answer: %w", err)
	}
	if parsed.Elements == nil {
		return nil, errors.New("overpass: answer has no elements")
	}

	out := make([]dto.NearbyPlaceItem, 0)
	seen := map[string]bool{}
	for _, el := range parsed.Elements {
		var pLat, pLng float64
		switch {
		case el.Type == "node" && el.Lat != nil && el.Lon != nil:
			pLat, pLng = *el.Lat, *el.Lon
		case el.Center != nil:
			pLat, pLng = el.Center.Lat, el.Center.Lon
		default:
			continue
		}
		d := int(math.Round(geo.HaversineM(lat, lng, pLat, pLng)))
		if d > radiusM {
			continue
		}
		cat, brand, ok := osmCategoryOf(el.Tags)
		if !ok {
			continue
		}
		key := fmt.Sprintf("%s%d", el.Type, el.ID)
		if seen[key] {
			continue
		}
		seen[key] = true

		name := strings.TrimSpace(el.Tags["name"])
		if n := normPlaceName(name); n != "" {
			dup := false
			for _, o := range out {
				if o.Category == cat && normPlaceName(o.Name) == n && geo.HaversineM(o.Lat, o.Lng, pLat, pLng) <= 150 {
					dup = true
					break
				}
			}
			if dup {
				continue
			}
		}
		out = append(out, dto.NearbyPlaceItem{
			ID:        fmt.Sprintf("o:%s/%d", el.Type, el.ID),
			Name:      name,
			Category:  cat,
			Brand:     brand,
			Lat:       pLat,
			Lng:       pLng,
			DistanceM: d,
			Source:    constants.NearbySourceOSM,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].DistanceM < out[j].DistanceM })
	return out, nil
}
