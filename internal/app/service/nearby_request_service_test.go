package service

import (
	"context"
	"strings"
	"testing"

	"site-dev-backend/internal/app/constants"
	dto "site-dev-backend/internal/app/dto"
	"site-dev-backend/internal/app/model"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const scbdLat, scbdLng = -6.2245, 106.8095

func TestIsActiveRequestStatus(t *testing.T) {
	for s, want := range map[string]bool{
		"WAITING_DEPT_HEAD": true, "WAITING_DIV_HEAD": true, "REVISION": true, "COMPLETED": true,
		"DRAFT": false, "REJECTED": false, "SUBMITED": false, "": false,
		"REJECTED_BY_DIV_HEAD": false, // §5 is not ported
	} {
		assert.Equal(t, want, isActiveRequestStatus(s), s)
	}
}

func TestHaversineAndBox(t *testing.T) {
	d := haversineM(scbdLat, scbdLng, scbdLat+0.001, scbdLng)
	assert.InDelta(t, 111.2, d, 0.5)
	minLat, maxLat, minLng, maxLng := boundingBox(scbdLat, scbdLng, 500)
	assert.Less(t, minLat, scbdLat)
	assert.Greater(t, maxLat, scbdLat)
	assert.Less(t, minLng, scbdLng)
	assert.Greater(t, maxLng, scbdLng)
	assert.Error(t, validatePointInIndonesia(0, 0))
	assert.Error(t, validatePointInIndonesia(40.7, -74))
	assert.NoError(t, validatePointInIndonesia(scbdLat, scbdLng))
}

func TestNearbyActive_FiltersSortsAndNamesOwners(t *testing.T) {
	s, m := newTestTaskService(t)
	m.task.On("GetActiveRequestPoints", ctxAny, mock.Anything, mock.Anything, mock.Anything, mock.Anything, constants.NearbyRequestExtraStatuses).
		Return([]model.RequestPoint{
			{SiteID: "ST3", SiteName1: "Jauh", SiteLat: scbdLat + 0.004, SiteLon: scbdLng, TaskStatus: "WAITING_DEPT_HEAD", CreatedBy: 1},
			{SiteID: "ST2", SiteName1: "Dekat", SiteLat: scbdLat + 0.0005, SiteLon: scbdLng, TaskStatus: "REVISION", CreatedBy: 1},
			{SiteID: "ST1", SiteName1: "Saya", SiteLat: scbdLat, SiteLon: scbdLng, TaskStatus: "WAITING_DIV_HEAD", CreatedBy: 2},
			{SiteID: "ST4", SiteName1: "Draf", SiteLat: scbdLat, SiteLon: scbdLng, TaskStatus: "DRAFT", CreatedBy: 2},
			{SiteID: "ST5", SiteName1: "Nol", TaskStatus: "COMPLETED", CreatedBy: 2},
		}, nil)
	m.user.On("GetByID", ctxAny, int64(1)).Return(&model.User{Username: "andi"}, nil).Once()

	res, err := s.nearbyActive(context.Background(), scbdLat, scbdLng, 500, "ST1")

	require.NoError(t, err)
	assert.Equal(t, []dto.NearbyRequestResponseGet{
		{SiteID: "ST2", SiteName: "Dekat", TaskStatus: "REVISION", OwnerID: 1, OwnerName: "andi", Lat: scbdLat + 0.0005, Lng: scbdLng, DistanceM: 56},
		{SiteID: "ST3", SiteName: "Jauh", TaskStatus: "WAITING_DEPT_HEAD", OwnerID: 1, OwnerName: "andi", Lat: scbdLat + 0.004, Lng: scbdLng, DistanceM: 445},
	}, res)
}

func TestGetNearbyRequests_Limits(t *testing.T) {
	s, _ := newTestTaskService(t)

	_, err := s.GetNearbyRequests(context.Background(), dto.NearbyRequestGet{Lat: scbdLat, Lng: scbdLng, RadiusM: 2001})
	assert.Equal(t, http400("radius maksimal 2000 m"), err.Error())
	_, err = s.GetNearbyRequests(context.Background(), dto.NearbyRequestGet{})
	assert.Contains(t, err.Error(), "Koordinat tidak valid")
}

func http400(msg string) string { return "code=400, message=" + msg }

func TestDecideDuplicate(t *testing.T) {
	assert.Equal(t, duplicateNone, decideDuplicate(constants.DuplicateCheckOff, 3, false))
	assert.Equal(t, duplicateNone, decideDuplicate(constants.DuplicateCheckConfirm, 0, false))
	assert.Equal(t, duplicateBlock, decideDuplicate(constants.DuplicateCheckConfirm, 2, false))
	assert.Equal(t, duplicateConfirm, decideDuplicate(constants.DuplicateCheckConfirm, 2, true))
}

func TestDuplicateConfirmedNotes_MaxLength(t *testing.T) {
	items := make([]dto.NearbyRequestResponseGet, 40)
	for i := range items {
		items[i] = dto.NearbyRequestResponseGet{SiteID: "ST20261021000001", SiteName: "Toko Panjang Sekali", TaskStatus: "WAITING_DEPT_HEAD", DistanceM: 150}
	}
	n := duplicateConfirmedNotes(200, items)
	assert.Len(t, []rune(n), 500)
	assert.True(t, strings.HasSuffix(n, "..."))
	assert.True(t, strings.HasPrefix(n, "Staf tetap mengirim walau ada pengajuan aktif lain dalam 200 m: ST20261021000001 Toko Panjang Sekali (WAITING_DEPT_HEAD, 150 m)"))
}

func TestLoadDuplicateCheckConfig(t *testing.T) {
	assert.Equal(t, duplicateCheckConfig{mode: "off", radiusM: 200}, loadDuplicateCheckConfig(nil))
	v := viper.New()
	v.Set("nearby_request.duplicate_check", " Confirm ")
	v.Set("nearby_request.duplicate_radius_m", 150)
	assert.Equal(t, duplicateCheckConfig{mode: "confirm", radiusM: 150}, loadDuplicateCheckConfig(v))
	v.Set("nearby_request.duplicate_check", "block") // not a mode (open question): stays off
	assert.Equal(t, "off", loadDuplicateCheckConfig(v).mode)
}
