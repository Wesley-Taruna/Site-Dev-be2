package service

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/model"

	"github.com/labstack/echo/v4"
)

// No AI on empty inputs (Wes, 27 Oct). The survey counts as filled when at
// least ONE of these was entered by the staff: foot traffic, peak hours,
// vehicle traffic (level, count or note), parking (availability or note),
// public transport, landmark, demographics, competitor note, or an extra
// note (title or text). Name + map point + sales area alone are not enough;
// the competitor rows do not count either (the app fills them from the
// map, not the staff).
var ErrAIInputsEmpty = errors.New("ai_insight: survey inputs are empty")

// MsgAIInputsEmpty is the 400 text (Indonesian, lists what to fill).
const MsgAIInputsEmpty = "Analisis AI belum bisa dibuat: data survei masih kosong. Isi minimal satu: keramaian orang lewat, jam ramai, lalu lintas kendaraan, parkir, transportasi umum, landmark sekitar, penduduk sekitar, catatan kompetitor, atau catatan tambahan."

// errAIInputsEmpty is the 400 answer (familabs envelope, errors.error =
// MsgAIInputsEmpty); errors.Is(err, ErrAIInputsEmpty) holds.
func errAIInputsEmpty() error {
	return echo.NewHTTPError(http.StatusBadRequest, MsgAIInputsEmpty).SetInternal(ErrAIInputsEmpty)
}

func filled(s string) bool { return strings.TrimSpace(s) != "" }

func filledPtr(s *string) bool { return s != nil && filled(*s) }

// aiInputsFilled: the staff entered at least one survey input (see
// ErrAIInputsEmpty).
func aiInputsFilled(c *model.SiteContext, notes []*model.SiteCustomNote) bool {
	for _, n := range notes {
		if n != nil && (filled(n.Title) || filled(n.Content)) {
			return true
		}
	}
	if c == nil {
		return false
	}
	return filled(c.FootTrafficObservation) || filled(c.PeakHours) || filled(c.PublicTransport) ||
		filled(c.AnchorLandmark) || filled(c.ObservedDemographics) ||
		filledPtr(c.VehicleTrafficLevel) || c.VehicleTrafficCount != nil || filledPtr(c.VehicleTrafficNote) ||
		filledPtr(c.ParkingAvailability) || filledPtr(c.ParkingNote) || filledPtr(c.CompetitorNote)
}

// previewInputsFilled is aiInputsFilled on a preview body (survey not
// saved).
func previewInputsFilled(c *dto.SiteContextRequestUpsert) bool {
	if c == nil {
		return false
	}
	notes := make([]*model.SiteCustomNote, 0, len(c.CustomNotes))
	for _, n := range c.CustomNotes {
		notes = append(notes, &model.SiteCustomNote{Title: n.Title, Content: n.Content})
	}
	return aiInputsFilled(&model.SiteContext{
		FootTrafficObservation: c.FootTrafficObservation,
		PeakHours:              c.PeakHours,
		ParkingAvailability:    c.ParkingAvailability,
		PublicTransport:        c.PublicTransport,
		AnchorLandmark:         c.AnchorLandmark,
		ObservedDemographics:   c.ObservedDemographics,
		VehicleTrafficLevel:    nilIfEmpty(c.VehicleTrafficLevel),
		VehicleTrafficCount:    c.VehicleTrafficCount,
		VehicleTrafficNote:     nilIfEmpty(c.VehicleTrafficNote),
		ParkingNote:            nilIfEmpty(c.ParkingNote),
		CompetitorNote:         nilIfEmpty(c.CompetitorNote),
	}, notes)
}

// AIInsightSkipped: the AI analysis was refused because there is nothing
// (new) to analyse: empty survey, or the same inputs as the newest
// insight; a queued job ends without retry.
func AIInsightSkipped(err error) bool {
	return errors.Is(err, ErrAIInputsEmpty) || errors.Is(err, ErrAIInputsUnchanged)
}

// Re-run (Wes, 27 Oct): after "Tolak, bisa direvisi" or on a reopened
// draft the staff may make the AI analysis again; a new result only when
// the inputs changed. Same inputs as the site's newest insight = 409, no
// Gemini call, nothing counted. No "force" on purpose: the same inputs give
// (almost) the same answer, so a forced call would only spend the limits.
var ErrAIInputsUnchanged = errors.New("ai_insight: inputs unchanged since the last analysis")

// MsgAIInputsUnchanged is the 409 text.
const MsgAIInputsUnchanged = "Data belum berubah sejak analisis terakhir. Ubah data survei atau lokasi dulu untuk membuat analisis baru."

func errAIInputsUnchanged() error {
	return echo.NewHTTPError(http.StatusConflict, MsgAIInputsUnchanged).SetInternal(ErrAIInputsUnchanged)
}

func strOf(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

func intOf(p *int) string {
	if p == nil {
		return ""
	}
	return fmt.Sprintf("%d", *p)
}

// aiInputHash is the fingerprint of what the AI analysis is made from: the
// point (5 decimals, ~1 m), address parts, sales area, 24 h / alcohol, the
// whole survey, competitor rows and extra notes (trimmed; rows sorted). The
// site name is left out (it changes only the headline), so are fields the
// app's preview does not send. Same value for a preview and its saved site.
func aiInputHash(site *model.Site, c *model.SiteContext, comps []*model.SiteCompetitor, notes []*model.SiteCustomNote) string {
	t := strings.TrimSpace
	lines := []string{"v1"}
	if site != nil {
		lines = append(lines,
			fmt.Sprintf("pt=%.5f,%.5f", site.SiteLat, site.SiteLon),
			"addr="+t(site.SiteAddress1)+"|"+t(site.SiteDistrict)+"|"+t(site.SiteCity)+"|"+t(site.SiteRegion)+"|"+t(site.SitePostalCode),
			fmt.Sprintf("area=%.2f;24h=%t;alc=%t", site.SiteSalesArea, site.SiteOpen24Hours, site.SiteSellingAlcohol))
	}
	if c != nil {
		lines = append(lines,
			"foot="+t(c.FootTrafficObservation), "peak="+t(c.PeakHours), "transport="+t(c.PublicTransport),
			"landmark="+t(c.AnchorLandmark), "demo="+t(c.ObservedDemographics),
			"traffic="+strOf(c.VehicleTrafficLevel)+"|"+intOf(c.VehicleTrafficCount)+"|"+intOf(c.VehicleTrafficMinutes)+"|"+strOf(c.VehicleTrafficNote),
			"parking="+strOf(c.ParkingAvailability)+"|"+strOf(c.ParkingNote), "compnote="+strOf(c.CompetitorNote))
	}
	var rows []string
	for _, x := range comps {
		if x != nil {
			rows = append(rows, "comp="+t(x.CompetitorName)+"|"+t(x.CompetitorType)+"|"+t(x.Distance))
		}
	}
	for _, n := range notes {
		if n != nil && (filled(n.Title) || filled(n.Content)) {
			rows = append(rows, "note="+t(n.Title)+"|"+t(n.Content))
		}
	}
	sort.Strings(rows)
	sum := sha256.Sum256([]byte(strings.Join(append(lines, rows...), "\n")))
	return hex.EncodeToString(sum[:])
}
