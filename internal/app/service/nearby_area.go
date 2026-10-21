package service

import (
	"math"
	"sort"

	"site-dev-backend/internal/app/client"
	"site-dev-backend/internal/app/constants"
	dto "site-dev-backend/internal/app/dto"
)

// Pure helpers of the nearby places (local §6; site-dev-api 2191e07
// nearby_area.go, from the prototype's areaMeta.ts / server/area.ts): one
// fetched area filtered to a point, the prompt data, the nearest store.

// storedArea is one fetched area as cached: every place within
// FetchRadiusM of the rounded centre, before filtering to a point.
type storedArea struct {
	FetchedAtUnix  int64                             `json:"at"`
	Lat            float64                           `json:"lat"`
	Lng            float64                           `json:"lng"`
	FetchRadiusM   int                               `json:"radius"`
	Source         string                            `json:"source"`
	Places         []dto.NearbyPlaceItem             `json:"places"`
	Categories     map[string]dto.NearbyCategoryInfo `json:"categories"`
	GoogleCalls    int                               `json:"google_calls"`
	FallbackReason string                            `json:"fallback_reason,omitempty"`
}

// nearbyAttribution is the data credit line for a source.
func nearbyAttribution(source string) string {
	switch source {
	case constants.NearbySourceOSM:
		return "© OpenStreetMap contributors"
	case constants.NearbySourceMixed:
		return "Google Places + © OpenStreetMap contributors"
	}
	return "Google Places"
}

// countable: named places only, competitors only of a known chain.
func countable(p dto.NearbyPlaceItem) bool {
	if p.Name == "" {
		return false
	}
	if p.Category == constants.NearbyCategoryCompetitor && (p.Brand == "" || p.Brand == client.BrandOther) {
		return false
	}
	return true
}

// areaForPoint filters a stored area to the circle around the exact point:
// distances from the point, nearest first, each category count = the
// places listed for it.
func areaForPoint(st *storedArea, lat float64, lng float64, radiusM int) ([]dto.NearbyPlaceItem, map[string]dto.NearbyCategoryInfo) {
	known := map[string]bool{}
	for _, c := range constants.NearbyCategories {
		known[c] = true
	}

	places := make([]dto.NearbyPlaceItem, 0, len(st.Places))
	for _, p := range st.Places {
		if !known[p.Category] || !countable(p) {
			continue
		}
		p.DistanceM = int(math.Round(haversineM(lat, lng, p.Lat, p.Lng)))
		if p.DistanceM > radiusM {
			continue
		}
		places = append(places, p)
	}
	sort.SliceStable(places, func(i, j int) bool { return places[i].DistanceM < places[j].DistanceM })

	shift := int(math.Round(haversineM(lat, lng, st.Lat, st.Lng)))
	cats := make(map[string]dto.NearbyCategoryInfo, len(constants.NearbyCategories))
	for _, c := range constants.NearbyCategories {
		info, ok := st.Categories[c]
		if !ok {
			info = dto.NearbyCategoryInfo{Source: st.Source}
			if st.Source == constants.NearbySourceMixed {
				info.Source = constants.NearbySourceGoogle
			}
		}
		info.Count = 0
		for _, p := range places {
			if p.Category == c {
				info.Count++
			}
		}
		if info.Capped && info.CompleteWithinM != nil {
			v := *info.CompleteWithinM - shift
			if v < 0 {
				v = 0
			}
			info.CompleteWithinM = &v
		}
		cats[c] = info
	}
	return places, cats
}

// measuredFromArea is the prompt data of an area already filtered to the
// site: every competitor and the nearest few of each other category.
func measuredFromArea(area dto.NearbyAreaResponseGet) *dto.NearbyMeasured {
	m := &dto.NearbyMeasured{
		RadiusM:           area.RadiusM,
		Counts:            map[string]int{},
		ExamplesPerGroup:  client.NearbyExamplesPerGroup,
		NearestFamilyMart: area.NearestFamilyMart,
		StoreSource:       area.StoreSource,
		FetchedAt:         area.FetchedAt,
	}
	sources := map[string]bool{}
	for _, c := range constants.NearbyCategories {
		info := area.Categories[c]
		m.Counts[c] = info.Count
		if info.Source != "" {
			sources[info.Source] = true
		}
		if info.Capped && (info.CompleteWithinM == nil || *info.CompleteWithinM < area.RadiusM) {
			m.Capped = append(m.Capped, c)
		}
	}
	switch {
	case len(sources) > 1:
		m.Source = constants.NearbySourceMixed
	case len(sources) == 1:
		for s := range sources {
			m.Source = s
		}
	default:
		m.Source = area.Source
	}

	perCategory := map[string]int{}
	for _, p := range area.Places {
		if p.Category != constants.NearbyCategoryCompetitor {
			if perCategory[p.Category] >= client.NearbyExamplesPerGroup {
				continue
			}
			perCategory[p.Category]++
		}
		m.Items = append(m.Items, dto.NearbyMeasuredItem{Name: p.Name, Category: p.Category, Brand: p.Brand, DistanceM: p.DistanceM})
	}
	return m
}

// storePoint is one active FamilyMart with coordinates.
type storePoint struct {
	Code string
	Name string
	Lat  float64
	Lng  float64
}

// nearestStore is the nearest store and how many are within 1 km.
func nearestStore(lat float64, lng float64, stores []storePoint) *dto.NearestStoreResponse {
	var best *dto.NearestStoreResponse
	bestD := math.MaxFloat64
	within := 0
	for _, s := range stores {
		if s.Lat == 0 && s.Lng == 0 {
			continue
		}
		d := haversineM(lat, lng, s.Lat, s.Lng)
		if d <= 1000 {
			within++
		}
		if d < bestD {
			bestD = d
			best = &dto.NearestStoreResponse{Name: s.Name, Code: s.Code, DistanceM: int(math.Round(d))}
		}
	}
	if best != nil {
		best.Within1Km = within
	}
	return best
}
