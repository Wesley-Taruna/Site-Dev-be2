package service

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"

	"site-dev-backend/internal/app/constants"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/model"

	"github.com/labstack/echo/v4"
	"gitlab.com/fami-app-v2/familib/familabs/env"
	"gitlab.com/fami-app-v2/familib/familabs/log"
)

// Other requests already in process near a point (local §8, SPV 15 Oct
// 2026): so two staff do not survey and send the same spot.

const (
	defaultNearbyRequestRadiusM = 500
	maxNearbyRequestRadiusM     = 2000
	defaultDuplicateRadiusM     = 200
)

// isActiveRequestStatus is the same rule as the SQL of
// GetActiveRequestPoints.
func isActiveRequestStatus(status string) bool {
	if strings.HasPrefix(status, "WAITING_") {
		return true
	}
	for _, s := range constants.NearbyRequestExtraStatuses {
		if status == s {
			return true
		}
	}
	return false
}

// nearbyActive is the active requests within radiusM of the point (exact
// distance after the SQL box), nearest first, without excludeSiteID.
func (s *TaskService) nearbyActive(ctx context.Context, lat float64, lng float64, radiusM int, excludeSiteID string) (res []dto.NearbyRequestResponseGet, err error) {
	minLat, maxLat, minLng, maxLng := boundingBox(lat, lng, float64(radiusM))
	var rows []model.RequestPoint
	rows, err = s.RepositoryTask.GetActiveRequestPoints(ctx, minLat, maxLat, minLng, maxLng, constants.NearbyRequestExtraStatuses)
	if err != nil {
		return
	}

	type withDistance struct {
		row model.RequestPoint
		d   float64
	}
	var kept []withDistance
	for _, r := range rows {
		if r.SiteID == excludeSiteID || !isActiveRequestStatus(r.TaskStatus) || (r.SiteLat == 0 && r.SiteLon == 0) {
			continue
		}
		if d := haversineM(lat, lng, r.SiteLat, r.SiteLon); d <= float64(radiusM) {
			kept = append(kept, withDistance{r, d})
		}
	}
	sort.SliceStable(kept, func(i, j int) bool { return kept[i].d < kept[j].d })

	names := map[int64]string{}
	res = make([]dto.NearbyRequestResponseGet, 0, len(kept))
	for _, k := range kept {
		name, ok := names[k.row.CreatedBy]
		if !ok {
			name = usernameOf(ctx, s.RepositoryUser, k.row.CreatedBy)
			names[k.row.CreatedBy] = name
		}
		res = append(res, dto.NearbyRequestResponseGet{
			SiteID:     k.row.SiteID,
			SiteName:   k.row.SiteName1,
			TaskStatus: k.row.TaskStatus,
			OwnerID:    k.row.CreatedBy,
			OwnerName:  name,
			Lat:        k.row.SiteLat,
			Lng:        k.row.SiteLon,
			DistanceM:  int(math.Round(k.d)),
		})
	}
	return
}

// GetNearbyRequests is GET /v1/task/nearby (all roles): radius 0 = 500 m,
// at most 2000 m.
func (s *TaskService) GetNearbyRequests(ctx context.Context, req dto.NearbyRequestGet) (res []dto.NearbyRequestResponseGet, err error) {
	ctx, span := s.opt.Trace.Start(ctx, "TaskService.GetNearbyRequests")
	defer span.End()

	radiusM := req.RadiusM
	if err = validatePointInIndonesia(req.Lat, req.Lng); err == nil {
		if radiusM <= 0 {
			radiusM = defaultNearbyRequestRadiusM
		}
		if radiusM > maxNearbyRequestRadiusM {
			err = echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("radius maksimal %d m", maxNearbyRequestRadiusM))
		}
	}
	if err == nil {
		res, err = s.nearbyActive(ctx, req.Lat, req.Lng, radiusM, req.ExcludeSiteID)
	}
	if err != nil {
		span.RecordError(err)
		s.opt.Logger.AddMessage(log.ErrorLevel, err)
		res = nil
		return
	}

	return
}

// DuplicateNearbyError: a submit with active requests within the duplicate
// radius and no confirm_duplicate. The handler answers 409 with Items as
// data.
type DuplicateNearbyError struct {
	RadiusM int
	Items   []dto.NearbyRequestResponseGet
}

func (e *DuplicateNearbyError) Error() string {
	return fmt.Sprintf("DUPLICATE_NEARBY: sudah ada %d pengajuan aktif dalam %d m dari titik ini. Cek dulu, lalu kirim ulang dengan confirm_duplicate = true bila tetap ingin mengirim.", len(e.Items), e.RadiusM)
}

// duplicateOutcome is what a submit does about requests close by.
type duplicateOutcome int

const (
	duplicateNone    duplicateOutcome = iota // nothing close by, or check off
	duplicateBlock                           // answer 409
	duplicateConfirm                         // send + DUPLICATE_CONFIRMED history row
)

// decideDuplicate: warn-and-allow, as built in local §8 (whether to block
// instead is open, plan / review §5 #5).
func decideDuplicate(mode string, found int, confirmed bool) duplicateOutcome {
	if mode != constants.DuplicateCheckConfirm || found == 0 {
		return duplicateNone
	}
	if confirmed {
		return duplicateConfirm
	}
	return duplicateBlock
}

// duplicateConfirmedNotes is the DUPLICATE_CONFIRMED history note (at most
// 500 characters).
func duplicateConfirmedNotes(radiusM int, items []dto.NearbyRequestResponseGet) string {
	parts := make([]string, 0, len(items))
	for _, it := range items {
		parts = append(parts, fmt.Sprintf("%s %s (%s, %d m)", it.SiteID, it.SiteName, it.TaskStatus, it.DistanceM))
	}
	msg := fmt.Sprintf("Staf tetap mengirim walau ada pengajuan aktif lain dalam %d m: %s", radiusM, strings.Join(parts, "; "))
	if r := []rune(msg); len(r) > 500 {
		msg = string(r[:497]) + "..."
	}
	return msg
}

// duplicateCheckConfig is [nearby_request] in the config.
type duplicateCheckConfig struct {
	mode    string
	radiusM int
}

func loadDuplicateCheckConfig(env env.Provider) duplicateCheckConfig {
	c := duplicateCheckConfig{mode: constants.DuplicateCheckOff, radiusM: defaultDuplicateRadiusM}
	if env == nil {
		return c
	}
	if v := strings.ToLower(strings.TrimSpace(env.GetString("nearby_request.duplicate_check"))); v == constants.DuplicateCheckConfirm {
		c.mode = v
	}
	if n := env.GetInt("nearby_request.duplicate_radius_m"); n > 0 {
		c.radiusM = n
	}
	return c
}
