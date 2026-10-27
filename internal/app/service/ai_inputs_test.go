package service

import (
	"testing"

	"site-dev-backend/internal/app/model"

	"github.com/stretchr/testify/assert"
)

func ptr[T any](v T) *T { return &v }

func hashFixture() (*model.Site, *model.SiteContext, []*model.SiteCompetitor, []*model.SiteCustomNote) {
	site := &model.Site{SiteName1: "Toko", SiteAddress1: "Jl. A", SiteCity: "Jakarta", SiteLat: -6.26, SiteLon: 106.81, SiteSalesArea: 120}
	ctx := &model.SiteContext{PeakHours: "17-20", ParkingAvailability: ptr("LIMITED"), VehicleTrafficLevel: ptr("BUSY"), VehicleTrafficCount: ptr(40)}
	comps := []*model.SiteCompetitor{{CompetitorName: "Indomaret", Distance: "50m"}, {CompetitorName: "Alfamart", Distance: "120m"}}
	notes := []*model.SiteCustomNote{{Title: "Sewa", Content: "mahal"}}
	return site, ctx, comps, notes
}

func TestAIInputHash(t *testing.T) {
	site, ctx, comps, notes := hashFixture()
	base := aiInputHash(site, ctx, comps, notes)
	assert.Len(t, base, 64)

	// not a change: name, saved-only fields, spaces, row order, empty notes
	s2, c2, k2, n2 := hashFixture()
	s2.SiteName1, s2.SiteName2, s2.SiteAddress2, s2.SitePic = "Nama lain", "x", "RT 1", "Pak"
	c2.PeakHours = " 17-20 "
	k2[0], k2[1] = k2[1], k2[0]
	n2 = append(n2, &model.SiteCustomNote{})
	assert.Equal(t, base, aiInputHash(s2, c2, k2, n2))

	// a change: every input moves the fingerprint
	for name, change := range map[string]func(*model.Site, *model.SiteContext, *[]*model.SiteCompetitor, *[]*model.SiteCustomNote){
		"pin moved": func(s *model.Site, _ *model.SiteContext, _ *[]*model.SiteCompetitor, _ *[]*model.SiteCustomNote) {
			s.SiteLat += 0.0001
		},
		"sales area": func(s *model.Site, _ *model.SiteContext, _ *[]*model.SiteCompetitor, _ *[]*model.SiteCustomNote) {
			s.SiteSalesArea = 90
		},
		"24 h": func(s *model.Site, _ *model.SiteContext, _ *[]*model.SiteCompetitor, _ *[]*model.SiteCustomNote) {
			s.SiteOpen24Hours = true
		},
		"peak hours": func(_ *model.Site, c *model.SiteContext, _ *[]*model.SiteCompetitor, _ *[]*model.SiteCustomNote) {
			c.PeakHours = "07-09"
		},
		"parking": func(_ *model.Site, c *model.SiteContext, _ *[]*model.SiteCompetitor, _ *[]*model.SiteCustomNote) {
			c.ParkingAvailability = ptr("NONE")
		},
		"parking note": func(_ *model.Site, c *model.SiteContext, _ *[]*model.SiteCompetitor, _ *[]*model.SiteCustomNote) {
			c.ParkingNote = ptr("motor")
		},
		"traffic count": func(_ *model.Site, c *model.SiteContext, _ *[]*model.SiteCompetitor, _ *[]*model.SiteCustomNote) {
			c.VehicleTrafficCount = ptr(41)
		},
		"competitor note": func(_ *model.Site, c *model.SiteContext, _ *[]*model.SiteCompetitor, _ *[]*model.SiteCustomNote) {
			c.CompetitorNote = ptr("tutup")
		},
		"competitor gone": func(_ *model.Site, _ *model.SiteContext, k *[]*model.SiteCompetitor, _ *[]*model.SiteCustomNote) {
			*k = (*k)[:1]
		},
		"note text": func(_ *model.Site, _ *model.SiteContext, _ *[]*model.SiteCompetitor, n *[]*model.SiteCustomNote) {
			(*n)[0].Content = "murah"
		},
		"demographics": func(_ *model.Site, c *model.SiteContext, _ *[]*model.SiteCompetitor, _ *[]*model.SiteCustomNote) {
			c.ObservedDemographics = "pelajar"
		},
		"no survey at all": func(_ *model.Site, c *model.SiteContext, _ *[]*model.SiteCompetitor, _ *[]*model.SiteCustomNote) {
			*c = model.SiteContext{}
		},
	} {
		s, c, k, n := hashFixture()
		change(s, c, &k, &n)
		assert.NotEqual(t, base, aiInputHash(s, c, k, n), name)
	}
}

func TestAIInputsFilled(t *testing.T) {
	assert.False(t, aiInputsFilled(nil, nil))
	assert.False(t, aiInputsFilled(&model.SiteContext{PeakHours: "  "}, []*model.SiteCustomNote{{Title: " "}}))
	assert.True(t, aiInputsFilled(nil, []*model.SiteCustomNote{{Content: "banjir"}}))
	assert.True(t, aiInputsFilled(&model.SiteContext{VehicleTrafficCount: ptr(0)}, nil))
	assert.True(t, aiInputsFilled(&model.SiteContext{CompetitorNote: ptr("x")}, nil))
}
