package service

import (
	"errors"
	"net/http"
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
// to analyse (empty survey); a queued job ends without retry.
func AIInsightSkipped(err error) bool {
	return errors.Is(err, ErrAIInputsEmpty)
}
